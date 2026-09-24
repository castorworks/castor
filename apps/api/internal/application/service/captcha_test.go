package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
)

// 图形验证码是防刷的闸门：答案不对是用户的问题，存储故障是我们的问题，两者不能混为一谈，
// 更不能因为存储不可用就放行。
func TestCheckCaptcha(t *testing.T) {
	ctx := context.Background()
	storeDown := errors.New("redis: connection refused")

	if err := checkCaptcha(ctx, &mockCaptchaClient{valid: true}, "id", "1234"); err != nil {
		t.Fatalf("correct answer: err = %v", err)
	}
	if err := checkCaptcha(ctx, &mockCaptchaClient{valid: false}, "id", "0000"); !errors.Is(err, ErrInvalidCaptchaCode) {
		t.Fatalf("wrong answer: err = %v, want ErrInvalidCaptchaCode", err)
	}

	err := checkCaptcha(ctx, &mockCaptchaClient{valid: true, err: storeDown}, "id", "1234")
	if !errors.Is(err, storeDown) {
		t.Fatalf("store failure: err = %v, want it to wrap the store error", err)
	}
	if errors.Is(err, ErrInvalidCaptchaCode) {
		t.Fatal("a store failure must not be reported as a wrong answer")
	}
}

// 限流触发后发码必须过验证码；验证码存储故障时不得照常发码。
func TestAuthService_PostCode_FailsClosedWhenCaptchaStoreIsDown(t *testing.T) {
	storeDown := errors.New("redis: connection refused")
	verifyCode := &mockVerifyCodeClient{valid: true}
	svc := &authService{
		rateLimiter: &mockRateLimiter{blocked: true},
		captcha:     &mockCaptchaClient{valid: true, err: storeDown},
		verifyCode:  verifyCode,
		policy:      RuntimePolicy{Development: false},
	}

	err := svc.PostCode(context.Background(),
		&dto.ConfirmCodeReq{CodeType: "email", Username: "alice@example.com"}, "203.0.113.5")
	if !errors.Is(err, storeDown) {
		t.Fatalf("err = %v, want the captcha store failure", err)
	}
}
