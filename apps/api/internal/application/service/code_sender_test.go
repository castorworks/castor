package service

import (
	"context"
	"errors"
	"testing"
)

// 未配置的投递通道返回明确的业务错误，而不是空指针或连接错误。
func TestCodeSenderUnconfiguredChannel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	production := &codeSender{policy: RuntimePolicy{Development: false}}

	if err := production.send(ctx, ContactTypeMobile, "13800138000", "123456"); !errors.Is(err, ErrDeliveryChannelNotConfigured) {
		t.Fatalf("SMS without a provider: error = %v, want ErrDeliveryChannelNotConfigured", err)
	}
	if err := production.send(ctx, ContactTypeEmail, "user@example.com", "123456"); !errors.Is(err, ErrDeliveryChannelNotConfigured) {
		t.Fatalf("email without a mail server: error = %v, want ErrDeliveryChannelNotConfigured", err)
	}

	// 开发模式照旧跳过短信发送（验证码写日志），不需要配置短信服务。
	development := &codeSender{policy: RuntimePolicy{Development: true}}
	if err := development.send(ctx, ContactTypeMobile, "13800138000", "123456"); err != nil {
		t.Fatalf("development SMS: error = %v, want nil", err)
	}
}
