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

// TestVerificationLoginDoesNotRegisterWhenRegistrationDisabled 确认注册关闭时，
// 验证码登录不会为不存在的账号自动建号，否则关闭注册形同虚设。
func TestVerificationLoginDoesNotRegisterWhenRegistrationDisabled(t *testing.T) {
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

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			body := `{"username":"` + tt.username + `","credential":"123456"}`
			c.Request = httptest.NewRequest(http.MethodPost, "/login?method=email", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			_, err := loginSvc.getOrCreateUserForVerificationLogin(c, tt.username)

			if err == nil {
				t.Errorf("verification login for unknown user %q succeeded with RegisterEnabled=%v; want an error",
					tt.username, tt.registerEnabled)
			}

			if len(userRepo.created) > 0 {
				t.Errorf("verification login created %d user(s) with RegisterEnabled=%v; want none",
					len(userRepo.created), tt.registerEnabled)
			}
		})
	}
}

// TestVerificationLoginUsesConfiguredRateLimit 确认验证码登录的失败次数限制
// 读取系统设置（最大次数、锁定分钟数），与密码登录一致，而不是固定值。
func TestVerificationLoginUsesConfiguredRateLimit(t *testing.T) {
	t.Parallel()

	// 记录登录限流实际使用的参数
	var capturedDuration time.Duration
	var capturedMaxAllowed int64

	trackingRL := &trackingRateLimiterMock{
		onCall: func(key string, duration time.Duration, maxAllowed int64) {
			// 只记录登录限流（忽略验证码相关的调用）
			if strings.Contains(key, "login:") {
				capturedDuration = duration
				capturedMaxAllowed = maxAllowed
			}
		},
	}

	// 设置值与代码内的兜底值不同，便于区分
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

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"username":"existing@example.com","credential":"123456"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/login?method=email", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	req := &LoginReq{
		Username:   "existing@example.com",
		Credential: "123456",
	}
	_, _ = loginSvc.loginByVerificationCode(c, req)

	// 固定值 1 分钟 / 5 次说明设置被忽略；设置为 maxAttempts=10、lockMinutes=15
	hardcodedDuration := 1 * time.Minute
	hardcodedMax := int64(5)

	if capturedDuration == hardcodedDuration && capturedMaxAllowed == hardcodedMax {
		t.Errorf("verification login rate limit used duration=%v, max=%d instead of the configured "+
			"duration=%v, max=%d from the setting service",
			capturedDuration, capturedMaxAllowed,
			15*time.Minute, int64(10))
	}
}
