package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/user"
)

// denyManageRBAC 模拟目标用户权限超出调用者的情形。
type denyManageRBAC struct {
	stubRBACService
	checked [][2]uint
}

func (r *denyManageRBAC) EnsureCanManageUser(_ context.Context, callerID, targetID uint) error {
	r.checked = append(r.checked, [2]uint{callerID, targetID})
	return apperror.ErrTargetUserExceedsCaller
}

// 受托的用户管理员不能通过改密码、改状态或删除来接管权限更高的账号。
func TestUserService_RejectsManagingMorePrivilegedUser(t *testing.T) {
	t.Parallel()
	newFixture := func() (*userService, *mockUserRepo, *denyManageRBAC) {
		repo := newMockUserRepo()
		repo.users["boss"] = &user.User{ID: 5, Username: "boss", Password: "old-hash", Enable: true}
		rbac := &denyManageRBAC{}
		return &userService{userRepo: repo, settingHelper: NewSettingHelper(newMockSettingRepo()), rsaService: &mockRsaService{}, rbac: rbac}, repo, rbac
	}

	t.Run("password reset", func(t *testing.T) {
		t.Parallel()
		svc, repo, rbac := newFixture()
		_, err := svc.Put(managerCtx(), fullScope, 5, &dto.UserPutReq{Password: "Takeover-123"})
		if !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("Put() error = %v, want ErrTargetUserExceedsCaller", err)
		}
		if repo.users["boss"].Password != "old-hash" {
			t.Fatal("password must not change when the guard rejects")
		}
		if len(rbac.checked) != 1 || rbac.checked[0] != [2]uint{managerID, 5} {
			t.Fatalf("guard checked %v, want caller %d on target 5", rbac.checked, managerID)
		}
	})
	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newFixture()
		if err := svc.Delete(managerCtx(), fullScope, 5); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("Delete() error = %v, want ErrTargetUserExceedsCaller", err)
		}
		if _, ok := repo.users["boss"]; !ok {
			t.Fatal("user must not be deleted when the guard rejects")
		}
	})
	t.Run("missing caller identity fails closed", func(t *testing.T) {
		t.Parallel()
		repo := newMockUserRepo()
		repo.users["boss"] = &user.User{ID: 5, Username: "boss", Enable: true}
		svc := &userService{userRepo: repo, settingHelper: NewSettingHelper(newMockSettingRepo()), rsaService: &mockRsaService{}, rbac: &stubRBACService{}}
		if _, err := svc.Put(context.Background(), fullScope, 5, &dto.UserPutReq{Name: "x"}); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("Put() without caller error = %v, want ErrTargetUserExceedsCaller", err)
		}
	})
}
