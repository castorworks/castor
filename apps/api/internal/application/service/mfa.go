package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/png"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/mfa"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/castorworks/castor/internal/pkg/secretbox"
	"github.com/hyperits/gosuite/security/hash"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
)

const (
	// totpPeriod TOTP 时间步长（秒），与主流验证器应用一致
	totpPeriod = 30
	// totpSkew 允许前后各一个时间步的时钟偏差
	totpSkew = 1
	// recoveryCodeCount 每次生成的恢复码个数
	recoveryCodeCount = 10
	// mfaChallengeTTL 输对第一重凭证后，提交验证码的期限
	mfaChallengeTTL = 5 * time.Minute
	// mfaChallengeAttempts 一次登录挑战允许的验证码尝试次数
	mfaChallengeAttempts = 5
)

// MFALoginChallenge 输对第一重凭证、等待验证码的登录。存在 Redis 里，令牌是随机的不可猜值。
type MFALoginChallenge struct {
	UserID    uint   `json:"userId"`
	Username  string `json:"username"`
	Method    string `json:"method"`
	Remember  bool   `json:"remember"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

// MFAService 两步验证（TOTP）：绑定、确认、关闭、恢复码，以及登录第二步。
type MFAService interface {
	Status(ctx context.Context, userID uint) (*dto.TOTPStatusResp, error)
	// BeginTOTPSetup 生成新密钥（覆盖尚未确认的旧密钥），返回二维码与手工输入用的密钥
	BeginTOTPSetup(ctx context.Context, u *user.User) (*dto.TOTPSetupResp, error)
	// EnableTOTP 用验证器上的码确认绑定，返回只显示这一次的恢复码
	EnableTOTP(ctx context.Context, userID uint, code string) (*dto.RecoveryCodesResp, error)
	// DisableTOTP 关闭：有密码的账号要验证当前密码（RSA 密文），并验证一次性码或恢复码
	DisableTOTP(ctx context.Context, u *user.User, req *dto.TOTPDisableReq) error
	RegenerateRecoveryCodes(ctx context.Context, userID uint, code string) (*dto.RecoveryCodesResp, error)
	// ResetTOTP 管理员为丢失设备的用户清除两步验证（调用方先做 EnsureCanManageUser）
	ResetTOTP(ctx context.Context, targetID uint) error
	IsEnabled(ctx context.Context, userID uint) (bool, error)
	EnabledAmong(ctx context.Context, userIDs []uint) (map[uint]bool, error)

	// StartLoginChallenge 登录第一步成功且需要验证码时创建挑战，返回挑战令牌
	StartLoginChallenge(ctx context.Context, challenge MFALoginChallenge) (string, error)
	// CompleteLoginChallenge 校验挑战对应用户的验证码；成功后挑战作废，返回它。
	// 挑战不存在、过期或尝试次数用尽返回 ErrMFAChallengeExpired；码错返回 ErrInvalidTOTPCode 与挑战（记录失败用）。
	CompleteLoginChallenge(ctx context.Context, token, code string) (*MFALoginChallenge, error)
}

type mfaService struct {
	repo     mfa.Repository
	users    user.Repository
	settings setting.Repository
	rsa      RsaService
	box      *secretbox.Box
	redis    redis.UniversalClient
	keys     rediskey.Namespace
	now      func() time.Time
}

// NewMFAService 创建两步验证服务
func NewMFAService(repo mfa.Repository, users user.Repository, settings setting.Repository, rsa RsaService,
	keyring *secretbox.Keyring, rdb redis.UniversalClient, ns rediskey.Namespace) (MFAService, error) {
	box, err := keyring.Box("totp")
	if err != nil {
		return nil, err
	}
	return &mfaService{repo: repo, users: users, settings: settings, rsa: rsa, box: box, redis: rdb, keys: ns, now: time.Now}, nil
}

func (s *mfaService) Status(ctx context.Context, userID uint) (*dto.TOTPStatusResp, error) {
	item, err := s.repo.Get(ctx, userID)
	if errors.Is(err, shared.ErrNotFound) {
		return &dto.TOTPStatusResp{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !item.Enabled {
		return &dto.TOTPStatusResp{}, nil
	}
	return &dto.TOTPStatusResp{Enabled: true, EnabledAt: item.EnabledAt, RecoveryCodesRemaining: len(item.RecoveryCodeHashes)}, nil
}

func (s *mfaService) issuer(ctx context.Context) string {
	if item, err := s.settings.GetByKey(ctx, setting.KeySiteName); err == nil && strings.TrimSpace(item.Value) != "" {
		return strings.TrimSpace(item.Value)
	}
	return "Castor"
}

func (s *mfaService) BeginTOTPSetup(ctx context.Context, u *user.User) (*dto.TOTPSetupResp, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: s.issuer(ctx), AccountName: u.Username, Period: totpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	sealed, err := s.box.Seal(key.Secret())
	if err != nil {
		return nil, err
	}
	saved, err := s.repo.SavePending(ctx, u.ID, sealed)
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, apperror.ErrTOTPAlreadyEnabled
	}
	img, err := key.Image(240, 240)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &dto.TOTPSetupResp{
		Secret:     key.Secret(),
		OtpauthURL: key.URL(),
		QRCode:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// matchStep 返回 code 匹配的时间步（允许 ±totpSkew）；不匹配返回 0。
func (s *mfaService) matchStep(secret, code string) int64 {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0
	}
	current := s.now().Unix() / totpPeriod
	for step := current - totpSkew; step <= current+totpSkew; step++ {
		expected, err := totp.GenerateCodeCustom(secret, time.Unix(step*totpPeriod, 0), totp.ValidateOpts{
			Period: totpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return step
		}
	}
	return 0
}

func (s *mfaService) EnableTOTP(ctx context.Context, userID uint, code string) (*dto.RecoveryCodesResp, error) {
	item, err := s.repo.Get(ctx, userID)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, apperror.ErrTOTPSetupNotStarted
	}
	if err != nil {
		return nil, err
	}
	if item.Enabled {
		return nil, apperror.ErrTOTPAlreadyEnabled
	}
	secret, err := s.box.Open(item.SecretCiphertext)
	if err != nil {
		return nil, err
	}
	step := s.matchStep(secret, code)
	if step == 0 {
		return nil, apperror.ErrInvalidTOTPCode
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	enabled, err := s.repo.Enable(ctx, userID, hashes, step)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, apperror.ErrTOTPAlreadyEnabled
	}
	return &dto.RecoveryCodesResp{RecoveryCodes: codes}, nil
}

// verify 校验已启用用户的一次性码或恢复码；两者都是用过即废。
func (s *mfaService) verify(ctx context.Context, userID uint, code string) error {
	item, err := s.repo.Get(ctx, userID)
	if errors.Is(err, shared.ErrNotFound) {
		return apperror.ErrTOTPNotEnabled
	}
	if err != nil {
		return err
	}
	if !item.Enabled {
		return apperror.ErrTOTPNotEnabled
	}
	code = strings.TrimSpace(code)
	if len(code) == 6 {
		secret, err := s.box.Open(item.SecretCiphertext)
		if err != nil {
			return err
		}
		step := s.matchStep(secret, code)
		if step == 0 {
			return apperror.ErrInvalidTOTPCode
		}
		// 条件更新保证同一个码（及更早的码）在任何副本上只能用一次。
		advanced, err := s.repo.AdvanceStep(ctx, userID, step)
		if err != nil {
			return err
		}
		if !advanced {
			return apperror.ErrInvalidTOTPCode
		}
		return nil
	}
	consumed, err := s.repo.ConsumeRecoveryCode(ctx, userID, hashRecoveryCode(code))
	if err != nil {
		return err
	}
	if !consumed {
		return apperror.ErrInvalidTOTPCode
	}
	return nil
}

func (s *mfaService) DisableTOTP(ctx context.Context, u *user.User, req *dto.TOTPDisableReq) error {
	// 只有 OIDC 登录、从未设过密码的账号没有密码可验证；其余一律要当前密码。
	if u.Password != "" {
		password, err := s.rsa.Decrypt(ctx, req.Password)
		if err != nil {
			return ErrPasswordDecryptFailed
		}
		if !hash.BcryptMatchPassword(password, u.Password) {
			return ErrCurrentPasswordIncorrect
		}
	}
	if err := s.verify(ctx, u.ID, req.Code); err != nil {
		return err
	}
	return s.repo.Delete(ctx, u.ID)
}

func (s *mfaService) RegenerateRecoveryCodes(ctx context.Context, userID uint, code string) (*dto.RecoveryCodesResp, error) {
	if err := s.verify(ctx, userID, code); err != nil {
		return nil, err
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return &dto.RecoveryCodesResp{RecoveryCodes: codes}, nil
}

func (s *mfaService) ResetTOTP(ctx context.Context, targetID uint) error {
	return s.repo.Delete(ctx, targetID)
}

func (s *mfaService) IsEnabled(ctx context.Context, userID uint) (bool, error) {
	enabled, err := s.repo.EnabledAmong(ctx, []uint{userID})
	return enabled[userID], err
}

func (s *mfaService) EnabledAmong(ctx context.Context, userIDs []uint) (map[uint]bool, error) {
	return s.repo.EnabledAmong(ctx, userIDs)
}

func (s *mfaService) challengeKey(token string) string {
	return s.keys.Key("mfa:challenge:" + token)
}

func (s *mfaService) attemptsKey(token string) string {
	return s.keys.Key("mfa:challenge-attempts:" + token)
}

func (s *mfaService) StartLoginChallenge(ctx context.Context, challenge MFALoginChallenge) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	payload, err := json.Marshal(challenge)
	if err != nil {
		return "", err
	}
	if err := s.redis.Set(ctx, s.challengeKey(token), payload, mfaChallengeTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (s *mfaService) CompleteLoginChallenge(ctx context.Context, token, code string) (*MFALoginChallenge, error) {
	if token == "" {
		return nil, apperror.ErrMFAChallengeExpired
	}
	payload, err := s.redis.Get(ctx, s.challengeKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, apperror.ErrMFAChallengeExpired
	}
	if err != nil {
		return nil, err
	}
	var challenge MFALoginChallenge
	if err := json.Unmarshal(payload, &challenge); err != nil {
		return nil, apperror.ErrMFAChallengeExpired
	}
	// 尝试次数独立计数：并发提交也不会绕过上限。
	attempts, err := s.redis.Incr(ctx, s.attemptsKey(token)).Result()
	if err != nil {
		return nil, err
	}
	if attempts == 1 {
		s.redis.Expire(ctx, s.attemptsKey(token), mfaChallengeTTL)
	}
	if attempts > mfaChallengeAttempts {
		s.redis.Del(ctx, s.challengeKey(token), s.attemptsKey(token))
		return &challenge, apperror.ErrMFAChallengeExpired
	}
	if err := s.verify(ctx, challenge.UserID, code); err != nil {
		if errors.Is(err, apperror.ErrTOTPNotEnabled) {
			// 等待期间两步验证被重置：这次挑战不再成立。
			s.redis.Del(ctx, s.challengeKey(token), s.attemptsKey(token))
			return &challenge, apperror.ErrMFAChallengeExpired
		}
		return &challenge, err
	}
	// 只有删掉挑战的那一次请求可以继续登录，同一挑战不会换出两个会话。
	deleted, err := s.redis.Del(ctx, s.challengeKey(token)).Result()
	if err != nil {
		return nil, err
	}
	s.redis.Del(ctx, s.attemptsKey(token))
	if deleted == 0 {
		return &challenge, apperror.ErrMFAChallengeExpired
	}
	return &challenge, nil
}

// recoveryCodeAlphabet 恢复码只用小写字母与数字，便于抄写
var recoveryCodeEncoding = base32.NewEncoding("abcdefghijkmnpqrstuvwxyz23456789").WithPadding(base32.NoPadding)

// newRecoveryCodes 生成恢复码（形如 abcd-efgh-jkmn，60 位随机性）及其哈希
func newRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, recoveryCodeCount)
	hashes := make([]string, recoveryCodeCount)
	for i := range codes {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, err
		}
		raw := recoveryCodeEncoding.EncodeToString(buf)[:12]
		codes[i] = raw[:4] + "-" + raw[4:8] + "-" + raw[8:12]
		hashes[i] = hashRecoveryCode(codes[i])
	}
	return codes, hashes, nil
}

// hashRecoveryCode 忽略大小写、空格与连字符：用户照抄时的格式差异不影响匹配。
// 恢复码有 60 位随机性，直接用 SHA-256 存储即可（不需要慢哈希）。
func hashRecoveryCode(code string) string {
	normalized := strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, strings.ToLower(strings.TrimSpace(code)))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
