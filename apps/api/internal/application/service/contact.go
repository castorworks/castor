package service

import (
	"context"
	"net/mail"
	"strings"

	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/hyperits/gosuite/logger"
	gomail "github.com/hyperits/gosuite/net/mail"
	"github.com/hyperits/gosuite/net/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
)

// 联系方式类型与登录方式常量共用取值，避免出现第二套 EMAIL/MOBILE 字面量。
const (
	ContactTypeEmail  = constant.LOGIN_METHOD_EMAIL
	ContactTypeMobile = constant.LOGIN_METHOD_MOBILE
)

// normalizeContactType 归一化联系方式类型，接受任意大小写形式（EMAIL/email 等）。
// 第二个返回值为 false 表示类型无法识别。
func normalizeContactType(contactType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(contactType)) {
	case ContactTypeEmail:
		return ContactTypeEmail, true
	case ContactTypeMobile:
		return ContactTypeMobile, true
	default:
		return "", false
	}
}

// validateContact 按类型校验联系方式格式，与 POST /auth/code 使用同一套规则。
func validateContact(contactType, contact string) error {
	switch contactType {
	case ContactTypeEmail:
		if _, err := mail.ParseAddress(contact); err != nil {
			return ErrInvalidEmailFormat
		}
	case ContactTypeMobile:
		if !phoneRegex.MatchString(contact) {
			return ErrInvalidPhoneFormat
		}
	default:
		return ErrInvalidContactType
	}
	return nil
}

// codeSender 把一次性验证码投递到邮箱或手机号，供认证与账户绑定流程共用。
// sms / mail 为 nil 表示部署未配置该通道，投递时返回 ErrDeliveryChannelNotConfigured。
type codeSender struct {
	sms    sms.Sender
	mail   *smtpmail.Client
	policy RuntimePolicy
}

// send 按联系方式类型选择投递通道；未知类型返回 ErrInvalidCodeType。
func (s *codeSender) send(ctx context.Context, contactType, target, code string) error {
	switch contactType {
	case ContactTypeEmail:
		return s.sendEmailCode(ctx, target, code)
	case ContactTypeMobile:
		return s.sendMobileCode(ctx, target, code)
	default:
		return ErrInvalidCodeType
	}
}

func (s *codeSender) sendMobileCode(ctx context.Context, mobile, code string) error {
	if s.policy.Development {
		logger.Warnf("Development mode: skip sms send to %s, code %s", mobile, code)
		return nil
	}
	if s.sms == nil {
		return ErrDeliveryChannelNotConfigured
	}
	return sms.SendCode(ctx, s.sms, mobile, code)
}

func (s *codeSender) sendEmailCode(ctx context.Context, email, code string) error {
	if s.mail == nil {
		return ErrDeliveryChannelNotConfigured
	}
	msg := &gomail.Message{
		From:        s.mail.DefaultFrom(),
		To:          []string{email},
		Subject:     "验证码邮件，请勿泄露!",
		Body:        "<h1>您的验证码为: " + code + "</h1>",
		ContentType: gomail.ContentTypeHTML,
	}
	return s.mail.Send(ctx, msg)
}
