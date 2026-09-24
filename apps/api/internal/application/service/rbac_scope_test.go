package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
)

// scopeRepo 在 guardRepo 之上补齐会话查询，用于数据范围测试。
type scopeRepo struct {
	*guardRepo
	sessions     map[string]*permission.AuthorizationSession
	sessionRoles map[string][]uint
}

func (r *scopeRepo) GetSession(_ context.Context, id string) (*permission.AuthorizationSession, error) {
	if s, ok := r.sessions[id]; ok {
		return s, nil
	}
	return nil, apperror.ErrRecordNotFound
}
func (r *scopeRepo) GetSessionRoleIDs(_ context.Context, id string) ([]uint, error) {
	return r.sessionRoles[id], nil
}

func deptID(v uint) *uint { return &v }

// 部门树：1 总部 ─ 2 研发 ─ 4 前端
//
//	└ 3 销售
//
// 角色：1 all(ALL) 2 tree(DEPT_AND_CHILDREN) 3 dept(DEPT) 4 self(SELF) 5 custom(CUSTOM: 3) 6 lead(DEPT, 下级角色 custom)
func newScopeService() (*rbacService, *scopeRepo, *guardUsers) {
	repo := &scopeRepo{
		guardRepo:    &guardRepo{assignments: map[uint][]permission.UserRole{}},
		sessions:     map[string]*permission.AuthorizationSession{},
		sessionRoles: map[string][]uint{},
	}
	roles := &guardRoles{roles: []permission.Role{
		{ID: 1, Code: "all", IsEnabled: true, DataScope: permission.DataScopeAll},
		{ID: 2, Code: "tree", IsEnabled: true, DataScope: permission.DataScopeDeptTree},
		{ID: 3, Code: "dept", IsEnabled: true, DataScope: permission.DataScopeDept},
		{ID: 4, Code: "self", IsEnabled: true, DataScope: permission.DataScopeSelf},
		{ID: 5, Code: "custom", IsEnabled: true, DataScope: permission.DataScopeCustom, DepartmentIDs: []uint{3}},
		{ID: 6, Code: "lead", IsEnabled: true, DataScope: permission.DataScopeDept},
		{ID: 7, Code: "system", IsEnabled: true, IsSystem: true, DataScope: permission.DataScopeAll},
	}}
	depts := &memDepartments{}
	for _, d := range []department.Department{
		{ParentID: nil, Code: "hq"}, {ParentID: deptID(1), Code: "rd"}, {ParentID: deptID(1), Code: "sales"}, {ParentID: deptID(2), Code: "fe"},
	} {
		d.ID = uint(len(depts.items) + 1)
		depts.items = append(depts.items, d)
	}
	users := &guardUsers{byName: map[string]*user.User{
		"alice": {ID: 1, Username: "alice", Enable: true, DepartmentID: deptID(2)},
		"bob":   {ID: 2, Username: "bob", Enable: true, DepartmentID: deptID(4)},
		"carol": {ID: 3, Username: "carol", Enable: true, DepartmentID: deptID(3)},
		"dave":  {ID: 4, Username: "dave", Enable: true},
	}}
	svc := NewRBACService(repo, roles, &guardResources{}, users, depts).(*rbacService)
	return svc, repo, users
}

func (r *scopeRepo) GetRoleHierarchies(context.Context) ([]permission.RoleHierarchy, error) {
	return []permission.RoleHierarchy{{SeniorRoleID: 6, JuniorRoleID: 5}}, nil
}

func TestScopeForRoles(t *testing.T) {
	t.Parallel()
	svc, _, users := newScopeService()
	alice := users.byName["alice"]
	dave := users.byName["dave"]
	ctx := context.Background()
	cases := []struct {
		name  string
		u     *user.User
		roles []uint
		want  permission.AccessScope
	}{
		{"all", alice, []uint{1}, permission.AccessScope{All: true, UserID: 1}},
		{"department tree", alice, []uint{2}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2, 4}}},
		{"own department", alice, []uint{3}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2}}},
		{"self", alice, []uint{4}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{}}},
		{"custom", alice, []uint{5}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{3}}},
		{"union of roles", alice, []uint{3, 5}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2, 3}}},
		{"junior roles are included", alice, []uint{6}, permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2, 3}}},
		{"ALL wins", alice, []uint{4, 1}, permission.AccessScope{All: true, UserID: 1}},
		{"relative scopes are empty without a department", dave, []uint{2, 3}, permission.AccessScope{UserID: 4, DepartmentIDs: []uint{}}},
	}
	for _, tc := range cases {
		got, err := svc.scopeForRoles(ctx, tc.u, tc.roles)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got.All != tc.want.All || got.UserID != tc.want.UserID || (!got.All && !slices.Equal(got.DepartmentIDs, tc.want.DepartmentIDs)) {
			t.Fatalf("%s: scope = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// 列表过滤按会话的激活角色：未激活的宽范围角色不生效（RBAC3 最小激活）。
func TestSessionScopeUsesActiveRolesOnly(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newScopeService()
	repo.assignments[1] = []permission.UserRole{{UserID: 1, RoleID: 1}, {UserID: 1, RoleID: 3}}
	repo.sessions["s"] = &permission.AuthorizationSession{ID: "s", UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}
	repo.sessionRoles["s"] = []uint{3}
	scope, err := svc.SessionScope(context.Background(), "s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if scope.All || !slices.Equal(scope.DepartmentIDs, []uint{2}) {
		t.Fatalf("SessionScope = %+v, want only the active DEPT role", scope)
	}
	if _, err := svc.SessionScope(context.Background(), "s", 2); err == nil {
		t.Fatal("another user's session must be rejected")
	}
}

func TestDataScopeGuards(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("assigning a role cannot widen the target beyond the caller", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newScopeService()
		repo.assignments[1] = []permission.UserRole{{UserID: 1, RoleID: 2}} // alice: 研发及以下
		if err := svc.EnsureCanAssignRole(ctx, "alice", "bob", "dept"); err != nil {
			t.Fatalf("bob's own department (前端) is inside alice's tree: %v", err)
		}
		if err := svc.EnsureCanAssignRole(ctx, "alice", "bob", "all"); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("granting ALL beyond the caller: err = %v", err)
		}
		if err := svc.EnsureCanAssignRole(ctx, "alice", "carol", "dept"); !errors.Is(err, apperror.ErrDataScopeExceedsCaller) {
			t.Fatalf("carol's department (销售) is outside alice's tree: err = %v", err)
		}
	})
	t.Run("managing a user with a wider data scope is rejected", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newScopeService()
		repo.assignments[1] = []permission.UserRole{{UserID: 1, RoleID: 3}} // alice: 研发
		repo.assignments[2] = []permission.UserRole{{UserID: 2, RoleID: 1}} // bob: ALL
		if err := svc.EnsureCanManageUser(ctx, 1, 2); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
			t.Fatalf("taking over an ALL-scope account: err = %v", err)
		}
		repo.assignments[2] = []permission.UserRole{{UserID: 2, RoleID: 4}}
		if err := svc.EnsureCanManageUser(ctx, 1, 2); err != nil {
			t.Fatalf("a SELF-scope account is within any scope: %v", err)
		}
	})
	t.Run("setting a role's data scope", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := newScopeService()
		repo.assignments[1] = []permission.UserRole{{UserID: 1, RoleID: 2}} // alice: 研发及以下
		repo.assignments[3] = []permission.UserRole{{UserID: 3, RoleID: 1}} // carol: ALL
		for _, tc := range []struct {
			caller string
			role   string
			scope  permission.DataScope
			depts  []uint
			want   error
		}{
			{"alice", "tree", permission.DataScopeSelf, nil, nil},
			{"alice", "tree", permission.DataScopeCustom, []uint{4}, nil},
			{"alice", "tree", permission.DataScopeCustom, []uint{3}, apperror.ErrDataScopeExceedsCaller},
			{"alice", "tree", permission.DataScopeAll, nil, apperror.ErrDataScopeExceedsCaller},
			{"alice", "tree", permission.DataScopeDept, nil, apperror.ErrDataScopeExceedsCaller},
			{"carol", "tree", permission.DataScopeAll, nil, nil},
			{"carol", "system", permission.DataScopeSelf, nil, apperror.ErrCannotModifySystemRole},
		} {
			if err := svc.EnsureCanSetRoleDataScope(ctx, tc.caller, tc.role, tc.scope, tc.depts); !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
				t.Fatalf("%s sets %s to %s %v: err = %v, want %v", tc.caller, tc.role, tc.scope, tc.depts, err, tc.want)
			}
		}
	})
	t.Run("users outside the scope are invisible", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newScopeService()
		scope := permission.AccessScope{UserID: 1, DepartmentIDs: []uint{2, 4}}
		if err := svc.EnsureUserVisible(ctx, scope, "bob"); err != nil {
			t.Fatalf("bob is in scope: %v", err)
		}
		for _, name := range []string{"carol", "nobody"} {
			if err := svc.EnsureUserVisible(ctx, scope, name); !errors.Is(err, apperror.ErrRecordNotFound) {
				t.Fatalf("%s: err = %v, want ErrRecordNotFound", name, err)
			}
		}
	})
}

func (r *scopeRepo) GetRoleUserIDs(_ context.Context, roleID uint) ([]uint, error) {
	var ids []uint
	for userID, assignments := range r.assignments {
		for _, a := range assignments {
			if a.RoleID == roleID {
				ids = append(ids, userID)
			}
		}
	}
	return ids, nil
}

// 角色成员列表（以及据此计算的成员数）只包含调用者范围内的用户。
func TestGetRoleUsernamesIsScoped(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newScopeService()
	for _, id := range []uint{1, 2, 3, 4} {
		repo.assignments[id] = []permission.UserRole{{UserID: id, RoleID: 4}}
	}
	ctx := context.Background()
	all, err := svc.GetRoleUsernames(ctx, permission.AccessScope{All: true}, "self")
	if err != nil || !slices.Equal(all, []string{"alice", "bob", "carol", "dave"}) {
		t.Fatalf("ALL scope members = %v, %v", all, err)
	}
	scoped, err := svc.GetRoleUsernames(ctx, permission.AccessScope{UserID: 4, DepartmentIDs: []uint{2, 4}}, "self")
	if err != nil || !slices.Equal(scoped, []string{"alice", "bob", "dave"}) {
		t.Fatalf("scoped members = %v, %v (want alice/bob in scope plus the caller dave)", scoped, err)
	}
}
