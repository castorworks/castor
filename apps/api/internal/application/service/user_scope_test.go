package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
)

// optionRecordingUserRepo 记下列表查询收到的条件，验证数据范围被追加到了查询里。
type optionRecordingUserRepo struct {
	*mockUserRepo
	opts []query.Option
}

func (r *optionRecordingUserRepo) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]user.User, int64, error) {
	r.opts = opts
	return r.mockUserRepo.Gets(ctx, page, size, order, opts...)
}

func newUserScopeFixture() (*userService, *optionRecordingUserRepo) {
	repo := &optionRecordingUserRepo{mockUserRepo: newMockUserRepo()}
	repo.users["inside"] = &user.User{ID: 21, Username: "inside", Enable: true, DepartmentID: deptID(2)}
	repo.users["outside"] = &user.User{ID: 22, Username: "outside", Enable: true, DepartmentID: deptID(3)}
	repo.users["loose"] = &user.User{ID: 23, Username: "loose", Enable: true}
	depts := &memDepartments{}
	for _, code := range []string{"hq", "rd", "sales"} {
		d := department.Department{Code: code, Name: code}
		d.ID = uint(len(depts.items) + 1)
		depts.items = append(depts.items, d)
	}
	svc := &userService{
		userRepo: repo, settingHelper: NewSettingHelper(newMockSettingRepo()), rsaService: &mockRsaService{},
		rbac: &stubRBACService{}, departments: depts,
	}
	return svc, repo
}

func TestUserService_DataScope(t *testing.T) {
	t.Parallel()
	scoped := permission.AccessScope{UserID: managerID, DepartmentIDs: []uint{2}}
	ctx := managerCtx()

	t.Run("list is filtered by the scope", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserScopeFixture()
		if _, _, err := svc.Gets(ctx, scoped, 1, 10, "id desc"); err != nil {
			t.Fatal(err)
		}
		want := query.UserScope("id", false, managerID, []uint{2})
		if len(repo.opts) != 1 || repo.opts[0].Condition != want.Condition {
			t.Fatalf("scope option not applied: %+v", repo.opts)
		}
		if _, _, err := svc.Gets(ctx, fullScope, 1, 10, "id desc"); err != nil || len(repo.opts) != 0 {
			t.Fatalf("ALL scope must not filter: %+v", repo.opts)
		}
	})
	t.Run("users outside the scope do not exist for the caller", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserScopeFixture()
		if _, err := svc.Get(ctx, scoped, 21); err != nil {
			t.Fatalf("user inside the scope: %v", err)
		}
		for _, id := range []uint{22, 23} {
			if _, err := svc.Get(ctx, scoped, id); !errors.Is(err, shared.ErrNotFound) {
				t.Fatalf("Get(%d) err = %v, want ErrNotFound", id, err)
			}
			if _, err := svc.Put(ctx, scoped, id, &dto.UserPutReq{Name: "x"}); !errors.Is(err, shared.ErrNotFound) {
				t.Fatalf("Put(%d) err = %v, want ErrNotFound", id, err)
			}
			if err := svc.Delete(ctx, scoped, id); !errors.Is(err, shared.ErrNotFound) {
				t.Fatalf("Delete(%d) err = %v, want ErrNotFound", id, err)
			}
		}
		if repo.users["outside"].Name == "x" {
			t.Fatal("out-of-scope user must not change")
		}
	})
	t.Run("department assignment stays inside the scope", func(t *testing.T) {
		t.Parallel()
		svc, repo := newUserScopeFixture()
		post := func(scope permission.AccessScope, dept *uint) error {
			_, err := svc.Post(ctx, scope, &dto.UserPostReq{Username: "n" + time.Now().Format("150405.000000"), Password: "Password123", DepartmentID: dept})
			return err
		}
		if err := post(scoped, deptID(2)); err != nil {
			t.Fatalf("creating inside the scope: %v", err)
		}
		if err := post(scoped, deptID(3)); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("creating outside the scope: %v", err)
		}
		if err := post(scoped, nil); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("a scoped manager must place new users in a department: %v", err)
		}
		if err := post(fullScope, nil); err != nil {
			t.Fatalf("ALL scope may leave the department empty: %v", err)
		}
		if err := post(fullScope, deptID(99)); !errors.Is(err, apperror.ErrDepartmentNotFound) {
			t.Fatalf("unknown department: %v", err)
		}
		zero := uint(0)
		if _, err := svc.Put(ctx, scoped, 21, &dto.UserPutReq{DepartmentID: &zero}); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("a scoped manager must not drop a user out of every department: %v", err)
		}
		if _, err := svc.Put(ctx, scoped, 21, &dto.UserPutReq{DepartmentID: deptID(3)}); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("moving a user out of the scope: %v", err)
		}
		if _, err := svc.Put(ctx, fullScope, 21, &dto.UserPutReq{DepartmentID: &zero}); err != nil || repo.users["inside"].DepartmentID != nil {
			t.Fatalf("ALL scope clears the department: err=%v dept=%v", err, repo.users["inside"].DepartmentID)
		}
	})
}

func TestAuditLogService_CleanupNeedsFullScope(t *testing.T) {
	t.Parallel()
	svc := NewAuditLogService(&mockAuditLogRepo{})
	if _, err := svc.DeleteBefore(context.Background(), permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2}}, time.Now()); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
		t.Fatalf("scoped cleanup would delete other people's logs: err = %v", err)
	}
}

func TestSessionService_RevokeOutsideScope(t *testing.T) {
	t.Parallel()
	now := time.Now()
	repo := &sessionRepoStub{sessions: map[string]*permission.AuthorizationSession{
		"live": {ID: "live", UserID: 5, Username: "boss", ExpiresAt: now.Add(time.Hour)},
	}}
	users := newMockUserRepo()
	users.users["boss"] = &user.User{ID: 5, Username: "boss", DepartmentID: deptID(3)}
	svc := &sessionService{repo: repo, rbac: &manageGuardRBAC{allow: true}, users: users}
	if _, err := svc.Revoke(context.Background(), permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2}}, 1, "live"); !errors.Is(err, apperror.ErrRecordNotFound) {
		t.Fatalf("revoking a session outside the scope: err = %v", err)
	}
	if len(repo.revoked) != 0 {
		t.Fatal("nothing must be revoked")
	}
}
