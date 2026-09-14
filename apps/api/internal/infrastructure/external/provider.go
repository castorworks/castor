package external

import (
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/net/sms"
	aliyunsms "github.com/hyperits/gosuite/providers/aliyun/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/hyperits/gosuite/security/captcha"
	"github.com/redis/go-redis/v9"
)

// NewCaptchaClient 创建验证码客户端
func NewCaptchaClient(rdb redis.UniversalClient) *captcha.CaptchaClient {
	captchaRedisStore := captcha.NewCaptchaRedisStore(rdb, config.C.Captcha.ExpireSeconds)
	captchaClient := captcha.NewCaptchaClient(
		captchaRedisStore,
		config.C.Captcha.Length,
		config.C.Captcha.Width,
		config.C.Captcha.Height,
	)
	return captchaClient
}

// NewMailClient 创建邮件客户端
func NewMailClient() *smtpmail.Client {
	return smtpmail.NewClient(&config.C.Mail)
}

// NewAliyunSmsClient 创建阿里云短信客户端
func NewAliyunSmsClient() (sms.Sender, error) {
	return aliyunsms.NewClient(&config.C.AliyunSms)
}
