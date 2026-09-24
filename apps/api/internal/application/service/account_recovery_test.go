package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

// purposeScopedCodeStore 按 (purpose, target) 存放验证码，用于验证用途隔离：
// 登录用途的验证码不能被重放去重置密码。
type purposeScopedCodeStore struct {
	codes map[string]string
}

func newPurposeScopedCodeStore() *purposeScopedCodeStore {
	return &purposeScopedCodeStore{codes: make(map[string]string)}
}

func (s *purposeScopedCodeStore) key(purpose, target string) string {
	return purpose + "\x00" + target
}

func (s *purposeScopedCodeStore) issue(purpose, target, code string) {
	s.codes[s.key(purpose, target)] = code
}

func (s *purposeScopedCodeStore) Generate(_ context.Context, purpose, target string) (string, error) {
	code := "654321"
	s.codes[s.key(purpose, target)] = code
	return code, nil
}

func (s *purposeScopedCodeStore) Verify(_ context.Context, purpose, target, code string) (bool, error) {
	stored, ok := s.codes[s.key(purpose, target)]
	if !ok || stored != code {
		return false, nil
	}
	delete(s.codes, s.key(purpose, target))
	return true, nil
}

// newRecoveryFixture 构造一个带联系方式的账户服务，验证码存储按用途隔离。
func newRecoveryFixture(t *testing.T) (*accountService, *mockUserRepo, *purposeScopedCodeStore, *revokeTrackingRBAC) {
	t.Helper()
	hashed, err := hash.BcryptHashPassword("old-password")
	if err != nil {
		t.Fatal(err)
	}
	repo := newMockUserRepo()
	repo.users["pw-user"] = &user.User{
		ID: 11, Username: "pw-user", Password: hashed,
		Email: "owner@example.com", EmailVerified: true,
		Mobile: "13800138000", MobileVerified: true,
	}
	store := newPurposeScopedCodeStore()
	rbac := &revokeTrackingRBAC{}
	svc := &accountService{
		userRepo:    repo,
		verifyCode:  store,
		codeSender:  &codeSender{policy: RuntimePolicy{Development: true}},
		rateLimiter: &mockRateLimiter{},
		rsaService:  &mockRsaService{},
		rbac:        rbac,
		settings:    NewSettingHelper(newMockSettingRepo()),
	}
	return svc, repo, store, rbac
}

func TestAccountService_ResetResolvesAccountByContact(t *testing.T) {
	t.Parallel()

	t.Run("verified email", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, rbac := newRecoveryFixture(t)
		store.issue(VerificationPurposeReset, "owner@example.com", "123456")

		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
			Username: "owner@example.com", ConfirmCode: "123456", NewPassword: "New-password1",
		})
		if err != nil {
			t.Fatalf("PutUserPasswordByReset() error = %v", err)
		}
		if !hash.BcryptMatchPassword("New-password1", repo.users["pw-user"].Password) {
			t.Fatal("password of the account owning the email was not updated")
		}
		if len(rbac.revoked) != 1 || rbac.revoked[0] != 11 {
			t.Fatalf("sessions revoked = %v, want [11]", rbac.revoked)
		}
	})

	t.Run("verified mobile", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, rbac := newRecoveryFixture(t)
		store.issue(VerificationPurposeReset, "13800138000", "123456")

		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
			Username: "13800138000", ConfirmCode: "123456", NewPassword: "New-password1",
		})
		if err != nil {
			t.Fatalf("PutUserPasswordByReset() error = %v", err)
		}
		if !hash.BcryptMatchPassword("New-password1", repo.users["pw-user"].Password) {
			t.Fatal("password of the account owning the mobile was not updated")
		}
		if len(rbac.revoked) != 1 || rbac.revoked[0] != 11 {
			t.Fatalf("sessions revoked = %v, want [11]", rbac.revoked)
		}
	})

	t.Run("username fallback", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, _ := newRecoveryFixture(t)
		store.issue(VerificationPurposeReset, "pw-user", "123456")

		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
			Username: "pw-user", ConfirmCode: "123456", NewPassword: "New-password1",
		})
		if err != nil {
			t.Fatalf("PutUserPasswordByReset() error = %v", err)
		}
		if !hash.BcryptMatchPassword("New-password1", repo.users["pw-user"].Password) {
			t.Fatal("password was not updated through the username fallback")
		}
	})

	t.Run("unverified contact is not a reset target", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, rbac := newRecoveryFixture(t)
		repo.users["pw-user"].EmailVerified = false
		store.issue(VerificationPurposeReset, "owner@example.com", "123456")

		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
			Username: "owner@example.com", ConfirmCode: "123456", NewPassword: "New-password1",
		})
		if !errors.Is(err, ErrInvalidConfirmCode) {
			t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
		}
		if len(rbac.revoked) != 0 {
			t.Fatalf("no sessions should be revoked: %v", rbac.revoked)
		}
	})

	t.Run("unknown contact is indistinguishable from a wrong code", func(t *testing.T) {
		t.Parallel()
		svc, _, store, rbac := newRecoveryFixture(t)
		// 验证码本身有效，但没有任何账号持有该联系方式。
		store.issue(VerificationPurposeReset, "nobody@example.com", "123456")

		err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
			Username: "nobody@example.com", ConfirmCode: "123456", NewPassword: "New-password1",
		})
		if !errors.Is(err, ErrInvalidConfirmCode) {
			t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
		}
		if len(rbac.revoked) != 0 {
			t.Fatalf("no sessions should be revoked: %v", rbac.revoked)
		}
	})
}

// TestAccountService_LoginCodeCannotResetPassword 锁定跨用途重放：
// POST /auth/code 默认下发的 auth 用途验证码，不得用于密码重置。
func TestAccountService_LoginCodeCannotResetPassword(t *testing.T) {
	t.Parallel()
	svc, repo, store, rbac := newRecoveryFixture(t)
	store.issue(VerificationPurposeAuth, "owner@example.com", "123456")

	err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
		Username: "owner@example.com", ConfirmCode: "123456", NewPassword: "New-password1",
	})
	if !errors.Is(err, ErrInvalidConfirmCode) {
		t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
	}
	if hash.BcryptMatchPassword("New-password1", repo.users["pw-user"].Password) {
		t.Fatal("an auth-purpose code must not reset a password")
	}
	if len(rbac.revoked) != 0 {
		t.Fatalf("no sessions should be revoked: %v", rbac.revoked)
	}
	// 该验证码必须仍然留在 auth 用途下，未被重置流程消耗。
	if _, ok := store.codes[store.key(VerificationPurposeAuth, "owner@example.com")]; !ok {
		t.Fatal("the auth-purpose code must not be consumed by the reset flow")
	}
}

// TestAccountService_BindCodeCannotResetPassword 锁定另一个方向的跨用途重放。
func TestAccountService_BindCodeCannotResetPassword(t *testing.T) {
	t.Parallel()
	svc, _, store, _ := newRecoveryFixture(t)
	store.issue(VerificationPurposeBind, "owner@example.com", "123456")

	err := svc.PutUserPasswordByReset(context.Background(), &dto.UserPasswordResetPutReq{
		Username: "owner@example.com", ConfirmCode: "123456", NewPassword: "New-password1",
	})
	if !errors.Is(err, ErrInvalidConfirmCode) {
		t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
	}
}

func TestAccountService_PutContact(t *testing.T) {
	t.Parallel()

	t.Run("binds a verified email to the caller", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, _ := newRecoveryFixture(t)
		repo.users["newcomer"] = &user.User{ID: 12, Username: "newcomer"}
		store.issue(VerificationPurposeBind, "fresh@example.com", "123456")

		err := svc.PutContact(context.Background(), 12, &dto.ContactPutReq{
			ContactType: "EMAIL", Contact: "fresh@example.com", Code: "123456",
		})
		if err != nil {
			t.Fatalf("PutContact() error = %v", err)
		}
		bound := repo.users["newcomer"]
		if bound.Email != "fresh@example.com" || !bound.EmailVerified {
			t.Fatalf("email = %q verified = %v, want fresh@example.com true", bound.Email, bound.EmailVerified)
		}
	})

	t.Run("rejects a contact owned by another user", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, _ := newRecoveryFixture(t)
		repo.users["newcomer"] = &user.User{ID: 12, Username: "newcomer"}
		store.issue(VerificationPurposeBind, "owner@example.com", "123456")

		err := svc.PutContact(context.Background(), 12, &dto.ContactPutReq{
			ContactType: "email", Contact: "owner@example.com", Code: "123456",
		})
		if !errors.Is(err, ErrContactAlreadyUsed) {
			t.Fatalf("error = %v, want ErrContactAlreadyUsed", err)
		}
		if repo.users["newcomer"].Email != "" {
			t.Fatalf("email must stay unbound, got %q", repo.users["newcomer"].Email)
		}
		if repo.users["pw-user"].Email != "owner@example.com" {
			t.Fatal("the original owner must keep the contact")
		}
	})

	t.Run("rejects a mobile owned by another user", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, _ := newRecoveryFixture(t)
		repo.users["newcomer"] = &user.User{ID: 12, Username: "newcomer"}
		store.issue(VerificationPurposeBind, "13800138000", "123456")

		err := svc.PutContact(context.Background(), 12, &dto.ContactPutReq{
			ContactType: "MOBILE", Contact: "13800138000", Code: "123456",
		})
		if !errors.Is(err, ErrContactAlreadyUsed) {
			t.Fatalf("error = %v, want ErrContactAlreadyUsed", err)
		}
	})

	t.Run("rebinding the same contact to its owner is allowed", func(t *testing.T) {
		t.Parallel()
		svc, _, store, _ := newRecoveryFixture(t)
		store.issue(VerificationPurposeBind, "owner@example.com", "123456")

		err := svc.PutContact(context.Background(), 11, &dto.ContactPutReq{
			ContactType: "email", Contact: "owner@example.com", Code: "123456",
		})
		if err != nil {
			t.Fatalf("PutContact() error = %v", err)
		}
	})

	t.Run("a reset-purpose code cannot bind a contact", func(t *testing.T) {
		t.Parallel()
		svc, repo, store, _ := newRecoveryFixture(t)
		repo.users["newcomer"] = &user.User{ID: 12, Username: "newcomer"}
		store.issue(VerificationPurposeReset, "fresh@example.com", "123456")

		err := svc.PutContact(context.Background(), 12, &dto.ContactPutReq{
			ContactType: "email", Contact: "fresh@example.com", Code: "123456",
		})
		if !errors.Is(err, ErrInvalidConfirmCode) {
			t.Fatalf("error = %v, want ErrInvalidConfirmCode", err)
		}
	})

	t.Run("rejects an unknown contact type", func(t *testing.T) {
		t.Parallel()
		svc, _, _, _ := newRecoveryFixture(t)
		err := svc.PutContact(context.Background(), 11, &dto.ContactPutReq{
			ContactType: "fax", Contact: "owner@example.com", Code: "123456",
		})
		if !errors.Is(err, ErrInvalidContactType) {
			t.Fatalf("error = %v, want ErrInvalidContactType", err)
		}
	})

	t.Run("rejects a malformed contact", func(t *testing.T) {
		t.Parallel()
		svc, _, _, _ := newRecoveryFixture(t)
		err := svc.PutContact(context.Background(), 11, &dto.ContactPutReq{
			ContactType: "EMAIL", Contact: "not-an-email", Code: "123456",
		})
		if !errors.Is(err, ErrInvalidEmailFormat) {
			t.Fatalf("error = %v, want ErrInvalidEmailFormat", err)
		}
	})
}

func TestAccountService_PostContactCode(t *testing.T) {
	t.Parallel()

	t.Run("issues a bind-purpose code for the contact", func(t *testing.T) {
		t.Parallel()
		svc, _, store, _ := newRecoveryFixture(t)
		// 用手机号通道：开发模式下短信直接短路，无需真实发送器。
		err := svc.PostContactCode(context.Background(), 11, &dto.ContactCodePostReq{
			ContactType: "Mobile", Contact: "13900139000",
		})
		if err != nil {
			t.Fatalf("PostContactCode() error = %v", err)
		}
		if _, ok := store.codes[store.key(VerificationPurposeBind, "13900139000")]; !ok {
			t.Fatal("no bind-purpose code was issued for the contact")
		}
		if _, ok := store.codes[store.key(VerificationPurposeAuth, "13900139000")]; ok {
			t.Fatal("the bind flow must not issue an auth-purpose code")
		}
	})

	t.Run("rate limited callers get a 429-mapped error", func(t *testing.T) {
		t.Parallel()
		svc, _, _, _ := newRecoveryFixture(t)
		svc.rateLimiter = &mockRateLimiter{blocked: true}
		err := svc.PostContactCode(context.Background(), 11, &dto.ContactCodePostReq{
			ContactType: "EMAIL", Contact: "fresh@example.com",
		})
		if !errors.Is(err, ErrTooManyRequests) {
			t.Fatalf("error = %v, want ErrTooManyRequests", err)
		}
	})

	t.Run("rejects a malformed mobile before issuing a code", func(t *testing.T) {
		t.Parallel()
		svc, _, store, _ := newRecoveryFixture(t)
		err := svc.PostContactCode(context.Background(), 11, &dto.ContactCodePostReq{
			ContactType: "MOBILE", Contact: "not-a-phone",
		})
		if !errors.Is(err, ErrInvalidPhoneFormat) {
			t.Fatalf("error = %v, want ErrInvalidPhoneFormat", err)
		}
		if len(store.codes) != 0 {
			t.Fatalf("no code should be issued, got %v", store.codes)
		}
	})
}
