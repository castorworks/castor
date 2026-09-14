package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/hyperits/gosuite/logger"
	gomail "github.com/hyperits/gosuite/net/mail"
	"github.com/hyperits/gosuite/net/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/hyperits/gosuite/security/captcha"
	"github.com/hyperits/gosuite/security/hash"
)

// 默认用户角色
const DefaultUserRole = "user"

// phoneRegex matches E.164 format (+1234567890) and common national formats (e.g., 13800138000)
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)

// AuthService 认证应用服务接口
type AuthService interface {
	GetCaptcha(ctx context.Context) (*dto.CaptchaResp, error)
	PostCode(ctx context.Context, req *dto.ConfirmCodeReq) error
	Register(ctx context.Context, req *dto.UserRegisterReq) (*dto.UserResp, error)
	// AssignDefaultRole 为用户分配默认角色
	AssignDefaultRole(ctx context.Context, username string) error
}

type captchaVerifier interface {
	VerifyCaptcha(captchaId string, digits string) bool
}

type captchaProvider interface {
	captchaVerifier
	GetCaptcha() (captchaId string, imageData string, err error)
}

type requestRateLimiter interface {
	IsRequestReachLimit(ctx context.Context, key string, duration time.Duration, maxAllowed int64) (bool, error)
}

type authService struct {
	userRepo      user.Repository
	settingHelper *SettingHelper // 共享的系统配置读取辅助工具
	captcha       captchaProvider
	rateLimiter   requestRateLimiter
	verifyCode    VerificationCodeStore
	sms           sms.Sender
	mail          *smtpmail.Client
	rbac          RBACService
	rsaService    RsaService
	policy        RuntimePolicy
}

// NewAuthService 创建认证应用服务
func NewAuthService(
	userRepo user.Repository,
	settingHelper *SettingHelper,
	captcha *captcha.CaptchaClient,
	rateLimiter *ratelimit.RateLimiter,
	verifyCode VerificationCodeStore,
	sms sms.Sender,
	mail *smtpmail.Client,
	rbac RBACService,
	rsaService RsaService,
	policy RuntimePolicy,
) AuthService {
	return &authService{
		userRepo:      userRepo,
		settingHelper: settingHelper,
		captcha:       captcha,
		rateLimiter:   rateLimiter,
		verifyCode:    verifyCode,
		sms:           sms,
		mail:          mail,
		rbac:          rbac,
		rsaService:    rsaService,
		policy:        policy,
	}
}

func (svc *authService) Register(ctx context.Context, req *dto.UserRegisterReq) (*dto.UserResp, error) {
	plainPassword := req.Password
	if svc.rsaService != nil {
		decryptedPassword, err := svc.rsaService.Decrypt(ctx, req.Password)
		if err != nil {
			return nil, ErrPasswordDecryptFailed
		}
		plainPassword = decryptedPassword
	}

	// 从系统配置获取注册开关，如果获取失败则使用 config.toml 中的配置
	if !svc.settingHelper.IsRegisterEnabled(ctx, svc.policy.RegisterEnabled) {
		return nil, ErrRegisterNotEnabled
	}

	// 从系统配置获取密码最小长度并校验
	if err := svc.settingHelper.ValidatePasswordLength(ctx, plainPassword); err != nil {
		return nil, err
	}

	if !svc.captcha.VerifyCaptcha(req.CaptchaId, req.CaptchaCode) {
		return nil, ErrInvalidCaptchaCode
	}

	// 检查用户是否已存在
	_, err := svc.userRepo.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, ErrUsernameAlreadyTaken
	}
	if !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	// Determine display name: use trimmed Name if non-empty, otherwise fall back to Username
	displayName := strings.TrimSpace(req.Name)
	if displayName == "" {
		displayName = req.Username
	}

	// Validate trimmed name length
	if len(displayName) > 100 {
		return nil, ErrNameTooLong
	}

	encPassword, err := hash.BcryptHashPassword(plainPassword)
	if err != nil {
		return nil, err
	}

	newUser := user.User{
		Username:      req.Username,
		Name:          displayName,
		Password:      encPassword,
		AccountSource: constant.ACCOUNT_SOURCE_INTERNAL,
	}

	if err := svc.userRepo.Create(ctx, &newUser); err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}
	if err := svc.AssignDefaultRole(ctx, req.Username); err != nil {
		if delErr := svc.userRepo.Delete(ctx, newUser.ID); delErr != nil {
			logger.Errorf("Compensating delete of user %d after role assignment failure failed: %v", newUser.ID, delErr)
		}
		return nil, fmt.Errorf("assign default role: %w", err)
	}

	resp := &dto.UserResp{}
	if err := resp.FromEntity(&newUser); err != nil {
		return nil, err
	}

	return resp, nil
}

// AssignDefaultRole 为用户分配默认角色
func (svc *authService) AssignDefaultRole(ctx context.Context, username string) error {
	return svc.rbac.AddUserRole(ctx, username, DefaultUserRole, true)
}

func (svc *authService) PostCode(ctx context.Context, req *dto.ConfirmCodeReq) error {
	// 非开发环境，执行限制
	if !svc.policy.Development {
		required, err := svc.rateLimiter.IsRequestReachLimit(ctx, fmt.Sprintf("%s:%s", "code", req.Username), 1*time.Minute, 2)
		if err != nil {
			logger.Errorf("Rate limit check error for code request: username=%s, err=%v", req.Username, err)
			return err
		}

		if required {
			if !svc.captcha.VerifyCaptcha(req.CaptchaId, req.CaptchaCode) {
				return ErrInvalidCaptchaCode
			}
		}
	}

	var (
		err  error
		code string
	)

	// Validate email/phone format before attempting to send
	switch req.CodeType {
	case constant.LOGIN_METHOD_EMAIL:
		if _, parseErr := mail.ParseAddress(req.Username); parseErr != nil {
			return ErrInvalidEmailFormat
		}
	case constant.LOGIN_METHOD_MOBILE:
		if !phoneRegex.MatchString(req.Username) {
			return ErrInvalidPhoneFormat
		}
	}

	switch req.CodeType {
	case constant.LOGIN_METHOD_EMAIL:
		code, err = svc.verifyCode.Generate(ctx, VerificationPurposeAuth, req.Username)
		if err != nil {
			return err
		}
		err = svc.sendEmailCode(ctx, req.Username, code)
	case constant.LOGIN_METHOD_MOBILE:
		code, err = svc.verifyCode.Generate(ctx, VerificationPurposeAuth, req.Username)
		if err != nil {
			return err
		}
		err = svc.sendMobileCode(ctx, req.Username, code)
	default:
		err = ErrInvalidCodeType
	}

	return err
}

func (svc *authService) GetCaptcha(ctx context.Context) (*dto.CaptchaResp, error) {
	id, b64, err := svc.captcha.GetCaptcha()
	if err != nil {
		return nil, err
	}

	logger.Debugf("Generated captcha: id=%s", id)

	return &dto.CaptchaResp{
		ID:  id,
		IMG: b64,
	}, nil
}

func (svc *authService) sendMobileCode(ctx context.Context, mobile, code string) error {
	if svc.policy.Development {
		logger.Warnf("Development mode: skip sms send to %s, code %s", mobile, code)
		return nil
	}
	return svc.sms.SendCode(ctx, mobile, code)
}

func (svc *authService) sendEmailCode(ctx context.Context, email, code string) error {
	msg := &gomail.Message{
		From:        svc.mail.DefaultFrom(),
		To:          []string{email},
		Subject:     "验证码邮件，请勿泄露!",
		Body:        "<h1>您的验证码为: " + code + "</h1>",
		ContentType: gomail.ContentTypeHTML,
	}
	return svc.mail.Send(ctx, msg)
}
