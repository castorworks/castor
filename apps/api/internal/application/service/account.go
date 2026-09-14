package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

// AccountService 账户应用服务接口
type AccountService interface {
	PutUserName(ctx context.Context, id uint, name string) error
	PutUserPassword(ctx context.Context, id uint, req *dto.UserPasswordPutReq) error
	PutUserPasswordByReset(ctx context.Context, req *dto.UserPasswordResetPutReq) error
}

type accountService struct {
	userRepo   user.Repository
	verifyCode VerificationCodeStore
	rsaService RsaService
	rbac       RBACService
}

// NewAccountService 创建账户应用服务
func NewAccountService(
	userRepo user.Repository,
	verifyCode VerificationCodeStore,
	rsaService RsaService,
	rbac RBACService,
) AccountService {
	return &accountService{
		userRepo:   userRepo,
		verifyCode: verifyCode,
		rsaService: rsaService,
		rbac:       rbac,
	}
}

// PutUserPasswordByReset resets a password with a one-time code. The code is verified
// before the account is looked up, and an unknown username yields the same error as a
// wrong code, so the endpoint does not reveal which usernames exist.
func (svc *accountService) PutUserPasswordByReset(ctx context.Context, req *dto.UserPasswordResetPutReq) error {
	// RSA 解密新密码（在消耗验证码之前完成，避免因请求格式错误浪费验证码）
	decryptedNewPassword, err := svc.rsaService.Decrypt(ctx, req.NewPassword)
	if err != nil {
		return ErrPasswordDecryptFailed
	}

	ok, err := svc.verifyCode.Verify(ctx, VerificationPurposeAuth, req.Username, req.ConfirmCode)
	if err != nil {
		return fmt.Errorf("verify reset code: %w", err)
	}
	if !ok {
		return ErrInvalidConfirmCode
	}

	item, err := svc.userRepo.GetByUsername(ctx, req.Username)
	if errors.Is(err, shared.ErrNotFound) {
		return ErrInvalidConfirmCode
	}
	if err != nil {
		return err
	}

	return svc.updatePassword(ctx, item, decryptedNewPassword)
}

func (svc *accountService) PutUserName(ctx context.Context, id uint, name string) error {
	item, err := svc.userRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	item.Name = name

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

	return svc.updatePassword(ctx, item, decryptedNewPassword)
}

// updatePassword stores the new hash and revokes every authorization session of the
// user, which invalidates all issued access tokens bound to those sessions.
func (svc *accountService) updatePassword(ctx context.Context, item *user.User, plainPassword string) error {
	encPassword, err := hash.BcryptHashPassword(plainPassword)
	if err != nil {
		return ErrPasswordEncryptFailed
	}
	item.Password = encPassword

	if err := svc.userRepo.Update(ctx, item); err != nil {
		return err
	}
	if err := svc.rbac.RevokeUserSessions(ctx, item.ID); err != nil {
		return fmt.Errorf("revoke sessions after password change: %w", err)
	}
	return nil
}
