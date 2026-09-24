package external

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/hyperits/gosuite/logger"
	"github.com/redis/go-redis/v9"
)

const (
	verificationCodeKeyPrefix     = "verify_code:"
	verificationAttemptsKeyPrefix = "verify_code_attempts:"
)

// incrAttemptsScript atomically increments the failed-attempt counter and aligns its
// lifetime with the code it protects.
var incrAttemptsScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

// VerificationCodeStore is a Redis-backed one-time code store with a per
// (purpose, target) failed-attempt limit.
type VerificationCodeStore struct {
	rdb         redis.UniversalClient
	keySpace    rediskey.Namespace
	ttl         time.Duration
	length      int
	maxAttempts int64
}

// NewVerificationCodeStore creates the verification code store from configuration.
func NewVerificationCodeStore(rdb redis.UniversalClient, ns rediskey.Namespace) *VerificationCodeStore {
	cfg := config.C.VerifyCode
	return newVerificationCodeStore(rdb, ns, time.Duration(cfg.ExpireSeconds)*time.Second, cfg.Length, cfg.MaxAttempts)
}

func newVerificationCodeStore(rdb redis.UniversalClient, ns rediskey.Namespace, ttl time.Duration, length, maxAttempts int) *VerificationCodeStore {
	if ttl < time.Second {
		ttl = 10 * time.Minute
	}
	if length < 6 || length > 10 {
		length = 6
	}
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	return &VerificationCodeStore{rdb: rdb, keySpace: ns, ttl: ttl, length: length, maxAttempts: int64(maxAttempts)}
}

func (s *VerificationCodeStore) keys(purpose, target string) (string, string) {
	// Hash the target so arbitrary user input never shapes Redis key structure. The hash
	// tag keeps both keys in one cluster slot for the transactional pipeline.
	sum := sha256.Sum256([]byte(purpose + "\x00" + target))
	suffix := "{" + purpose + ":" + hex.EncodeToString(sum[:]) + "}"
	return s.keySpace.Key(verificationCodeKeyPrefix + suffix), s.keySpace.Key(verificationAttemptsKeyPrefix + suffix)
}

// Generate issues a new code and resets the failed-attempt counter.
func (s *VerificationCodeStore) Generate(ctx context.Context, purpose, target string) (string, error) {
	code, err := s.randomCode()
	if err != nil {
		return "", fmt.Errorf("generate verification code: %w", err)
	}
	codeKey, attemptsKey := s.keys(purpose, target)
	if _, err := s.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, codeKey, code, s.ttl)
		pipe.Del(ctx, attemptsKey)
		return nil
	}); err != nil {
		return "", fmt.Errorf("store verification code: %w", err)
	}
	return code, nil
}

// Verify checks a code. Every call counts as an attempt; once the limit is reached the
// active code is deleted and the caller must request a new one.
func (s *VerificationCodeStore) Verify(ctx context.Context, purpose, target, code string) (bool, error) {
	codeKey, attemptsKey := s.keys(purpose, target)

	attempts, err := incrAttemptsScript.Run(ctx, s.rdb, []string{attemptsKey}, int64(s.ttl/time.Second)).Int64()
	if err != nil {
		return false, fmt.Errorf("count verification attempt: %w", err)
	}
	if attempts > s.maxAttempts {
		if err := s.rdb.Del(ctx, codeKey).Err(); err != nil {
			return false, fmt.Errorf("invalidate verification code: %w", err)
		}
		return false, nil
	}

	stored, err := s.rdb.Get(ctx, codeKey).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read verification code: %w", err)
	}

	if code == "" || subtle.ConstantTimeCompare([]byte(stored), []byte(code)) != 1 {
		if attempts >= s.maxAttempts {
			if err := s.rdb.Del(ctx, codeKey).Err(); err != nil {
				return false, fmt.Errorf("invalidate verification code: %w", err)
			}
		}
		return false, nil
	}

	// Claim the code: only one concurrent verifier can delete it.
	deleted, err := s.rdb.Del(ctx, codeKey).Result()
	if err != nil {
		return false, fmt.Errorf("consume verification code: %w", err)
	}
	if deleted == 0 {
		return false, nil
	}
	if err := s.rdb.Del(ctx, attemptsKey).Err(); err != nil {
		// The counter expires with the code TTL; a stale counter only affects this target.
		logger.Warnf("reset verification attempts for purpose %s: %v", purpose, err)
	}
	return true, nil
}

func (s *VerificationCodeStore) randomCode() (string, error) {
	max := big.NewInt(1)
	for i := 0; i < s.length; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", s.length, n), nil
}
