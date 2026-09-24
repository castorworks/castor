package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/hyperits/gosuite/security/hash"
)

func TestAccountService_ChangeExpiredPassword(t *testing.T) {
	t.Parallel()
	hashed, err := hash.BcryptHashPassword("Old-password1")
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Hour)
	newFixture := func(mutate func(*user.User)) (*accountService, *mockUserRepo, *revokeTrackingRBAC) {
		repo := newMockUserRepo()
		u := &user.User{ID: 3, Username: "carol", Password: hashed, Enable: true, CredentialExpireDate: expired}
		if mutate != nil {
			mutate(u)
		}
		repo.users["carol"] = u
		rbac := &revokeTrackingRBAC{}
		return &accountService{userRepo: repo, rbac: rbac, settings: NewSettingHelper(newMockSettingRepo())}, repo, rbac
	}

	t.Run("expired credentials are replaced and the new password restarts the clock", func(t *testing.T) {
		t.Parallel()
		svc, repo, rbac := newFixture(nil)
		if err := svc.ChangeExpiredPassword(context.Background(), "carol", "Old-password1", "New-password1"); err != nil {
			t.Fatalf("ChangeExpiredPassword() error = %v", err)
		}
		u := repo.users["carol"]
		if !hash.BcryptMatchPassword("New-password1", u.Password) {
			t.Fatal("password was not updated")
		}
		if !u.CredentialExpireDate.IsZero() {
			t.Fatalf("with no max age the new password must never expire, got %v", u.CredentialExpireDate)
		}
		if len(rbac.revoked) != 1 {
			t.Fatal("changing the password must revoke existing sessions")
		}
	})

	cases := []struct {
		name    string
		mutate  func(*user.User)
		current string
		next    string
		want    error
	}{
		{"wrong current password", nil, "Wrong-password1", "New-password1", apperror.ErrInvalidLoginCredential},
		{"credentials not expired", func(u *user.User) { u.CredentialExpireDate = time.Now().Add(time.Hour) }, "Old-password1", "New-password1", apperror.ErrCredentialNotExpired},
		{"never-expiring credentials", func(u *user.User) { u.CredentialExpireDate = time.Time{} }, "Old-password1", "New-password1", apperror.ErrCredentialNotExpired},
		{"disabled account stays blocked", func(u *user.User) { u.Enable = false }, "Old-password1", "New-password1", apperror.ErrUserDisabled},
		{"locked account stays blocked", func(u *user.User) { u.Locked = true }, "Old-password1", "New-password1", apperror.ErrUserLocked},
		{"expired account stays blocked", func(u *user.User) { u.AccountExpireDate = time.Now().Add(-time.Minute) }, "Old-password1", "New-password1", apperror.ErrAccountExpired},
		{"new password must differ", nil, "Old-password1", "Old-password1", apperror.ErrPasswordUnchanged},
		{"new password must meet the policy", nil, "Old-password1", "weak", apperror.ErrPasswordTooShort},
	}
	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newFixture(nil)
		if err := svc.ChangeExpiredPassword(context.Background(), "nobody", "Old-password1", "New-password1"); !errors.Is(err, apperror.ErrInvalidLoginCredential) {
			t.Fatalf("unknown user error = %v, want ErrInvalidLoginCredential (indistinguishable from a wrong password)", err)
		}
	})
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, rbac := newFixture(tt.mutate)
			err := svc.ChangeExpiredPassword(context.Background(), "carol", tt.current, tt.next)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ChangeExpiredPassword() error = %v, want %v", err, tt.want)
			}
			if u, ok := repo.users["carol"]; ok && u.Password != hashed {
				t.Fatal("password must not change on failure")
			}
			if len(rbac.revoked) != 0 {
				t.Fatal("sessions must not be revoked on failure")
			}
		})
	}
}
