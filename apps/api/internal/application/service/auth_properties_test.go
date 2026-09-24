package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"testing/quick"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/hyperits/gosuite/net/sms"
)

// stubSmsSender 记录是否发送过短信，可注入发送错误
type stubSmsSender struct {
	called   bool
	lastCode string
	err      error
}

// Send 实现 sms.Sender；验证码经 sms.SendCode 以模板参数 code 送达
func (m *stubSmsSender) Send(_ context.Context, msg *sms.Message) (*sms.SendResult, error) {
	m.called = true
	m.lastCode = msg.TemplateParam["code"]
	return &sms.SendResult{}, m.err
}

// mapSettingService 以内存 map 实现 SettingService
type mapSettingService struct {
	values map[string]string
}

func newMapSettingService() *mapSettingService {
	return &mapSettingService{
		values: map[string]string{
			setting.KeySecurityLoginMaxAttempts:  "5",
			setting.KeySecurityLoginLockMinutes:  "30",
			setting.KeyFeatureCaptchaEnabled:     "true",
			setting.KeyFeatureRegisterEnabled:    "false",
			setting.KeySecurityPasswordMinLength: "8",
		},
	}
}

func (m *mapSettingService) Gets(_ context.Context, _ string) ([]dto.SettingResp, error) {
	return nil, nil
}
func (m *mapSettingService) Get(_ context.Context, _ uint) (*dto.SettingResp, error) {
	return nil, nil
}
func (m *mapSettingService) GetByKey(_ context.Context, _ string) (*dto.SettingResp, error) {
	return nil, nil
}
func (m *mapSettingService) Post(_ context.Context, _ *dto.SettingPostReq) (*dto.SettingResp, error) {
	return nil, nil
}
func (m *mapSettingService) Put(_ context.Context, _ uint, _ *dto.SettingPutReq) (*dto.SettingResp, error) {
	return nil, nil
}
func (m *mapSettingService) Delete(_ context.Context, _ uint) error { return nil }
func (m *mapSettingService) BatchUpdate(_ context.Context, _ *dto.SettingBatchUpdateReq) error {
	return nil
}
func (m *mapSettingService) GetPublicSettings(_ context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mapSettingService) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := m.values[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}
func (m *mapSettingService) GetBool(_ context.Context, key string) (bool, error) {
	v, ok := m.values[key]
	if !ok {
		return false, fmt.Errorf("key not found: %s", key)
	}
	return v == "true" || v == "1", nil
}
func (m *mapSettingService) GetInt(_ context.Context, key string) (int, error) {
	v, ok := m.values[key]
	if !ok {
		return 0, fmt.Errorf("key not found: %s", key)
	}
	var i int
	_, err := fmt.Sscanf(v, "%d", &i)
	return i, err
}

// randomValidEmail generates a random valid email address
func randomValidEmail(rng *rand.Rand) string {
	localParts := []string{"user", "admin", "test", "info", "hello", "support"}
	domains := []string{"example.com", "test.org", "mail.io", "company.net"}
	local := localParts[rng.IntN(len(localParts))]
	suffix := fmt.Sprintf("%d", rng.IntN(9999))
	domain := domains[rng.IntN(len(domains))]
	return local + suffix + "@" + domain
}

// randomValidPhone generates a random valid phone number
func randomValidPhone(rng *rand.Rand) string {
	prefixes := []string{"138", "139", "150", "151", "186", "187", "188", "199"}
	prefix := prefixes[rng.IntN(len(prefixes))]
	return prefix + fmt.Sprintf("%08d", rng.IntN(100000000))
}

func seededRng(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed+1)))
}

// TestVerificationLoginReturnsExistingUser：对任意已存在、启用、未锁定、
// 未过期的用户，验证码正确时 loginByVerificationCode 返回该用户且无错误。
func TestVerificationLoginReturnsExistingUser(t *testing.T) {
	t.Parallel()

	f := func(seed int64) bool {
		rng := seededRng(seed)
		username := randomValidEmail(rng)

		userRepo := newMockUserRepo()
		settingRepo := newMockSettingRepo()

		// Create an existing valid user
		existingUser := &user.User{
			ID:                   uint(rng.IntN(10000) + 1),
			Username:             username,
			Name:                 username,
			Password:             "hashed",
			AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
			Enable:               true,
			Locked:               false,
			AccountExpireDate:    time.Now().Add(365 * 24 * time.Hour),
			CredentialExpireDate: time.Now().Add(365 * 24 * time.Hour),
		}
		userRepo.users[username] = existingUser

		authSvc := &mockAuthService{}
		svc := &LoginService{
			userService: &userService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				rsaService:    &mockRsaService{},
			},
			userRepo:       userRepo,
			rateLimiter:    &mockRateLimiter{},
			captcha:        &mockCaptchaClient{valid: true},
			verifyCode:     &mockVerifyCodeClient{valid: true},
			authService:    authSvc,
			settingService: newMapSettingService(),
		}

		c := newTestContext()
		u, err := svc.loginByVerificationCode(c, &LoginReq{
			Username:   username,
			Credential: "123456",
		})

		// Property: login succeeds and returns the correct user
		if err != nil {
			return false
		}
		if u == nil || u.Username != username {
			return false
		}
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatalf("verification login of an existing user did not return that user: %v", err)
	}
}

// TestPostCodeAcceptsValidEmail：任意合法邮箱都通过格式校验并生成验证码，
// 一直走到投递这一步（此处邮件未配置，因而以 ErrDeliveryChannelNotConfigured 结束）。
func TestPostCodeAcceptsValidEmail(t *testing.T) {
	t.Parallel()

	f := func(seed int64) bool {
		rng := seededRng(seed)
		email := randomValidEmail(rng)

		settingRepo := newMockSettingRepo()
		svc := &authService{
			userRepo:      newMockUserRepo(),
			settingHelper: NewSettingHelper(settingRepo),
			captcha:       &mockCaptchaClient{valid: true},
			rateLimiter:   &mockRateLimiter{},
			verifyCode:    &mockVerifyCodeClient{valid: true},
			codeSender:    &codeSender{sms: &stubSmsSender{}, mail: nil}, // 未配置邮件：投递是最后一步
		}

		req := &dto.ConfirmCodeReq{
			CodeType: constant.LOGIN_METHOD_EMAIL,
			Username: email,
		}

		// 校验全部通过、验证码已生成，才会走到投递并因邮件未配置而失败。
		err := svc.PostCode(context.Background(), req, "203.0.113.9")
		if !errors.Is(err, ErrDeliveryChannelNotConfigured) {
			t.Logf("email %q: PostCode() error = %v, want ErrDeliveryChannelNotConfigured", email, err)
			return false
		}
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatalf("valid email code request was rejected before delivery: %v", err)
	}
}

// TestPostCodeAcceptsValidMobile：任意合法手机号都通过格式校验，PostCode 返回 nil。
func TestPostCodeAcceptsValidMobile(t *testing.T) {
	t.Parallel()

	f := func(seed int64) bool {
		rng := seededRng(seed)
		phone := randomValidPhone(rng)

		settingRepo := newMockSettingRepo()
		smsMock := &stubSmsSender{}
		svc := &authService{
			userRepo:      newMockUserRepo(),
			settingHelper: NewSettingHelper(settingRepo),
			captcha:       &mockCaptchaClient{valid: true},
			rateLimiter:   &mockRateLimiter{},
			verifyCode:    &mockVerifyCodeClient{valid: true},
			codeSender:    &codeSender{sms: smsMock},
		}

		req := &dto.ConfirmCodeReq{
			CodeType: constant.LOGIN_METHOD_MOBILE,
			Username: phone,
		}

		err := svc.PostCode(context.Background(), req, "203.0.113.9")

		// Property: no error for valid phone numbers (validation passes)
		if err != nil {
			return false
		}
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatalf("valid mobile code request failed: %v", err)
	}
}

// TestSendMobileCodeSendsSMSOutsideDevelopment：非开发模式下 sendMobileCode
// 总是调用短信发送器且不返回错误（开发模式跳过发送，见 auth_hardening_test.go）。
func TestSendMobileCodeSendsSMSOutsideDevelopment(t *testing.T) {

	f := func(seed int64) bool {
		rng := seededRng(seed)
		phone := randomValidPhone(rng)
		code := fmt.Sprintf("%06d", rng.IntN(1000000))

		smsMock := &stubSmsSender{}
		svc := &authService{
			codeSender: &codeSender{sms: smsMock, policy: RuntimePolicy{Development: false}},
			policy:     RuntimePolicy{Development: false},
		}

		err := svc.codeSender.sendMobileCode(context.Background(), phone, code)

		// Property: SMS sender is called and no error
		if err != nil {
			return false
		}
		if !smsMock.called {
			return false
		}
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatalf("sendMobileCode outside development did not send the SMS: %v", err)
	}
}

// TestVerificationLoginSignsInExistingUserWithoutRegistering：无论注册开关如何，
// 已存在且状态正常的用户都能用验证码登录，且不会触发建号。
func TestVerificationLoginSignsInExistingUserWithoutRegistering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		registerEnabled bool
	}{
		{"register enabled", true},
		{"register disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := func(seed int64) bool {
				rng := seededRng(seed)
				username := randomValidPhone(rng)

				userRepo := newMockUserRepo()
				settingRepo := newMockSettingRepo()
				if !tt.registerEnabled {
					settingRepo.settings[setting.KeyFeatureRegisterEnabled] = &setting.Setting{
						Key:   setting.KeyFeatureRegisterEnabled,
						Value: "false",
					}
				}

				// Pre-create the user (existing user)
				existingUser := &user.User{
					ID:                   uint(rng.IntN(10000) + 1),
					Username:             username,
					Name:                 username,
					Password:             "hashed",
					AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
					Enable:               true,
					Locked:               false,
					AccountExpireDate:    time.Now().Add(365 * 24 * time.Hour),
					CredentialExpireDate: time.Now().Add(365 * 24 * time.Hour),
				}
				userRepo.users[username] = existingUser

				authSvc := &mockAuthService{}
				svc := &LoginService{
					userService: &userService{
						userRepo:      userRepo,
						settingHelper: NewSettingHelper(settingRepo),
						rsaService:    &mockRsaService{},
					},
					userRepo:       userRepo,
					rateLimiter:    &mockRateLimiter{},
					captcha:        &mockCaptchaClient{valid: true},
					verifyCode:     &mockVerifyCodeClient{valid: true},
					authService:    authSvc,
					settingService: newMapSettingService(),
				}

				c := newTestContext()
				u, err := svc.loginByVerificationCode(c, &LoginReq{
					Username:   username,
					Credential: "123456",
				})

				// Property: login succeeds
				if err != nil {
					return false
				}
				if u == nil || u.Username != username {
					return false
				}
				// Property: no account creation attempted
				// (roleAssigned stays false for existing users)
				if authSvc.roleAssigned {
					return false
				}
				return true
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
				t.Fatalf("(%s) verification login of an existing user failed or registered a new account: %v", tt.name, err)
			}
		})
	}
}

// TestLoginRateLimitConfigReadsSettings：getLoginRateLimitConfig 返回系统设置中
// 配置的最大失败次数与锁定分钟数，运维调整后无需发版即可生效。
func TestLoginRateLimitConfigReadsSettings(t *testing.T) {
	t.Parallel()

	f := func(maxAttempts uint8, lockMinutes uint8) bool {
		// Ensure positive values (avoid zero which triggers fallback)
		ma := int(maxAttempts)%20 + 1
		lm := int(lockMinutes)%60 + 1

		settingSvc := newMapSettingService()
		settingSvc.values[setting.KeySecurityLoginMaxAttempts] = fmt.Sprintf("%d", ma)
		settingSvc.values[setting.KeySecurityLoginLockMinutes] = fmt.Sprintf("%d", lm)

		svc := &LoginService{
			settingService: settingSvc,
		}

		c := newTestContext()
		gotMax, gotLock := svc.getLoginRateLimitConfig(c)

		// Property: values come from setting service
		if gotMax != ma {
			return false
		}
		if gotLock != lm {
			return false
		}
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatalf("login rate limit config did not come from the setting service: %v", err)
	}
}
