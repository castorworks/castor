package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/gin-gonic/gin"
)

// ============================================
// Bug Condition Exploration Tests - Login Service (Task 1)
// These tests are EXPECTED TO FAIL on unfixed code.
// Failure confirms the bugs exist.
// ============================================

// TestBugCondition_1i_OTPRegisterBypass verifies that OTP login auto-creates users
// even when registration is disabled (bug 9 exists).
// On unfixed code, this will FAIL because getOrCreateUserForVerificationLogin
// doesn't check RegisterEnabled.
//
// **Validates: Requirements 1.9**
func TestBugCondition_1i_OTPRegisterBypass(t *testing.T) {
	tests := []struct {
		name            string
		registerEnabled bool
		username        string
	}{
		{
			name:            "registration disabled via config",
			registerEnabled: false,
			username:        "newuser@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			userRepo := newMockUserRepo()
			userSvc := &mockUserService{
				users: make(map[string]*user.User),
			}
			mockAuth := &mockAuthService{}

			loginSvc := &LoginService{
				userService: userSvc,
				userRepo:    userRepo,
				rateLimiter: &mockRateLimiter{},
				captcha:     &mockCaptchaClient{valid: true},
				verifyCode:  &mockVerifyCodeClient{valid: true},
				authService: mockAuth,
				policy:      RuntimePolicy{RegisterEnabled: tt.registerEnabled},
			}

			// Create a gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			body := `{"username":"` + tt.username + `","credential":"123456"}`
			c.Request = httptest.NewRequest(http.MethodPost, "/login?method=email", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Call getOrCreateUserForVerificationLogin directly
			_, err := loginSvc.getOrCreateUserForVerificationLogin(c, tt.username)

			// BUG CONDITION: On unfixed code, the user is created even though
			// registration is disabled. The test expects an error.
			if err == nil {
				t.Errorf("Bug 9 confirmed: OTP login created user %q even though RegisterEnabled=%v. "+
					"getOrCreateUserForVerificationLogin should check RegisterEnabled before creating users.",
					tt.username, tt.registerEnabled)
			}

			// Also verify user was actually created (double-check the bug)
			if len(userRepo.created) > 0 {
				t.Errorf("Bug 9 confirmed: user was created in repository despite RegisterEnabled=%v. "+
					"Created users: %d", tt.registerEnabled, len(userRepo.created))
			}
		})
	}
}

// TestBugCondition_1j_OTPHardcodedRateLimit verifies that loginByVerificationCode
// uses hardcoded rate limit values instead of dynamic config (bug 10 exists).
// On unfixed code, this will FAIL because it uses hardcoded `1*time.Minute, 5`.
//
// **Validates: Requirements 1.10**
func TestBugCondition_1j_OTPHardcodedRateLimit(t *testing.T) {
	t.Parallel()

	// Track what rate limit parameters are actually used
	var capturedDuration time.Duration
	var capturedMaxAllowed int64

	trackingRL := &trackingRateLimiterMock{
		onCall: func(key string, duration time.Duration, maxAllowed int64) {
			// Only capture the login rate limit call (not captcha-related)
			if strings.Contains(key, "login:") {
				capturedDuration = duration
				capturedMaxAllowed = maxAllowed
			}
		},
	}

	// Configure dynamic settings with different values than hardcoded
	mockSettings := &mockSettingService{
		settings: map[string]string{
			setting.KeySecurityLoginMaxAttempts: "10",
			setting.KeySecurityLoginLockMinutes: "15",
			setting.KeyFeatureCaptchaEnabled:    "true",
		},
	}

	userSvc := &mockUserService{
		users: map[string]*user.User{
			"existing@example.com": {
				ID:                   1,
				Username:             "existing@example.com",
				Name:                 "Existing User",
				Enable:               true,
				Locked:               false,
				AccountExpireDate:    time.Now().Add(365 * 24 * time.Hour),
				CredentialExpireDate: time.Now().Add(365 * 24 * time.Hour),
			},
		},
	}

	mockAuth := &mockAuthService{}

	loginSvc := &LoginService{
		userService:    userSvc,
		rateLimiter:    trackingRL,
		captcha:        &mockCaptchaClient{valid: true},
		verifyCode:     &mockVerifyCodeClient{valid: true},
		authService:    mockAuth,
		settingService: mockSettings,
	}

	// Create gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"username":"existing@example.com","credential":"123456"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/login?method=email", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call loginByVerificationCode
	req := &LoginReq{
		Username:   "existing@example.com",
		Credential: "123456",
	}
	_, _ = loginSvc.loginByVerificationCode(c, req)

	// BUG CONDITION: On unfixed code, loginByVerificationCode uses hardcoded
	// `1*time.Minute, 5` instead of reading from settingService.
	// The dynamic config has maxAttempts=10, lockMinutes=15.
	hardcodedDuration := 1 * time.Minute
	hardcodedMax := int64(5)

	if capturedDuration == hardcodedDuration && capturedMaxAllowed == hardcodedMax {
		t.Errorf("Bug 10 confirmed: loginByVerificationCode uses hardcoded rate limit (duration=%v, max=%d) "+
			"instead of dynamic config from settingService (expected duration=%v, max=%d). "+
			"Should use getLoginRateLimitConfig() like loginByPassword does.",
			capturedDuration, capturedMaxAllowed,
			15*time.Minute, int64(10))
	}
}
