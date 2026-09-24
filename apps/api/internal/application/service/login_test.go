package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/hyperits/gosuite/security/hash"
)

func TestLoginService_LoginMethod(t *testing.T) {
	t.Run("rejects disabled login method before dispatch", func(t *testing.T) {
		svc := &LoginService{
			settingService: &mockSettingService{
				settings: map[string]string{
					setting.KeySecurityLoginAllowedMethods: `["password"]`,
				},
			},
			loginHistoryService: &mockLoginHistoryService{},
			auditLogService:     &mockAuditLogService{},
		}

		_, err := svc.Login(context.Background(), constant.LOGIN_METHOD_EMAIL, &LoginReq{Username: "user@example.com", Credential: "123456"}, LoginClient{})
		if !errors.Is(err, ErrLoginMethodDisabled) {
			t.Fatalf("expected ErrLoginMethodDisabled, got %v", err)
		}
	})

	t.Run("allows password login when allowed methods setting is missing", func(t *testing.T) {
		passwordHash, err := hash.BcryptHashPassword("secret123")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		u := &user.User{
			ID:                   1,
			Username:             "system",
			Name:                 "System",
			Password:             passwordHash,
			AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
			Enable:               true,
			AccountExpireDate:    time.Now().Add(time.Hour),
			CredentialExpireDate: time.Now().Add(time.Hour),
		}
		svc := &LoginService{
			userService: &mockUserService{
				users: map[string]*user.User{"system": u},
			},
			rateLimiter:         &mockRateLimiter{},
			captcha:             &mockCaptchaClient{valid: true},
			rsaService:          &mockRsaService{},
			settingService:      &mockSettingService{settings: map[string]string{}},
			loginHistoryService: &mockLoginHistoryService{},
			auditLogService:     &mockAuditLogService{},
		}

		got, err := svc.Login(context.Background(), constant.LOGIN_METHOD_PASSWORD, &LoginReq{Username: "system", Credential: "secret123"}, LoginClient{})
		if err != nil {
			t.Fatalf("expected login to succeed, got %v", err)
		}
		if got.Username != "system" {
			t.Fatalf("expected user system, got %s", got.Username)
		}
	})
}

func TestLoginService_VerifyCodeLoginCreatesMissingUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		login    func(*LoginService, context.Context, *LoginReq) (*user.User, error)
	}{
		{
			name:     "email login auto creates missing user",
			username: "new@example.com",
			login:    (*LoginService).loginByEmail,
		},
		{
			name:     "mobile login auto creates missing user",
			username: "13812345678",
			login:    (*LoginService).loginByMobile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()
			authSvc := &mockAuthService{}
			svc := &LoginService{
				userService: &userService{
					userRepo:      userRepo,
					settingHelper: NewSettingHelper(settingRepo),
					rsaService:    &mockRsaService{},
				},
				userRepo:    userRepo,
				rateLimiter: &mockRateLimiter{},
				verifyCode:  &mockVerifyCodeClient{valid: true},
				authService: authSvc,
				settingService: &mockSettingService{
					settings: map[string]string{
						"security.login.maxAttempts": "5",
						"security.login.lockMinutes": "1",
						"feature.captcha.enabled":    "false",
						"feature.register.enabled":   "true",
					},
				},
				captcha: &mockCaptchaClient{valid: true},
				// 注册开关开启时，验证码登录才会为新账号自动建号
				policy: RuntimePolicy{RegisterEnabled: true},
			}

			u, err := tt.login(svc, context.Background(), &LoginReq{
				Username:   tt.username,
				Credential: "123456",
			})

			if err != nil {
				t.Fatalf("expected login to succeed, got %v", err)
			}
			if u.Username != tt.username {
				t.Fatalf("expected username %q, got %q", tt.username, u.Username)
			}
			if !u.Enable || u.Locked {
				t.Fatalf("expected auto-created user to be enabled and unlocked, got enable=%v locked=%v", u.Enable, u.Locked)
			}
			if _, err := userRepo.GetByUsername(context.Background(), tt.username); err != nil {
				t.Fatalf("expected user to be persisted, got %v", err)
			}
			if !authSvc.roleAssigned {
				t.Fatalf("expected default role assignment for %q", tt.username)
			}
		})
	}
}
