package service

import (
	"context"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/creasty/defaults"
)

// ============================================
// Bug Condition Exploration Tests (Task 1)
// These tests are EXPECTED TO FAIL on unfixed code.
// Failure confirms the bugs exist.
// ============================================

// bcSmsSender tracks whether SendCode was called (bug condition variant)
type bcSmsSender struct {
	called bool
	mobile string
	code   string
}

func (m *bcSmsSender) Send(_ context.Context, _, _, _ string) error {
	m.called = true
	return nil
}

func (m *bcSmsSender) SendCode(_ context.Context, mobile, code string) error {
	m.called = true
	m.mobile = mobile
	m.code = code
	return nil
}

// bcVerifyCodeTracker tracks whether GenerateCode was called
type bcVerifyCodeTracker struct {
	generateCalled bool
	generateKey    string
}

func (m *bcVerifyCodeTracker) Generate(_ context.Context, _, key string) (string, error) {
	m.generateCalled = true
	m.generateKey = key
	return "123456", nil
}

func (m *bcVerifyCodeTracker) Verify(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

// TestBugCondition_1c_JWTDefaultTimeouts verifies that the default config has
// sensible JWT timeout values where JwtMaxRefreshHours > JwtTimeoutHours.
// On unfixed code (both defaults were 12), this FAILS because refresh is useless.
// On fixed code (defaults are 1 and 168), this PASSES.
//
// **Validates: Requirements 1.3**
func TestBugCondition_1c_JWTDefaultTimeouts(t *testing.T) {
	t.Parallel()

	// Use the actual default values from the struct tags in config.go:
	// JwtTimeoutHours `default:"1"` and JwtMaxRefreshHours `default:"168"`
	var cfg config.General
	defaults.Set(&cfg)

	// BUG CONDITION: MaxRefresh should be GREATER than Timeout for refresh to be useful.
	// With both at 12 (old defaults), gin-jwt measures MaxRefresh from token issue time,
	// so the refresh window expires exactly when the token does - making refresh impossible.
	if cfg.JwtMaxRefreshHours <= cfg.JwtTimeoutHours {
		t.Errorf("Bug 3 confirmed: JwtMaxRefreshHours (%d) <= JwtTimeoutHours (%d). "+
			"Token refresh is useless because MaxRefresh window expires when token does. "+
			"Expected JwtMaxRefreshHours > JwtTimeoutHours.",
			cfg.JwtMaxRefreshHours, cfg.JwtTimeoutHours)
	}
}

// TestBugCondition_1e_EmailValidationMissing verifies that PostCode accepts
// invalid email addresses without validation (bug 5 exists).
// On unfixed code, this will FAIL because no email format validation exists.
//
// **Validates: Requirements 1.5**
func TestBugCondition_1e_EmailValidationMissing(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{name: "plain string", username: "not-an-email"},
		{name: "missing at sign", username: "userexample.com"},
		{name: "spaces in address", username: "user @example.com"},
		{name: "no domain part", username: "user@"},
		{name: "just numbers", username: "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smsMock := &bcSmsSender{}
			verifyTracker := &bcVerifyCodeTracker{}
			svc := &authService{
				sms:         smsMock,
				verifyCode:  verifyTracker,
				rateLimiter: &mockRateLimiter{},
				captcha:     &mockCaptchaClient{valid: true},
				policy:      RuntimePolicy{Development: true}, // 开发模式跳过频率限制
			}

			req := &dto.ConfirmCodeReq{
				CodeType: "email", // lowercase to match constant.LOGIN_METHOD_EMAIL
				Username: tt.username,
			}

			// Use recover to catch panic from nil mail client
			// If PostCode doesn't validate email and tries to send, it will panic on nil svc.mail
			var panicOccurred bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicOccurred = true
					}
				}()
				_ = svc.PostCode(context.Background(), req)
			}()

			// BUG CONDITION: On unfixed code, PostCode does NOT validate email format.
			// It proceeds to GenerateCode and then tries to send email (panics on nil mail).
			// The test expects a validation error BEFORE reaching GenerateCode.
			if verifyTracker.generateCalled || panicOccurred {
				t.Errorf("Bug 5 confirmed: PostCode proceeded to generate/send code for invalid email %q "+
					"without validation. Expected validation error before attempting send. "+
					"GenerateCode called: %v, Panic (nil mail): %v",
					tt.username, verifyTracker.generateCalled, panicOccurred)
			}
		})
	}
}

// TestBugCondition_1f_PhoneValidationMissing verifies that PostCode accepts
// invalid phone numbers without validation (bug 6 exists).
// On unfixed code, this will FAIL because no phone format validation exists.
//
// **Validates: Requirements 1.6**
func TestBugCondition_1f_PhoneValidationMissing(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{name: "alphabetic string", username: "abcdef"},
		{name: "random text", username: "hello-world"},
		{name: "email address", username: "user@example.com"},
		{name: "special chars", username: "!@#$%^"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smsMock := &bcSmsSender{}
			verifyTracker := &bcVerifyCodeTracker{}
			svc := &authService{
				sms:         smsMock,
				verifyCode:  verifyTracker,
				rateLimiter: &mockRateLimiter{},
				captcha:     &mockCaptchaClient{valid: true},
				policy:      RuntimePolicy{Development: true}, // 开发模式跳过频率限制
			}

			req := &dto.ConfirmCodeReq{
				CodeType: "mobile", // lowercase to match constant.LOGIN_METHOD_MOBILE
				Username: tt.username,
			}

			_ = svc.PostCode(context.Background(), req)

			// BUG CONDITION: On unfixed code, PostCode does NOT validate phone format.
			// It proceeds to GenerateCode and then calls sendMobileCode without validation.
			// The test expects a validation error BEFORE reaching GenerateCode.
			if verifyTracker.generateCalled {
				t.Errorf("Bug 6 confirmed: PostCode proceeded to generate/send code for invalid phone %q "+
					"without validation. Expected validation error before attempting send. "+
					"SMS sender called: %v", tt.username, smsMock.called)
			}
		})
	}
}

// TestBugCondition_1h_DevModeSMSSent verifies that sendMobileCode still calls
// the SMS sender in development mode (bug 8 exists).
// On unfixed code, this will FAIL because the function doesn't return early.
//
// **Validates: Requirements 1.8**
func TestBugCondition_1h_DevModeSMSSent(t *testing.T) {
	smsMock := &bcSmsSender{}
	svc := &authService{
		sms:    smsMock,
		policy: RuntimePolicy{Development: true},
	}

	// Call sendMobileCode
	err := svc.sendMobileCode(context.Background(), "13800138000", "123456")

	// BUG CONDITION: On unfixed code, sendMobileCode logs a warning but still
	// calls svc.sms.SendCode(). The test expects SMS sender is NOT called.
	if err != nil {
		t.Fatalf("sendMobileCode returned unexpected error: %v", err)
	}

	if smsMock.called {
		t.Errorf("Bug 8 confirmed: SMS sender was called in development mode. "+
			"sendMobileCode should return nil immediately after logging skip warning, "+
			"but it still calls svc.sms.SendCode(). Mobile: %s, Code: %s",
			smsMock.mobile, smsMock.code)
	}
}
