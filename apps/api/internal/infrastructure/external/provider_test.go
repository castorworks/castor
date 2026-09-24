package external

import (
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/config"
)

// 短信与邮件是可选的验证码通道：部署没配置它们时服务必须照常启动
// （gosuite v1 会拒绝空的 AccessKey / Host，不能把空配置交给它创建客户端）。
func TestUnconfiguredDeliveryChannelsDoNotBlockStartup(t *testing.T) {
	// Not parallel: mutates config.C.
	original := config.C
	t.Cleanup(func() { config.C = original })
	config.C = new(config.Config)

	sender, err := NewAliyunSmsClient()
	if err != nil || sender != nil {
		t.Fatalf("NewAliyunSmsClient() without keys = %v, %v; want nil, nil", sender, err)
	}
	mail, err := NewMailClient()
	if err != nil || mail != nil {
		t.Fatalf("NewMailClient() without host = %v, %v; want nil, nil", mail, err)
	}
}

// 只填了一半的短信配置是运维失误，应在启动时暴露，而不是等到发送时才失败。
func TestPartialSmsConfigFailsStartup(t *testing.T) {
	original := config.C
	t.Cleanup(func() { config.C = original })

	for name, keys := range map[string][2]string{
		"access key only": {"access-key", ""},
		"secret key only": {"", "secret-key"},
	} {
		config.C = new(config.Config)
		config.C.AliyunSms.AccessKey, config.C.AliyunSms.SecretKey = keys[0], keys[1]
		if _, err := NewAliyunSmsClient(); err == nil {
			t.Errorf("%s: expected a configuration error", name)
		}
	}

	config.C = new(config.Config)
	config.C.AliyunSms.AccessKey, config.C.AliyunSms.SecretKey = "access-key", "secret-key"
	config.C.AliyunSms.SignName, config.C.AliyunSms.TemplateCode = "Castor", "SMS_0001"
	if sender, err := NewAliyunSmsClient(); err != nil || sender == nil {
		t.Fatalf("configured SMS: NewAliyunSmsClient() = %v, %v", sender, err)
	}
}
