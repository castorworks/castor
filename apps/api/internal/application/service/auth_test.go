package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/setting"
)

// ============================================
// Tests
// ============================================

func TestAuthService_Register_Validation(t *testing.T) {
	tests := []struct {
		name         string
		req          *dto.UserRegisterReq
		setupRepo    func(*mockUserRepo)
		setupSetting func(*mockSettingRepo)
		wantErr      error
	}{
		{
			name: "register disabled returns error",
			req: &dto.UserRegisterReq{
				Username:    "newuser",
				Password:    "Password123",
				CaptchaId:   "captcha-id",
				CaptchaCode: "1234",
			},
			setupSetting: func(m *mockSettingRepo) {
				m.settings[setting.KeyFeatureRegisterEnabled] = &setting.Setting{
					Key:   setting.KeyFeatureRegisterEnabled,
					Value: "false",
				}
			},
			wantErr: ErrRegisterNotEnabled,
		},
		{
			name: "password too short returns error",
			req: &dto.UserRegisterReq{
				Username:    "newuser",
				Password:    "123",
				CaptchaId:   "captcha-id",
				CaptchaCode: "1234",
			},
			setupSetting: func(m *mockSettingRepo) {
				m.settings[setting.KeySecurityPasswordMinLength] = &setting.Setting{
					Key:   setting.KeySecurityPasswordMinLength,
					Value: "8",
				}
			},
			wantErr: nil, // will get a password length error (not a sentinel)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			settingRepo := newMockSettingRepo()

			if tt.setupRepo != nil {
				tt.setupRepo(userRepo)
			}
			if tt.setupSetting != nil {
				tt.setupSetting(settingRepo)
			}

			svc := &authService{
				userRepo:      userRepo,
				settingHelper: NewSettingHelper(settingRepo),
				captcha:       &mockCaptchaClient{valid: true},
			}

			_, err := svc.Register(context.Background(), tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				// For non-sentinel errors, just verify an error occurred
				if tt.name == "password too short returns error" && err == nil {
					t.Error("expected a password length error, got nil")
				}
			}
		})
	}
}

func TestAuthService_Register_RequiresValidCaptcha(t *testing.T) {
	userRepo := newMockUserRepo()
	settingRepo := newMockSettingRepo()
	svc := &authService{
		userRepo:      userRepo,
		settingHelper: NewSettingHelper(settingRepo),
		captcha:       &mockCaptchaClient{valid: false},
	}

	_, err := svc.Register(context.Background(), &dto.UserRegisterReq{
		Username:    "newuser",
		Password:    "Password123",
		CaptchaId:   "captcha-id",
		CaptchaCode: "bad",
	})

	if !errors.Is(err, ErrInvalidCaptchaCode) {
		t.Fatalf("expected ErrInvalidCaptchaCode, got %v", err)
	}
}

func TestAuthService_IsRegisterEnabled(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "enabled with true", value: "true", expected: true},
		{name: "enabled with 1", value: "1", expected: true},
		{name: "disabled with false", value: "false", expected: false},
		{name: "disabled with 0", value: "0", expected: false},
		{name: "disabled with empty", value: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settingRepo := newMockSettingRepo()
			settingRepo.settings[setting.KeyFeatureRegisterEnabled] = &setting.Setting{
				Key:   setting.KeyFeatureRegisterEnabled,
				Value: tt.value,
			}

			svc := &authService{
				settingHelper: NewSettingHelper(settingRepo),
			}

			result := svc.settingHelper.IsRegisterEnabled(context.Background(), false)
			if result != tt.expected {
				t.Errorf("isRegisterEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}
