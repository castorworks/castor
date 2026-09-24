package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/generator"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/hyperits/gosuite/logger"
	"github.com/hyperits/gosuite/security/captcha"
	"github.com/hyperits/gosuite/security/hash"
)

// 登录速率限制默认配置（当无法从数据库获取配置时使用）
const (
	DefaultLoginRateLimitWindow   = 5 * time.Minute // 默认速率限制时间窗口
	DefaultLoginRateLimitMaxTries = 5               // 默认最大尝试次数
)

// LoginReq 登录请求
type LoginReq struct {
	Username    string `json:"username" binding:"required"`
	Credential  string `json:"credential" binding:"required"`
	CaptchaId   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

// LoginClient 发起登录的客户端信息，由接口层从请求中提取
type LoginClient struct {
	IP        string
	UserAgent string
}

// LoginService 登录服务，处理多种登录方式的认证逻辑
type LoginService struct {
	userService         UserService
	userRepo            user.Repository // 用于验证码登录的新用户创建与失败补偿
	loginHistoryService LoginHistoryService
	rateLimiter         requestRateLimiter
	captcha             captchaVerifier
	verifyCode          VerificationCodeStore
	rsaService          RsaService
	authService         AuthService
	auditLogService     AuditLogService
	settingService      SettingService // 系统配置服务，用于读取动态配置
	accountService      AccountService // 过期改密复用账户服务的改密逻辑
	policy              RuntimePolicy
}

// NewLoginService 创建登录服务
func NewLoginService(
	userService UserService,
	userRepo user.Repository,
	loginHistoryService LoginHistoryService,
	rateLimiter *ratelimit.RateLimiter,
	captcha *captcha.Client,
	verifyCode VerificationCodeStore,
	rsaService RsaService,
	authService AuthService,
	auditLogService AuditLogService,
	settingService SettingService,
	accountService AccountService,
	policy RuntimePolicy,
) *LoginService {
	return &LoginService{
		userService:         userService,
		userRepo:            userRepo,
		loginHistoryService: loginHistoryService,
		rateLimiter:         rateLimiter,
		captcha:             captcha,
		verifyCode:          verifyCode,
		rsaService:          rsaService,
		authService:         authService,
		auditLogService:     auditLogService,
		settingService:      settingService,
		accountService:      accountService,
		policy:              policy,
	}
}

// ChangeExpiredPassword 处理凭证已过期用户的改密：它和密码登录共用同一个失败计数与
// 验证码开关，因此不能被当作绕过登录限流的密码猜测入口。成功后用户用新密码重新登录。
func (s *LoginService) ChangeExpiredPassword(ctx context.Context, req *dto.ExpiredPasswordPutReq) error {
	maxAttempts, lockMinutes := s.getLoginRateLimitConfig(ctx)
	rateLimitKey := fmt.Sprintf("login:password:%s", req.Username)
	blocked, err := s.rateLimiter.IsRequestReachLimit(ctx, rateLimitKey, time.Duration(lockMinutes)*time.Minute, int64(maxAttempts))
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Rate limit check error")
		return ErrRateLimitCheckFailed
	}
	if blocked {
		return ErrTooManyLoginAttempts
	}
	if s.isCaptchaEnabled(ctx) && !s.policy.Development {
		if err := checkCaptcha(ctx, s.captcha, req.CaptchaId, req.CaptchaCode); err != nil {
			return err
		}
	}
	currentPassword, err := s.rsaService.Decrypt(ctx, req.CurrentPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}
	newPassword, err := s.rsaService.Decrypt(ctx, req.NewPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}
	return s.accountService.ChangeExpiredPassword(ctx, req.Username, currentPassword, newPassword)
}

// Login 按登录方式认证用户（第一重凭证）。失败会异步记录登录历史与审计日志；
// 成功不在这里记录：接口层签发会话后（开启两步验证时是第二步通过后）调用 RecordLogin。
func (s *LoginService) Login(ctx context.Context, loginMethod string, req *LoginReq, client LoginClient) (*user.User, error) {
	if !isSupportedLoginMethod(loginMethod) {
		return nil, ErrAuthenticationFailed
	}
	if !s.isLoginMethodAllowed(ctx, loginMethod) {
		return nil, ErrLoginMethodDisabled
	}

	var u *user.User
	var err error
	switch loginMethod {
	case constant.LOGIN_METHOD_PASSWORD:
		u, err = s.loginByPassword(ctx, req)
	case constant.LOGIN_METHOD_EMAIL:
		u, err = s.loginByEmail(ctx, req)
	case constant.LOGIN_METHOD_MOBILE:
		u, err = s.loginByMobile(ctx, req)
	default:
		return nil, ErrAuthenticationFailed
	}

	if err != nil {
		var userID uint
		if u != nil {
			userID = u.ID
		}
		s.RecordLogin(req.Username, client, loginMethod, false, userID)
	}
	return u, err
}

// ResumeLogin 两步验证通过后重新读取用户并检查状态：等待验证码期间账号可能被禁用或锁定。
func (s *LoginService) ResumeLogin(ctx context.Context, userID uint) (*user.User, error) {
	u, err := s.userRepo.Get(ctx, userID)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}
	if err := ValidateUserStatus(u); err != nil {
		return nil, err
	}
	return u, nil
}

// RecordLogin 异步记录一次登录的结果（登录历史与审计日志），不依赖请求生命周期。
func (s *LoginService) RecordLogin(username string, client LoginClient, method string, success bool, userID uint) {
	go s.recordLogin(username, client, method, success, userID)
}

func (s *LoginService) recordLogin(username string, client LoginClient, method string, success bool, userID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.loginHistoryService.Record(ctx, &dto.LoginHistoryPostReq{
		UserID:      userID,
		Username:    username,
		IpAddr:      client.IP,
		UserAgent:   client.UserAgent,
		LoginMethod: method,
		Success:     success,
	}); err != nil {
		logger.Errorf("Failed to record login history: username=%s, err=%v", username, err)
	}
	details := fmt.Sprintf("Login via %s", method)
	if !success {
		details = fmt.Sprintf("Login failed via %s", method)
	}
	s.auditLogService.LogAsync(&audit_log.AuditLog{
		LogType:    audit_log.AuditLogTypeLogin,
		Operator:   username,
		OperatorID: userID,
		Target:     username,
		Details:    details,
		IpAddr:     client.IP,
		Success:    success,
	})
}

func isSupportedLoginMethod(method string) bool {
	switch method {
	case constant.LOGIN_METHOD_PASSWORD, constant.LOGIN_METHOD_EMAIL, constant.LOGIN_METHOD_MOBILE:
		return true
	default:
		return false
	}
}

func (s *LoginService) isLoginMethodAllowed(ctx context.Context, method string) bool {
	if s.settingService == nil {
		return true
	}

	value, err := s.settingService.GetValue(ctx, setting.KeySecurityLoginAllowedMethods)
	if err != nil {
		logger.Warnf("Failed to get allowed login methods setting, using default: %v", err)
		return true
	}

	var methods []string
	if err := json.Unmarshal([]byte(value), &methods); err != nil {
		logger.Warnf("Invalid allowed login methods setting value=%q, using default", value)
		return true
	}
	if len(methods) == 0 {
		logger.Warnf("Allowed login methods setting is empty, using default")
		return true
	}

	for _, allowed := range methods {
		if allowed == method {
			return true
		}
	}
	return false
}

func (s *LoginService) loginByPassword(ctx context.Context, req *LoginReq) (*user.User, error) {
	// 从系统配置获取登录限流参数
	maxAttempts, lockMinutes := s.getLoginRateLimitConfig(ctx)
	rateLimitWindow := time.Duration(lockMinutes) * time.Minute

	// 检查登录速率限制
	rateLimitKey := fmt.Sprintf("login:password:%s", req.Username)
	blocked, err := s.rateLimiter.IsRequestReachLimit(ctx, rateLimitKey, rateLimitWindow, int64(maxAttempts))
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Rate limit check error")
		return nil, ErrRateLimitCheckFailed
	}
	if blocked {
		log.WarnCtx(ctx).Str("username", req.Username).Msg("Login rate limit exceeded")
		return nil, ErrTooManyLoginAttempts
	}

	// 从系统配置获取验证码开关（开发模式下始终跳过验证码）
	captchaEnabled := s.isCaptchaEnabled(ctx)
	if captchaEnabled && !s.policy.Development {
		if err := checkCaptcha(ctx, s.captcha, req.CaptchaId, req.CaptchaCode); err != nil {
			log.WarnCtx(ctx).Err(err).Str("username", req.Username).Msg("Captcha check failed")
			return nil, err
		}
	}

	decryptedPassword, err := s.rsaService.Decrypt(ctx, req.Credential)
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Failed to decrypt password")
		return nil, ErrAuthenticationFailed
	}

	u, err := s.userService.GetRawByUsername(ctx, req.Username)
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Failed to get user by username")
		return nil, ErrAuthenticationFailed
	}

	if !hash.BcryptMatchPassword(decryptedPassword, u.Password) {
		log.WarnCtx(ctx).Str("username", req.Username).Msg("Password is invalid")
		return nil, ErrAuthenticationFailed
	}

	// 验证用户状态
	if err := ValidateUserStatus(u); err != nil {
		log.WarnCtx(ctx).Err(err).Str("username", req.Username).Msg("User status check failed")
		return nil, err
	}

	return u, nil
}

func (s *LoginService) loginByEmail(ctx context.Context, req *LoginReq) (*user.User, error) {
	return s.loginByVerificationCode(ctx, req)
}

func (s *LoginService) loginByMobile(ctx context.Context, req *LoginReq) (*user.User, error) {
	return s.loginByVerificationCode(ctx, req)
}

func (s *LoginService) loginByVerificationCode(ctx context.Context, req *LoginReq) (*user.User, error) {
	// 从系统配置获取登录限流参数
	maxAttempts, lockMinutes := s.getLoginRateLimitConfig(ctx)
	rateLimitWindow := time.Duration(lockMinutes) * time.Minute

	required, err := s.rateLimiter.IsRequestReachLimit(ctx, fmt.Sprintf("%s:%s", "login", req.Username), rateLimitWindow, int64(maxAttempts))
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Rate limit check error")
		return nil, ErrRateLimitCheckFailed
	}

	if required {
		if err := checkCaptcha(ctx, s.captcha, req.CaptchaId, req.CaptchaCode); err != nil {
			log.WarnCtx(ctx).Err(err).Str("username", req.Username).Msg("Captcha check failed")
			return nil, err
		}
	}

	ok, err := s.verifyCode.Verify(ctx, VerificationPurposeAuth, req.Username, req.Credential)
	if err != nil {
		log.ErrCtx(ctx, err).Str("username", req.Username).Msg("Confirm code verification error")
		return nil, ErrAuthenticationFailed
	}
	if !ok {
		log.WarnCtx(ctx).Str("username", req.Username).Msg("Confirm code is invalid")
		return nil, ErrAuthenticationFailed
	}

	return s.getOrCreateUserForVerificationLogin(ctx, req.Username)
}

func (s *LoginService) getOrCreateUserForVerificationLogin(ctx context.Context, username string) (*user.User, error) {
	u, err := s.userService.GetRawByUsername(ctx, username)
	if errors.Is(err, shared.ErrNotFound) {
		// 检查注册开关：静态配置 + 动态配置
		if !s.isRegisterEnabled(ctx) {
			return nil, ErrRegisterNotEnabled
		}

		// 创建 30 秒超时上下文用于完整的用户初始化
		createCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		rawPassword := generator.GenerateCustomUUIDNoConfusingChars(8)
		cryptoPassword, hashErr := hash.BcryptHashPassword(rawPassword)
		if hashErr != nil {
			return nil, ErrAuthenticationFailed
		}
		newUser := &user.User{
			Username:             username,
			Name:                 username,
			Password:             cryptoPassword,
			AccountSource:        constant.ACCOUNT_SOURCE_INTERNAL,
			Enable:               true,
			Locked:               false,
			AccountExpireDate:    time.Now().Add(100 * 365 * 24 * time.Hour),
			CredentialExpireDate: time.Now().Add(100 * 365 * 24 * time.Hour),
		}
		if err := s.userRepo.Create(createCtx, newUser); err != nil {
			log.ErrCtx(ctx, err).Str("username", username).Msg("Failed to create user")
			return nil, ErrAuthenticationFailed
		}
		if err := s.authService.AssignDefaultRole(createCtx, username); err != nil {
			if delErr := s.userRepo.Delete(createCtx, newUser.ID); delErr != nil {
				log.ErrCtx(ctx, delErr).Uint("userId", newUser.ID).Msg("Compensating delete of user after authorization failure failed")
			}
			log.ErrCtx(ctx, err).Str("username", username).Msg("Failed to initialize user authorization")
			return nil, ErrAuthenticationFailed
		}

		// 用户创建和授权初始化成功后，查询完整的用户记录
		u, err = s.userService.GetRawByUsername(ctx, username)
		if err != nil {
			log.ErrCtx(ctx, err).Str("username", username).Msg("Failed to get newly created user")
			return nil, ErrAuthenticationFailed
		}
	} else if err != nil {
		log.ErrCtx(ctx, err).Str("username", username).Msg("Failed to get user by username")
		return nil, ErrAuthenticationFailed
	}

	// 验证用户状态（新注册用户默认状态正常，无需检查）
	if err := ValidateUserStatus(u); err != nil {
		log.WarnCtx(ctx).Err(err).Str("username", username).Msg("User status check failed")
		return nil, err
	}

	return u, nil
}

// ValidateUserStatus 验证用户状态
func ValidateUserStatus(u *user.User) error {
	if !u.Enable {
		return ErrUserDisabled
	}
	if u.Locked {
		return ErrUserLocked
	}
	// 检查账号过期时间（零值表示永不过期）
	if !u.AccountExpireDate.IsZero() && time.Now().After(u.AccountExpireDate) {
		return ErrAccountExpired
	}
	// 检查凭证过期时间（零值表示永不过期）
	if !u.CredentialExpireDate.IsZero() && time.Now().After(u.CredentialExpireDate) {
		return ErrCredentialExpired
	}
	return nil
}

// ============================================
// 系统配置读取方法
// ============================================

// isCaptchaEnabled 从系统配置获取验证码开关
// 配置 key: feature.captcha.enabled
func (s *LoginService) isCaptchaEnabled(ctx context.Context) bool {
	enabled, err := s.settingService.GetBool(ctx, setting.KeyFeatureCaptchaEnabled)
	if err != nil {
		// 配置读取失败时，默认启用验证码（安全优先）
		logger.Warnf("Failed to get captcha setting, using default (enabled): %v", err)
		return true
	}
	return enabled
}

// isRegisterEnabled 检查注册是否启用（静态配置 + 动态配置）
// 先检查 config.toml 中的静态配置，再通过 settingService 检查动态配置
func (s *LoginService) isRegisterEnabled(ctx context.Context) bool {
	// 静态配置检查
	if !s.policy.RegisterEnabled {
		return false
	}
	// 动态配置检查（如果 settingService 可用）
	if s.settingService != nil {
		enabled, err := s.settingService.GetBool(ctx, setting.KeyFeatureRegisterEnabled)
		if err != nil {
			// 读取失败时使用静态配置作为回退
			return s.policy.RegisterEnabled
		}
		return enabled
	}
	return true
}

// getLoginRateLimitConfig 从系统配置获取登录限流参数
// 配置 key: security.login.maxAttempts, security.login.lockMinutes
func (s *LoginService) getLoginRateLimitConfig(ctx context.Context) (maxAttempts int, lockMinutes int) {
	var err error

	maxAttempts, err = s.settingService.GetInt(ctx, setting.KeySecurityLoginMaxAttempts)
	if err != nil || maxAttempts <= 0 {
		logger.Warnf("Failed to get login max attempts setting, using default: %v", err)
		maxAttempts = DefaultLoginRateLimitMaxTries
	}

	lockMinutes, err = s.settingService.GetInt(ctx, setting.KeySecurityLoginLockMinutes)
	if err != nil || lockMinutes <= 0 {
		logger.Warnf("Failed to get login lock minutes setting, using default: %v", err)
		lockMinutes = int(DefaultLoginRateLimitWindow.Minutes())
	}

	return maxAttempts, lockMinutes
}
