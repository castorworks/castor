package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/hyperits/gosuite/logger"
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
	PostCode(ctx context.Context, req *dto.ConfirmCodeReq, clientIP string) error
	Register(ctx context.Context, req *dto.UserRegisterReq) (*dto.UserResp, error)
	// AssignDefaultRole 为用户分配默认角色
	AssignDefaultRole(ctx context.Context, username string) error
}

type captchaVerifier interface {
	Verify(ctx context.Context, id string, digits string) (bool, error)
}

type captchaProvider interface {
	captchaVerifier
	Generate(ctx context.Context) (id string, imageBase64 string, err error)
}

// checkCaptcha 校验图形验证码（一次性：无论对错都会被消费）。答案不对返回 ErrInvalidCaptchaCode；
// 存储故障原样上抛——验证码是防刷的闸门，不能因为 Redis 不可用就放行，也不该谎称用户输错了。
func checkCaptcha(ctx context.Context, verifier captchaVerifier, id, digits string) error {
	ok, err := verifier.Verify(ctx, id, digits)
	if err != nil {
		return fmt.Errorf("verify captcha: %w", err)
	}
	if !ok {
		return ErrInvalidCaptchaCode
	}
	return nil
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
	codeSender    *codeSender
	rbac          RBACService
	rsaService    RsaService
	policy        RuntimePolicy
}

// NewAuthService 创建认证应用服务
func NewAuthService(
	userRepo user.Repository,
	settingHelper *SettingHelper,
	captcha *captcha.Client,
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
		codeSender:    &codeSender{sms: sms, mail: mail, policy: policy},
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

	passwordPolicy := svc.settingHelper.PasswordPolicy(ctx)
	if err := passwordPolicy.Validate(plainPassword); err != nil {
		return nil, err
	}

	if err := checkCaptcha(ctx, svc.captcha, req.CaptchaId, req.CaptchaCode); err != nil {
		return nil, err
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
		Username:             req.Username,
		Name:                 displayName,
		Password:             encPassword,
		AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
		CredentialExpireDate: passwordPolicy.CredentialExpiry(time.Now()),
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

func (svc *authService) PostCode(ctx context.Context, req *dto.ConfirmCodeReq, clientIP string) error {
	// 客户端传入的 codeType/purpose 均大小写不敏感；purpose 缺省为 auth。
	// 验证码按 (purpose, target) 隔离存放，登录验证码因此无法被重放去重置密码。
	// 在限流之前先校验参数，避免无效请求消耗配额。
	codeType := normalizeCodeType(req.CodeType)
	purpose, ok := NormalizeVerificationPurpose(req.Purpose)
	if !ok {
		return ErrInvalidCodeType
	}

	// 非开发环境，执行限制。同时按收件人和客户端 IP 计数：单个 IP 轮换收件人
	// 无法再规避验证码门槛（防止把服务器当作未认证的邮件/短信中继）。
	if !svc.policy.Development {
		targetLimited, err := svc.rateLimiter.IsRequestReachLimit(ctx, "code:target:"+req.Username, 1*time.Minute, 2)
		if err != nil {
			logger.Errorf("Rate limit check error for code request: target=%s, err=%v", req.Username, err)
			return err
		}
		ipLimited, err := svc.rateLimiter.IsRequestReachLimit(ctx, "code:ip:"+clientIP, 1*time.Minute, 2)
		if err != nil {
			logger.Errorf("Rate limit check error for code request: ip=%s, err=%v", clientIP, err)
			return err
		}
		if targetLimited || ipLimited {
			if err := checkCaptcha(ctx, svc.captcha, req.CaptchaId, req.CaptchaCode); err != nil {
				return err
			}
		}
	}

	// Validate email/phone format before attempting to send
	switch codeType {
	case ContactTypeEmail, ContactTypeMobile:
		if err := validateContact(codeType, req.Username); err != nil {
			return err
		}
	default:
		return ErrInvalidCodeType
	}

	// 不查库、不区分账号是否存在：无论目标是否已注册都照常下发，
	// 避免把本接口变成账号枚举器。
	code, err := svc.verifyCode.Generate(ctx, purpose, req.Username)
	if err != nil {
		return err
	}
	return svc.codeSender.send(ctx, codeType, req.Username, code)
}

func (svc *authService) GetCaptcha(ctx context.Context) (*dto.CaptchaResp, error) {
	id, b64, err := svc.captcha.Generate(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Generated captcha: id=%s", id)

	return &dto.CaptchaResp{
		ID:  id,
		IMG: b64,
	}, nil
}

// normalizeCodeType 归一化验证码类型，接受任意大小写形式（EMAIL/email 等）。
// 返回值为内部常量形式；无法识别时原样返回，由调用方给出 ErrInvalidCodeType。
func normalizeCodeType(codeType string) string {
	return strings.ToLower(strings.TrimSpace(codeType))
}
