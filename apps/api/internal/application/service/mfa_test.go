package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/mfa"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/castorworks/castor/internal/pkg/secretbox"
	"github.com/hyperits/gosuite/security/hash"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// memMFARepo 是 mfa.Repository 的内存实现
type memMFARepo struct {
	mu    sync.Mutex
	items map[uint]*mfa.TOTP
}

func newMemMFARepo() *memMFARepo { return &memMFARepo{items: map[uint]*mfa.TOTP{}} }

func (r *memMFARepo) Get(_ context.Context, userID uint) (*mfa.TOTP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[userID]
	if !ok {
		return nil, shared.ErrNotFound
	}
	copied := *item
	copied.RecoveryCodeHashes = slices.Clone(item.RecoveryCodeHashes)
	return &copied, nil
}

func (r *memMFARepo) SavePending(_ context.Context, userID uint, secret string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.items[userID]; ok && item.Enabled {
		return false, nil
	}
	r.items[userID] = &mfa.TOTP{UserID: userID, SecretCiphertext: secret}
	return true, nil
}

func (r *memMFARepo) Enable(_ context.Context, userID uint, hashes []string, step int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[userID]
	if !ok || item.Enabled {
		return false, nil
	}
	now := time.Now()
	item.Enabled, item.RecoveryCodeHashes, item.LastUsedStep, item.EnabledAt = true, hashes, step, &now
	return true, nil
}

func (r *memMFARepo) Delete(_ context.Context, userID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, userID)
	return nil
}

func (r *memMFARepo) AdvanceStep(_ context.Context, userID uint, step int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[userID]
	if !ok || !item.Enabled || item.LastUsedStep >= step {
		return false, nil
	}
	item.LastUsedStep = step
	return true, nil
}

func (r *memMFARepo) ConsumeRecoveryCode(_ context.Context, userID uint, h string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[userID]
	if !ok || !item.Enabled {
		return false, nil
	}
	i := slices.Index(item.RecoveryCodeHashes, h)
	if i < 0 {
		return false, nil
	}
	item.RecoveryCodeHashes = slices.Delete(item.RecoveryCodeHashes, i, i+1)
	return true, nil
}

func (r *memMFARepo) ReplaceRecoveryCodes(_ context.Context, userID uint, hashes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.items[userID]; ok {
		item.RecoveryCodeHashes = hashes
	}
	return nil
}

func (r *memMFARepo) EnabledAmong(_ context.Context, userIDs []uint) (map[uint]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[uint]bool{}
	for _, id := range userIDs {
		if item, ok := r.items[id]; ok && item.Enabled {
			out[id] = true
		}
	}
	return out, nil
}

type mfaFixture struct {
	svc   *mfaService
	repo  *memMFARepo
	clock *time.Time
	user  *user.User
}

func newMFAFixture(t *testing.T) *mfaFixture {
	t.Helper()
	_, rdb := newJobTestRedis(t)
	ring, _ := secretbox.NewKeyring(strings.Repeat("k", 32))
	repo := newMemMFARepo()
	password, err := hash.BcryptHashPassword("Current-Passw0rd")
	if err != nil {
		t.Fatal(err)
	}
	svcAny, err := NewMFAService(repo, newMockUserRepo(), newMockSettingRepo(), &mockRsaService{}, ring, rdb, rediskey.Namespace("test"))
	if err != nil {
		t.Fatal(err)
	}
	svc := svcAny.(*mfaService)
	clock := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return clock }
	return &mfaFixture{svc: svc, repo: repo, clock: &clock, user: &user.User{ID: 7, Username: "alice", Password: password}}
}

// code 返回 fixture 当前时间（偏移 offset 个时间步）的一次性码
func (f *mfaFixture) code(t *testing.T, secret string, offset int) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, f.clock.Add(time.Duration(offset)*totpPeriod*time.Second), totp.ValidateOpts{
		Period: totpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func (f *mfaFixture) advance(steps int) {
	*f.clock = f.clock.Add(time.Duration(steps) * totpPeriod * time.Second)
}

// enable 走完绑定流程，返回密钥与恢复码
func (f *mfaFixture) enable(t *testing.T) (string, []string) {
	t.Helper()
	ctx := context.Background()
	setup, err := f.svc.BeginTOTPSetup(ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := f.svc.EnableTOTP(ctx, f.user.ID, f.code(t, setup.Secret, 0))
	if err != nil {
		t.Fatal(err)
	}
	return setup.Secret, codes.RecoveryCodes
}

func TestMFA_SetupAndEnable(t *testing.T) {
	f := newMFAFixture(t)
	ctx := context.Background()

	if _, err := f.svc.EnableTOTP(ctx, f.user.ID, "123456"); !errors.Is(err, apperror.ErrTOTPSetupNotStarted) {
		t.Fatalf("enable before setup err = %v", err)
	}
	setup, err := f.svc.BeginTOTPSetup(ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(setup.QRCode, "data:image/png;base64,") || !strings.Contains(setup.OtpauthURL, "otpauth://totp/") ||
		!strings.Contains(setup.OtpauthURL, "alice") || len(setup.Secret) < 16 {
		t.Fatalf("setup = %+v", setup)
	}
	stored := f.repo.items[f.user.ID]
	if strings.Contains(stored.SecretCiphertext, setup.Secret) || !strings.HasPrefix(stored.SecretCiphertext, "v1:") {
		t.Fatal("the secret must be stored encrypted")
	}
	if status, _ := f.svc.Status(ctx, f.user.ID); status.Enabled {
		t.Fatal("an unconfirmed setup must not count as enabled")
	}
	if _, err := f.svc.EnableTOTP(ctx, f.user.ID, "000000"); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("wrong code err = %v", err)
	}
	codes, err := f.svc.EnableTOTP(ctx, f.user.ID, f.code(t, setup.Secret, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(codes.RecoveryCodes) != recoveryCodeCount || !strings.Contains(codes.RecoveryCodes[0], "-") {
		t.Fatalf("recovery codes = %v", codes.RecoveryCodes)
	}
	for _, h := range f.repo.items[f.user.ID].RecoveryCodeHashes {
		if slices.Contains(codes.RecoveryCodes, h) {
			t.Fatal("recovery codes must be stored hashed")
		}
	}
	status, _ := f.svc.Status(ctx, f.user.ID)
	if !status.Enabled || status.RecoveryCodesRemaining != recoveryCodeCount {
		t.Fatalf("status = %+v", status)
	}
	if _, err := f.svc.BeginTOTPSetup(ctx, f.user); !errors.Is(err, apperror.ErrTOTPAlreadyEnabled) {
		t.Fatalf("setup while enabled err = %v; it must not replace the active secret", err)
	}
}

func TestMFA_LoginChallenge(t *testing.T) {
	f := newMFAFixture(t)
	ctx := context.Background()
	secret, recovery := f.enable(t)
	f.advance(2)
	start := func() string {
		t.Helper()
		token, err := f.svc.StartLoginChallenge(ctx, MFALoginChallenge{UserID: f.user.ID, Username: "alice", Method: "password", Remember: true})
		if err != nil || len(token) < 40 {
			t.Fatalf("StartLoginChallenge = %q, %v", token, err)
		}
		return token
	}

	token := start()
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, "000000"); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("wrong code err = %v", err)
	}
	challenge, err := f.svc.CompleteLoginChallenge(ctx, token, f.code(t, secret, 0))
	if err != nil || challenge.UserID != f.user.ID || !challenge.Remember || challenge.Method != "password" {
		t.Fatalf("CompleteLoginChallenge = %+v, %v", challenge, err)
	}
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, f.code(t, secret, 0)); !errors.Is(err, apperror.ErrMFAChallengeExpired) {
		t.Fatalf("a challenge must be single-use, err = %v", err)
	}

	// 同一个一次性码不能在另一次登录里重放；时钟偏差内的上一个码也不行。
	replay := start()
	if _, err := f.svc.CompleteLoginChallenge(ctx, replay, f.code(t, secret, 0)); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("replayed code err = %v", err)
	}
	if _, err := f.svc.CompleteLoginChallenge(ctx, replay, f.code(t, secret, -1)); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("an older code than the last used one must be rejected, err = %v", err)
	}
	if _, err := f.svc.CompleteLoginChallenge(ctx, replay, f.code(t, secret, 1)); err != nil {
		t.Fatalf("the next step's code (clock skew) must be accepted: %v", err)
	}

	// 恢复码：格式宽松，只能用一次。
	token = start()
	sloppy := strings.ToUpper(strings.ReplaceAll(recovery[0], "-", " "))
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, sloppy); err != nil {
		t.Fatalf("recovery code: %v", err)
	}
	token = start()
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, recovery[0]); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("a recovery code must be single-use, err = %v", err)
	}
	if status, _ := f.svc.Status(ctx, f.user.ID); status.RecoveryCodesRemaining != recoveryCodeCount-1 {
		t.Fatalf("remaining = %d", status.RecoveryCodesRemaining)
	}

	// 尝试次数用尽后挑战作废，正确的码也不再有效。
	token = start()
	for i := 1; i < mfaChallengeAttempts; i++ {
		if _, err := f.svc.CompleteLoginChallenge(ctx, token, "000000"); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
			t.Fatalf("attempt %d err = %v", i, err)
		}
	}
	f.advance(5)
	f.svc.CompleteLoginChallenge(ctx, token, "000000")
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, f.code(t, secret, 0)); !errors.Is(err, apperror.ErrMFAChallengeExpired) {
		t.Fatalf("challenge must be gone after %d attempts, err = %v", mfaChallengeAttempts, err)
	}
	if _, err := f.svc.CompleteLoginChallenge(ctx, "made-up", "123456"); !errors.Is(err, apperror.ErrMFAChallengeExpired) {
		t.Fatalf("unknown challenge err = %v", err)
	}

	// 等待期间管理员重置了两步验证：挑战失效，不能靠它登录。
	token = start()
	_ = f.svc.ResetTOTP(ctx, f.user.ID)
	if _, err := f.svc.CompleteLoginChallenge(ctx, token, "123456"); !errors.Is(err, apperror.ErrMFAChallengeExpired) {
		t.Fatalf("challenge after reset err = %v", err)
	}
}

func TestMFA_DisableAndRegenerate(t *testing.T) {
	f := newMFAFixture(t)
	ctx := context.Background()
	secret, recovery := f.enable(t)
	f.advance(2)

	if _, err := f.svc.RegenerateRecoveryCodes(ctx, f.user.ID, "000000"); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("regenerate with wrong code err = %v", err)
	}
	fresh, err := f.svc.RegenerateRecoveryCodes(ctx, f.user.ID, recovery[1])
	if err != nil || len(fresh.RecoveryCodes) != recoveryCodeCount {
		t.Fatalf("regenerate = %+v, %v", fresh, err)
	}
	if err := f.svc.DisableTOTP(ctx, f.user, &dto.TOTPDisableReq{Password: "Current-Passw0rd", Code: recovery[2]}); !errors.Is(err, apperror.ErrInvalidTOTPCode) {
		t.Fatalf("old recovery codes must stop working after regeneration, err = %v", err)
	}
	if err := f.svc.DisableTOTP(ctx, f.user, &dto.TOTPDisableReq{Password: "wrong", Code: f.code(t, secret, 0)}); !errors.Is(err, ErrCurrentPasswordIncorrect) {
		t.Fatalf("disable with wrong password err = %v", err)
	}
	if err := f.svc.DisableTOTP(ctx, f.user, &dto.TOTPDisableReq{Password: "Current-Passw0rd", Code: fresh.RecoveryCodes[0]}); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := f.svc.IsEnabled(ctx, f.user.ID); enabled {
		t.Fatal("two-factor must be off after disabling")
	}
	if err := f.svc.DisableTOTP(ctx, f.user, &dto.TOTPDisableReq{Password: "Current-Passw0rd", Code: "123456"}); !errors.Is(err, apperror.ErrTOTPNotEnabled) {
		t.Fatalf("disable when off err = %v", err)
	}
}
