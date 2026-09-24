package service

import (
	"context"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/creasty/defaults"
	"github.com/hyperits/gosuite/net/sms"
)

// recordingSmsSender 记录是否发送过短信及其目标与验证码
type recordingSmsSender struct {
	called bool
	mobile string
	code   string
}

// Send 实现 sms.Sender；验证码经 sms.SendCode 以模板参数 code 送达
func (m *recordingSmsSender) Send(_ context.Context, msg *sms.Message) (*sms.SendResult, error) {
	m.called = true
	m.mobile = msg.Mobile
	m.code = msg.TemplateParam["code"]
	return &sms.SendResult{}, nil
}

// codeGenerationTracker 记录是否生成过验证码
type codeGenerationTracker struct {
	generateCalled bool
	generateKey    string
}

func (m *codeGenerationTracker) Generate(_ context.Context, _, key string) (string, error) {
	m.generateCalled = true
	m.generateKey = key
	return "123456", nil
}

func (m *codeGenerationTracker) Verify(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

// TestDefaultJWTRefreshWindowExceedsAccessTokenLifetime 确认默认配置的刷新窗口
// （JwtMaxRefreshHours）长于访问令牌有效期（JwtTimeoutHours）。gin-jwt 从签发时刻
// 起算刷新窗口，两者相等时令牌一过期就无法刷新。
func TestDefaultJWTRefreshWindowExceedsAccessTokenLifetime(t *testing.T) {
	t.Parallel()

	// 取 config.go 结构体标签中的实际默认值
	var cfg config.General
	defaults.Set(&cfg)

	if cfg.JwtMaxRefreshHours <= cfg.JwtTimeoutHours {
		t.Errorf("default JwtMaxRefreshHours (%d) <= JwtTimeoutHours (%d); "+
			"the refresh window must outlast the access token, otherwise refresh is impossible",
			cfg.JwtMaxRefreshHours, cfg.JwtTimeoutHours)
	}
}

// TestPostCodeRejectsInvalidEmailBeforeGeneratingCode 确认邮件验证码请求先校验
// 邮箱格式：格式非法时不生成验证码、不尝试投递。
func TestPostCodeRejectsInvalidEmailBeforeGeneratingCode(t *testing.T) {
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

			smsMock := &recordingSmsSender{}
			verifyTracker := &codeGenerationTracker{}
			svc := &authService{
				codeSender:  &codeSender{sms: smsMock, policy: RuntimePolicy{Development: true}},
				verifyCode:  verifyTracker,
				rateLimiter: &mockRateLimiter{},
				captcha:     &mockCaptchaClient{valid: true},
				policy:      RuntimePolicy{Development: true}, // 开发模式跳过频率限制
			}

			req := &dto.ConfirmCodeReq{
				CodeType: "email", // lowercase to match constant.LOGIN_METHOD_EMAIL
				Username: tt.username,
			}

			// 未配置邮件客户端：若越过校验去投递会 panic，用 recover 捕获
			var panicOccurred bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicOccurred = true
					}
				}()
				_ = svc.PostCode(context.Background(), req, "203.0.113.9")
			}()

			if verifyTracker.generateCalled || panicOccurred {
				t.Errorf("PostCode generated or sent a code for invalid email %q instead of rejecting it; "+
					"GenerateCode called: %v, delivery attempted (nil mail panic): %v",
					tt.username, verifyTracker.generateCalled, panicOccurred)
			}
		})
	}
}

// TestPostCodeRejectsInvalidMobileBeforeGeneratingCode 确认短信验证码请求先校验
// 手机号格式：格式非法时不生成验证码、不发送短信。
func TestPostCodeRejectsInvalidMobileBeforeGeneratingCode(t *testing.T) {
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

			smsMock := &recordingSmsSender{}
			verifyTracker := &codeGenerationTracker{}
			svc := &authService{
				codeSender:  &codeSender{sms: smsMock, policy: RuntimePolicy{Development: true}},
				verifyCode:  verifyTracker,
				rateLimiter: &mockRateLimiter{},
				captcha:     &mockCaptchaClient{valid: true},
				policy:      RuntimePolicy{Development: true}, // 开发模式跳过频率限制
			}

			req := &dto.ConfirmCodeReq{
				CodeType: "mobile", // lowercase to match constant.LOGIN_METHOD_MOBILE
				Username: tt.username,
			}

			_ = svc.PostCode(context.Background(), req, "203.0.113.9")

			if verifyTracker.generateCalled {
				t.Errorf("PostCode generated a code for invalid mobile %q instead of rejecting it; "+
					"SMS sender called: %v", tt.username, smsMock.called)
			}
		})
	}
}

// TestSendMobileCodeSkipsSMSInDevelopment 确认开发模式下不真正发送短信
// （只记录跳过日志），避免本地环境产生费用或打扰真实号码。
func TestSendMobileCodeSkipsSMSInDevelopment(t *testing.T) {
	smsMock := &recordingSmsSender{}
	svc := &authService{
		codeSender: &codeSender{sms: smsMock, policy: RuntimePolicy{Development: true}},
		policy:     RuntimePolicy{Development: true},
	}

	err := svc.codeSender.sendMobileCode(context.Background(), "13800138000", "123456")

	if err != nil {
		t.Fatalf("sendMobileCode returned unexpected error: %v", err)
	}

	if smsMock.called {
		t.Errorf("SMS sender was called in development mode (mobile %s, code %s); "+
			"sendMobileCode must skip sending in development",
			smsMock.mobile, smsMock.code)
	}
}
