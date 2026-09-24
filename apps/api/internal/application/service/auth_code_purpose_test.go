package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
)

// recordingCodeStore 记录 Generate 收到的用途与目标。
type recordingCodeStore struct {
	purposes []string
	targets  []string
}

func (s *recordingCodeStore) Generate(_ context.Context, purpose, target string) (string, error) {
	s.purposes = append(s.purposes, purpose)
	s.targets = append(s.targets, target)
	return "123456", nil
}

func (s *recordingCodeStore) Verify(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func newPostCodeFixture() (*authService, *recordingCodeStore) {
	store := &recordingCodeStore{}
	svc := &authService{
		userRepo:      newMockUserRepo(),
		settingHelper: NewSettingHelper(newMockSettingRepo()),
		captcha:       &mockCaptchaClient{valid: true},
		rateLimiter:   &mockRateLimiter{},
		verifyCode:    store,
		// 开发模式下短信通道直接短路，不需要真实 sender。
		codeSender: &codeSender{policy: RuntimePolicy{Development: true}},
		policy:     RuntimePolicy{Development: true},
	}
	return svc, store
}

func TestAuthService_PostCodePurpose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		purpose string
		want    string
	}{
		{name: "absent defaults to auth", purpose: "", want: VerificationPurposeAuth},
		{name: "explicit auth", purpose: "auth", want: VerificationPurposeAuth},
		{name: "explicit reset", purpose: "reset", want: VerificationPurposeReset},
		{name: "uppercase reset", purpose: "RESET", want: VerificationPurposeReset},
		{name: "padded mixed case", purpose: " Reset ", want: VerificationPurposeReset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, store := newPostCodeFixture()
			err := svc.PostCode(context.Background(), &dto.ConfirmCodeReq{
				CodeType: "MOBILE", Username: "13800138000", Purpose: tt.purpose,
			}, "203.0.113.9")
			if err != nil {
				t.Fatalf("PostCode() error = %v", err)
			}
			if len(store.purposes) != 1 || store.purposes[0] != tt.want {
				t.Fatalf("generated purposes = %v, want [%s]", store.purposes, tt.want)
			}
		})
	}
}

func TestAuthService_PostCodeRejectsUnknownPurpose(t *testing.T) {
	t.Parallel()

	// bind 用途由服务端绑定流程独占，客户端不得自行索取。
	for _, purpose := range []string{"bind", "whatever", "auth;reset"} {
		svc, store := newPostCodeFixture()
		err := svc.PostCode(context.Background(), &dto.ConfirmCodeReq{
			CodeType: "MOBILE", Username: "13800138000", Purpose: purpose,
		}, "203.0.113.9")
		if !errors.Is(err, ErrInvalidCodeType) {
			t.Fatalf("purpose %q: error = %v, want ErrInvalidCodeType", purpose, err)
		}
		if len(store.purposes) != 0 {
			t.Fatalf("purpose %q: no code should be issued, got %v", purpose, store.purposes)
		}
	}
}

// TestAuthService_PostCodeDoesNotRevealAccountExistence 保留非枚举属性：
// 目标账号不存在时也照常下发验证码。
func TestAuthService_PostCodeDoesNotRevealAccountExistence(t *testing.T) {
	t.Parallel()
	svc, store := newPostCodeFixture()
	// 用手机号通道：开发模式下短信直接短路，无需真实发送器。
	err := svc.PostCode(context.Background(), &dto.ConfirmCodeReq{
		CodeType: "mobile", Username: "13900139000", Purpose: "reset",
	}, "203.0.113.9")
	if err != nil {
		t.Fatalf("PostCode() error = %v", err)
	}
	if len(store.targets) != 1 || store.targets[0] != "13900139000" {
		t.Fatalf("generated targets = %v, want [13900139000]", store.targets)
	}
	if len(store.purposes) != 1 || store.purposes[0] != VerificationPurposeReset {
		t.Fatalf("generated purposes = %v, want [reset]", store.purposes)
	}
}

func TestNormalizeVerificationPurpose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{in: "", want: VerificationPurposeAuth, ok: true},
		{in: "  ", want: VerificationPurposeAuth, ok: true},
		{in: "AUTH", want: VerificationPurposeAuth, ok: true},
		{in: "Reset", want: VerificationPurposeReset, ok: true},
		{in: "bind", ok: false},
		{in: "nope", ok: false},
	}
	for _, tt := range tests {
		got, ok := NormalizeVerificationPurpose(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("NormalizeVerificationPurpose(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
