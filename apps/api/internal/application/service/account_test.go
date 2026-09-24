package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

type revokeTrackingRBAC struct {
	RBACService
	revoked []uint
}

func (r *revokeTrackingRBAC) RevokeUserSessions(_ context.Context, userID uint) error {
	r.revoked = append(r.revoked, userID)
	return nil
}

type countingUserRepo struct {
	*mockUserRepo
	lookups int
}

func (r *countingUserRepo) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	r.lookups++
	return r.mockUserRepo.GetByUsername(ctx, username)
}

func newAccountFixture(t *testing.T, codeValid bool) (*accountService, *countingUserRepo, *revokeTrackingRBAC) {
	t.Helper()
	hashed, err := hash.BcryptHashPassword("old-password")
	if err != nil {
		t.Fatal(err)
	}
	repo := &countingUserRepo{mockUserRepo: newMockUserRepo()}
	repo.users["alice@example.com"] = &user.User{ID: 7, Username: "alice@example.com", Password: hashed}
	rbac := &revokeTrackingRBAC{}
	svc := &accountService{
		userRepo:   repo,
		verifyCode: &mockVerifyCodeClient{valid: codeValid},
		rsaService: &mockRsaService{},
		rbac:       rbac,
		settings:   NewSettingHelper(newMockSettingRepo()),
	}
	return svc, repo, rbac
}

func TestAccountService_PutUserPasswordRevokesSessions(t *testing.T) {
	t.Parallel()
	svc, repo, rbac := newAccountFixture(t, true)
	err := svc.PutUserPassword(context.Background(), 7, &dto.UserPasswordPutReq{CurrentPassword: "old-password", NewPassword: "New-password1"})
	if err != nil {
		t.Fatalf("PutUserPassword() error = %v", err)
	}
	if !hash.BcryptMatchPassword("New-password1", repo.users["alice@example.com"].Password) {
		t.Fatal("password was not updated")
	}
	if len(rbac.revoked) != 1 || rbac.revoked[0] != 7 {
		t.Fatalf("sessions revoked = %v, want [7]", rbac.revoked)
	}
}

func TestAccountService_PutUserPasswordWrongCurrentKeepsSessions(t *testing.T) {
	t.Parallel()
	svc, _, rbac := newAccountFixture(t, true)
	err := svc.PutUserPassword(context.Background(), 7, &dto.UserPasswordPutReq{CurrentPassword: "wrong", NewPassword: "New-password1"})
	if !errors.Is(err, ErrCurrentPasswordIncorrect) {
		t.Fatalf("error = %v, want ErrCurrentPasswordIncorrect", err)
	}
	if len(rbac.revoked) != 0 {
		t.Fatalf("sessions must not be revoked on failure: %v", rbac.revoked)
	}
}

func TestAccountService_PutUserPasswordByReset(t *testing.T) {
	t.Parallel()

	t.Run("valid code updates password and revokes sessions", func(t *testing.T) {
		t.Parallel()
		svc, repo, rbac := newAccountFixture(t, true)
		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{Username: "alice@example.com", ConfirmCode: "123456", NewPassword: "New-password1"})
		if err != nil {
			t.Fatalf("PutUserPasswordByReset() error = %v", err)
		}
		if !hash.BcryptMatchPassword("New-password1", repo.users["alice@example.com"].Password) {
			t.Fatal("password was not updated")
		}
		if len(rbac.revoked) != 1 || rbac.revoked[0] != 7 {
			t.Fatalf("sessions revoked = %v, want [7]", rbac.revoked)
		}
	})

	t.Run("invalid code is rejected before user lookup", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newAccountFixture(t, false)
		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{Username: "alice@example.com", ConfirmCode: "000000", NewPassword: "New-password1"})
		if !errors.Is(err, ErrInvalidConfirmCode) {
			t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
		}
		if repo.lookups != 0 {
			t.Fatalf("user lookups = %d, want 0", repo.lookups)
		}
	})

	t.Run("unknown user returns the same error as a wrong code", func(t *testing.T) {
		t.Parallel()
		for _, valid := range []bool{true, false} {
			svc, _, rbac := newAccountFixture(t, valid)
			err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{Username: "nobody@example.com", ConfirmCode: "123456", NewPassword: "New-password1"})
			if !errors.Is(err, ErrInvalidConfirmCode) {
				t.Fatalf("codeValid=%v: error = %v, want ErrInvalidConfirmCode", valid, err)
			}
			if len(rbac.revoked) != 0 {
				t.Fatalf("no sessions should be revoked: %v", rbac.revoked)
			}
		}
	})
}
