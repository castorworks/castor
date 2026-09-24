package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/user"
)

// revokeStubRBAC 允许 Put 走完撤销会话这一步。
type revokeStubRBAC struct{ stubRBACService }

func (*revokeStubRBAC) RevokeUserSessions(context.Context, uint) error { return nil }

func newUserContactFixture() (*userService, *mockUserRepo) {
	repo := newMockUserRepo()
	svc := &userService{
		userRepo:      repo,
		settingHelper: NewSettingHelper(newMockSettingRepo()),
		rsaService:    &mockRsaService{},
		rbac:          &revokeStubRBAC{},
	}
	return svc, repo
}

func TestUserService_AdminManagedContacts(t *testing.T) {
	t.Parallel()

	t.Run("admin created contacts are marked verified", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()

		resp, err := svc.Post(context.Background(), fullScope, &dto.UserPostReq{
			Username: "alice", Name: "Alice", Password: "Password123",
			Email: "alice@example.com", Mobile: "13800138001",
		})
		if err != nil {
			t.Fatalf("Post() error = %v", err)
		}
		if !resp.EmailVerified || !resp.MobileVerified {
			t.Fatalf("emailVerified = %v mobileVerified = %v, want both true", resp.EmailVerified, resp.MobileVerified)
		}
		created := repo.users["alice"]
		if created.Email != "alice@example.com" || created.Mobile != "13800138001" {
			t.Fatalf("stored contacts = %q/%q", created.Email, created.Mobile)
		}
	})

	t.Run("creating with a taken contact is rejected", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()
		repo.users["owner"] = &user.User{ID: 9, Username: "owner", Email: "taken@example.com", EmailVerified: true}

		_, err := svc.Post(context.Background(), fullScope, &dto.UserPostReq{
			Username: "bob", Password: "Password123", Email: "taken@example.com",
		})
		if !errors.Is(err, ErrContactAlreadyUsed) {
			t.Fatalf("error = %v, want ErrContactAlreadyUsed", err)
		}
	})

	t.Run("updating to a contact held by another user is rejected", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()
		repo.users["owner"] = &user.User{ID: 9, Username: "owner", Mobile: "13800138002", MobileVerified: true}
		repo.users["bob"] = &user.User{ID: 10, Username: "bob"}

		mobile := "13800138002"
		_, err := svc.Put(managerCtx(), fullScope, 10, &dto.UserPutReq{Mobile: &mobile})
		if !errors.Is(err, ErrContactAlreadyUsed) {
			t.Fatalf("error = %v, want ErrContactAlreadyUsed", err)
		}
		if repo.users["bob"].Mobile != "" {
			t.Fatalf("bob must stay without a mobile, got %q", repo.users["bob"].Mobile)
		}
	})

	t.Run("admin set contact on update is verified, empty string unbinds", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()
		repo.users["bob"] = &user.User{ID: 10, Username: "bob"}

		email := "bob@example.com"
		if _, err := svc.Put(managerCtx(), fullScope, 10, &dto.UserPutReq{Email: &email}); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		if repo.users["bob"].Email != email || !repo.users["bob"].EmailVerified {
			t.Fatalf("email = %q verified = %v, want %q true", repo.users["bob"].Email, repo.users["bob"].EmailVerified, email)
		}

		empty := ""
		if _, err := svc.Put(managerCtx(), fullScope, 10, &dto.UserPutReq{Email: &empty}); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		if repo.users["bob"].Email != "" || repo.users["bob"].EmailVerified {
			t.Fatalf("email = %q verified = %v, want unbound", repo.users["bob"].Email, repo.users["bob"].EmailVerified)
		}
	})

	t.Run("keeping one's own contact is allowed", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()
		repo.users["bob"] = &user.User{ID: 10, Username: "bob", Email: "bob@example.com", EmailVerified: true}

		email := "bob@example.com"
		if _, err := svc.Put(managerCtx(), fullScope, 10, &dto.UserPutReq{Email: &email}); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	})

	t.Run("malformed contacts are rejected", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserContactFixture()
		repo.users["bob"] = &user.User{ID: 10, Username: "bob"}

		bad := "not-an-email"
		if _, err := svc.Put(managerCtx(), fullScope, 10, &dto.UserPutReq{Email: &bad}); !errors.Is(err, ErrInvalidEmailFormat) {
			t.Fatalf("error = %v, want ErrInvalidEmailFormat", err)
		}
	})
}
