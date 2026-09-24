package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/hyperits/gosuite/net/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/hyperits/gosuite/security/hash"
)

// AccountService 账户应用服务接口
type AccountService interface {
	PutUserName(ctx context.Context, id uint, name string) error
	// SetNotificationEmailMute 开关通知邮件（只影响邮件通道）
	SetNotificationEmailMute(ctx context.Context, id uint, mute bool) error
	PutUserPassword(ctx context.Context, id uint, req *dto.UserPasswordPutReq) error
	PutUserPasswordByReset(ctx context.Context, req *dto.UserPasswordResetPutReq) error
	// ChangeExpiredPassword 让凭证已过期（因而无法登录）的用户凭当前密码换一个新密码；
	// 传入的是已解密的明文，限流、验证码与审计由调用方（LoginService）负责。
	ChangeExpiredPassword(ctx context.Context, username, currentPassword, newPassword string) error
	PostContactCode(ctx context.Context, userID uint, req *dto.ContactCodePostReq) error
	PutContact(ctx context.Context, userID uint, req *dto.ContactPutReq) error
}

type accountService struct {
	userRepo    user.Repository
	verifyCode  VerificationCodeStore
	codeSender  *codeSender
	rateLimiter requestRateLimiter
	rsaService  RsaService
	rbac        RBACService
	settings    *SettingHelper
}

// NewAccountService 创建账户应用服务
func NewAccountService(
	userRepo user.Repository,
	verifyCode VerificationCodeStore,
	rateLimiter *ratelimit.RateLimiter,
	sms sms.Sender,
	mail *smtpmail.Client,
	policy RuntimePolicy,
	rsaService RsaService,
	rbac RBACService,
	settings *SettingHelper,
) AccountService {
	return &accountService{
		userRepo:    userRepo,
		verifyCode:  verifyCode,
		codeSender:  &codeSender{sms: sms, mail: mail, policy: policy},
		rateLimiter: rateLimiter,
		rsaService:  rsaService,
		rbac:        rbac,
		settings:    settings,
	}
}

// PutUserPasswordByReset resets a password with a one-time code. The code is verified
// before the account is looked up, and an unknown account yields the same error as a
// wrong code, so the endpoint does not reveal which accounts exist.
//
// req.Username 是“账号标识”而不是严格意义上的用户名：密码注册的用户可以用任意用户名，
// 验证码只能发到邮箱/手机号，因此先按已验证的邮箱、再按已验证的手机号解析，
// 最后才回退到用户名（验证码登录自动创建的账号，其用户名本身就是联系方式）。
func (svc *accountService) PutUserPasswordByReset(ctx context.Context, req *dto.UserPasswordResetPutReq) error {
	// RSA 解密新密码（在消耗验证码之前完成，避免因请求格式错误浪费验证码）
	decryptedNewPassword, err := svc.rsaService.Decrypt(ctx, req.NewPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}
	// 同理，不合策略的新密码也在消耗验证码之前拒绝。
	if err := svc.settings.ValidatePassword(ctx, decryptedNewPassword); err != nil {
		return err
	}

	// 用途固定为 reset：登录用途的验证码无法用于重置密码。
	ok, err := svc.verifyCode.Verify(ctx, VerificationPurposeReset, req.Username, req.ConfirmCode)
	if err != nil {
		return fmt.Errorf("verify reset code: %w", err)
	}
	if !ok {
		return ErrInvalidConfirmCode
	}

	item, err := svc.resolveResetTarget(ctx, req.Username)
	if err != nil {
		return err
	}

	return svc.updatePassword(ctx, item, decryptedNewPassword)
}

// resolveResetTarget 按“已验证邮箱 → 已验证手机号 → 用户名”的顺序解析待重置的账号。
// 任何解析失败都返回 ErrInvalidConfirmCode，与验证码错误不可区分。
func (svc *accountService) resolveResetTarget(ctx context.Context, identifier string) (*user.User, error) {
	if item, err := svc.userRepo.GetByEmail(ctx, identifier); err == nil {
		if item.EmailVerified {
			return item, nil
		}
	} else if !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	if item, err := svc.userRepo.GetByMobile(ctx, identifier); err == nil {
		if item.MobileVerified {
			return item, nil
		}
	} else if !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	item, err := svc.userRepo.GetByUsername(ctx, identifier)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, ErrInvalidConfirmCode
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

// PostContactCode 向待绑定的联系方式下发 bind 用途的一次性验证码。
// 调用者已登录，因此同时按调用者和目标联系方式限流，避免把接口当作邮件/短信中继。
func (svc *accountService) PostContactCode(ctx context.Context, userID uint, req *dto.ContactCodePostReq) error {
	contactType, ok := normalizeContactType(req.ContactType)
	if !ok {
		return ErrInvalidContactType
	}
	contact := strings.TrimSpace(req.Contact)
	if err := validateContact(contactType, contact); err != nil {
		return err
	}

	callerLimited, err := svc.rateLimiter.IsRequestReachLimit(ctx, fmt.Sprintf("contact:caller:%d", userID), 1*time.Minute, 3)
	if err != nil {
		return fmt.Errorf("rate limit contact code by caller: %w", err)
	}
	targetLimited, err := svc.rateLimiter.IsRequestReachLimit(ctx, "contact:target:"+contact, 1*time.Minute, 2)
	if err != nil {
		return fmt.Errorf("rate limit contact code by contact: %w", err)
	}
	if callerLimited || targetLimited {
		return ErrTooManyRequests
	}

	code, err := svc.verifyCode.Generate(ctx, VerificationPurposeBind, contact)
	if err != nil {
		return err
	}
	return svc.codeSender.send(ctx, contactType, contact, code)
}

// PutContact 校验 bind 用途的验证码后，把联系方式写到当前登录用户上并标记为已验证。
// 联系方式非空时全局唯一（数据库侧由部分唯一索引兜底）。
func (svc *accountService) PutContact(ctx context.Context, userID uint, req *dto.ContactPutReq) error {
	contactType, ok := normalizeContactType(req.ContactType)
	if !ok {
		return ErrInvalidContactType
	}
	contact := strings.TrimSpace(req.Contact)
	if err := validateContact(contactType, contact); err != nil {
		return err
	}

	verified, err := svc.verifyCode.Verify(ctx, VerificationPurposeBind, contact, req.Code)
	if err != nil {
		return fmt.Errorf("verify bind code: %w", err)
	}
	if !verified {
		return ErrInvalidConfirmCode
	}

	item, err := svc.userRepo.Get(ctx, userID)
	if err != nil {
		return err
	}

	owner, err := svc.findContactOwner(ctx, contactType, contact)
	if err != nil {
		return err
	}
	if owner != nil && owner.ID != item.ID {
		return ErrContactAlreadyUsed
	}

	switch contactType {
	case ContactTypeEmail:
		item.Email = contact
		item.EmailVerified = true
	case ContactTypeMobile:
		item.Mobile = contact
		item.MobileVerified = true
	}

	return svc.userRepo.Update(ctx, item)
}

// findContactOwner 返回持有该联系方式的用户；无人持有时返回 (nil, nil)。
func (svc *accountService) findContactOwner(ctx context.Context, contactType, contact string) (*user.User, error) {
	var (
		owner *user.User
		err   error
	)
	switch contactType {
	case ContactTypeEmail:
		owner, err = svc.userRepo.GetByEmail(ctx, contact)
	case ContactTypeMobile:
		owner, err = svc.userRepo.GetByMobile(ctx, contact)
	default:
		return nil, ErrInvalidContactType
	}
	if errors.Is(err, shared.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return owner, nil
}

func (svc *accountService) PutUserName(ctx context.Context, id uint, name string) error {
	item, err := svc.userRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	item.Name = name

	return svc.userRepo.Update(ctx, item)
}

func (svc *accountService) SetNotificationEmailMute(ctx context.Context, id uint, mute bool) error {
	item, err := svc.userRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	item.MuteNotificationEmails = mute
	return svc.userRepo.Update(ctx, item)
}

func (svc *accountService) PutUserPassword(ctx context.Context, id uint, req *dto.UserPasswordPutReq) error {
	item, err := svc.userRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	// RSA 解密当前密码
	decryptedCurrentPassword, err := svc.rsaService.Decrypt(ctx, req.CurrentPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}

	if !hash.BcryptMatchPassword(decryptedCurrentPassword, item.Password) {
		return ErrCurrentPasswordIncorrect
	}

	// RSA 解密新密码
	decryptedNewPassword, err := svc.rsaService.Decrypt(ctx, req.NewPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}
	if decryptedNewPassword == decryptedCurrentPassword {
		return apperror.ErrPasswordUnchanged
	}

	return svc.updatePassword(ctx, item, decryptedNewPassword)
}

func (svc *accountService) ChangeExpiredPassword(ctx context.Context, username, currentPassword, newPassword string) error {
	item, err := svc.userRepo.GetByUsername(ctx, username)
	if errors.Is(err, shared.ErrNotFound) {
		return apperror.ErrInvalidLoginCredential
	}
	if err != nil {
		return err
	}
	if !hash.BcryptMatchPassword(currentPassword, item.Password) {
		return apperror.ErrInvalidLoginCredential
	}
	// 只有"仅凭证过期"的账号走这里：禁用、锁定、账号过期仍按原样拒绝，
	// 这个入口不能变成绕过其它账号状态的后门。
	if !item.Enable {
		return apperror.ErrUserDisabled
	}
	if item.Locked {
		return apperror.ErrUserLocked
	}
	now := time.Now()
	if !item.AccountExpireDate.IsZero() && !item.AccountExpireDate.After(now) {
		return apperror.ErrAccountExpired
	}
	if item.CredentialExpireDate.IsZero() || item.CredentialExpireDate.After(now) {
		return apperror.ErrCredentialNotExpired
	}
	if newPassword == currentPassword {
		return apperror.ErrPasswordUnchanged
	}
	return svc.updatePassword(ctx, item, newPassword)
}

// updatePassword validates the new password against the policy, stores its hash with a
// fresh credential expiry, and revokes every authorization session of the user, which
// invalidates all issued access tokens bound to those sessions.
func (svc *accountService) updatePassword(ctx context.Context, item *user.User, plainPassword string) error {
	policy := svc.settings.PasswordPolicy(ctx)
	if err := policy.Validate(plainPassword); err != nil {
		return err
	}
	encPassword, err := hash.BcryptHashPassword(plainPassword)
	if err != nil {
		return ErrPasswordEncryptFailed
	}
	item.Password = encPassword
	item.CredentialExpireDate = policy.CredentialExpiry(time.Now())

	if err := svc.userRepo.Update(ctx, item); err != nil {
		return err
	}
	if err := svc.rbac.RevokeUserSessions(ctx, item.ID); err != nil {
		return fmt.Errorf("revoke sessions after password change: %w", err)
	}
	return nil
}
