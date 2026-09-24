package external

import (
	"time"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/hyperits/gosuite/logger"
	"github.com/hyperits/gosuite/net/sms"
	aliyunsms "github.com/hyperits/gosuite/providers/aliyun/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/hyperits/gosuite/security/captcha"
	"github.com/redis/go-redis/v9"
)

// NewCaptchaClient 创建图形验证码客户端（答案存 Redis，一次性）
func NewCaptchaClient(rdb redis.UniversalClient, ns rediskey.Namespace) (*captcha.Client, error) {
	store, err := captcha.NewRedisStore(rdb, time.Duration(config.C.Captcha.ExpireSeconds)*time.Second)
	if err != nil {
		return nil, err
	}
	return captcha.NewClient(store.WithKeyPrefix(ns.Key(captcha.DefaultKeyPrefix)), captcha.Options{
		Length: config.C.Captcha.Length,
		Width:  config.C.Captcha.Width,
		Height: config.C.Captcha.Height,
	})
}

// NewMailClient 创建邮件客户端。未配置 Mail.Host 时返回 nil：邮件是可选的验证码通道，
// 不应阻止服务启动；向该通道发送验证码时返回 ErrDeliveryChannelNotConfigured。
func NewMailClient() (*smtpmail.Client, error) {
	if config.C.Mail.Host == "" {
		logger.Infof("Mail is not configured; email verification codes are unavailable")
		return nil, nil
	}
	return smtpmail.NewClient(&config.C.Mail)
}

// NewAliyunSmsClient 创建阿里云短信客户端。AccessKey 与 SecretKey 都未配置时返回 nil（同邮件）；
// 只填了其中一个视为配置错误，照常由 gosuite 校验拒绝启动。
func NewAliyunSmsClient() (sms.Sender, error) {
	if config.C.AliyunSms.AccessKey == "" && config.C.AliyunSms.SecretKey == "" {
		logger.Infof("Aliyun SMS is not configured; SMS verification codes are unavailable")
		return nil, nil
	}
	return aliyunsms.NewClient(&config.C.AliyunSms)
}
