package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Like the initialization test, this requires a disposable PostgreSQL database.
func TestRBACAuthorizationReflectsCommittedChanges(t *testing.T) {
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "InitialPassword123!")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, initErr := Initialize(ctx, db)
	must(initErr)
	resources := persistence.NewResourceRepository(db)
	roles := persistence.NewRoleRepository(db)
	relations := persistence.NewAuthorizationRepository(db)
	users := persistence.NewUserRepository(db)
	catalog := service.NewPermissionService(resources, roles, relations, persistence.NewDepartmentRepository(db))
	rbac := service.NewRBACService(relations, roles, resources, users, persistence.NewDepartmentRepository(db))
	other := service.NewRBACService(relations, roles, resources, users, persistence.NewDepartmentRepository(db))
	account, err := users.GetByUsername(ctx, "system")
	must(err)

	suffix := fmt.Sprint(time.Now().UnixNano())
	senior := &permission.Role{Code: "test_senior_" + suffix, Name: "Test senior", IsEnabled: true}
	junior := &permission.Role{Code: "test_junior_" + suffix, Name: "Test junior", IsEnabled: true}
	must(catalog.CreateRole(ctx, senior))
	must(catalog.CreateRole(ctx, junior))
	resource := &permission.Resource{
		Code: "admin:test-" + suffix + ":detail", Name: "Test detail", Path: "/api/v1/admin/test-" + suffix + "/:id",
		Category: permission.CategoryAdmin, Actions: permission.StringSlice{"GET"}, IsEnabled: true,
	}
	must(catalog.CreateResource(ctx, resource))
	grant := []permission.PermissionGrant{{ResourceID: resource.ID, Actions: []string{"GET"}}}
	must(rbac.SetRolePermissions(ctx, junior.Code, grant))
	must(rbac.SetRoleJuniors(ctx, senior.Code, []string{junior.Code}))
	must(rbac.AddUserRole(ctx, account.Username, senior.Code, false))
	access, err := rbac.CreateSession(ctx, account.ID, time.Hour, service.SessionOrigin{})
	must(err)
	_, err = rbac.SetActiveRoles(ctx, access.SessionID, account.ID, []string{senior.Code})
	must(err)
	menus := service.NewMenuService(persistence.NewMenuRepository(db), resources)
	root := &menu.Menu{Code: "test_directory_" + suffix, Kind: menu.Directory, Titles: menu.Titles{En: "Test", Zh: "测试", Ja: "テスト", Ko: "테스트"}, IsEnabled: true, AccessMode: "authenticated"}
	must(menus.Save(ctx, root))
	page := &menu.Menu{Code: "test_page_" + suffix, ParentID: &root.ID, Kind: menu.Page, Titles: root.Titles, Path: "/dashboard/test-" + suffix, IsEnabled: true, AccessMode: "permission", Permissions: []menu.Permission{{ResourceID: resource.ID, Action: "GET"}}}
	must(menus.Save(ctx, page))
	if err := menus.Delete(ctx, root.ID); !errors.Is(err, apperror.ErrMenuHasChildren) {
		t.Fatalf("parent deletion accepted: %v", err)
	}
	root.ParentID = &page.ID
	if err := menus.Save(ctx, root); !errors.Is(err, apperror.ErrMenuHierarchy) {
		t.Fatalf("cycle accepted: %v", err)
	}
	root.ParentID = nil
	check := func(want bool) {
		t.Helper()
		snapshot, err := other.GetSessionAccess(ctx, access.SessionID, account.ID)
		must(err)
		navigation, err := menus.Navigation(ctx, snapshot.Permissions)
		must(err)
		found := false
		for _, route := range navigation.Routes {
			if route.Path == page.Path {
				found = true
				if route.Allowed != want {
					t.Fatalf("menu visibility=%v want %v", route.Allowed, want)
				}
			}
		}
		if !found {
			t.Fatal("menu page missing from route catalog")
		}
		for i, instance := range []service.RBACService{rbac, other} {
			allowed, err := instance.AuthorizeSession(ctx, access.SessionID, account.ID, resource.Path, "GET")
			if err != nil || allowed != want {
				t.Fatalf("instance %d: allowed=%v err=%v; want %v", i, allowed, err, want)
			}
		}
	}
	check(true)
	must(rbac.SetRolePermissions(ctx, junior.Code, nil))
	check(false)
	must(rbac.SetRolePermissions(ctx, junior.Code, grant))
	check(true)
	resource.IsEnabled = false
	must(catalog.UpdateResource(ctx, resource))
	check(false)
	resource.IsEnabled = true
	must(catalog.UpdateResource(ctx, resource))
	check(true)
	must(rbac.SetRoleJuniors(ctx, senior.Code, nil))
	check(false)
	must(rbac.SetRoleJuniors(ctx, senior.Code, []string{junior.Code}))
	check(true)
	must(rbac.RemoveUserRole(ctx, account.Username, senior.Code))
	allowed, err := other.AuthorizeSession(ctx, access.SessionID, account.ID, resource.Path, "GET")
	if allowed || !errors.Is(err, apperror.ErrAuthorizationSessionRevoked) {
		t.Fatalf("removed role did not revoke session: allowed=%v err=%v", allowed, err)
	}
}

// SetRoleDataScope 在 LockRoles 的事务里写 rbac_roles：若改用另一个连接写，真实数据库上会一直
// 等自己持有的行锁。内存仓储测不出这种自锁，所以在这里用真库验证。
func TestSetRoleDataScopeOnRealDatabase(t *testing.T) {
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "InitialPassword123!")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	departments := persistence.NewDepartmentRepository(db)
	catalog := service.NewPermissionService(persistence.NewResourceRepository(db), persistence.NewRoleRepository(db), persistence.NewAuthorizationRepository(db), departments)
	suffix := fmt.Sprint(time.Now().UnixNano())
	role := &permission.Role{Code: "scope_" + suffix, Name: "Scope test", IsEnabled: true}
	if err := catalog.CreateRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	if role.DataScope != permission.DataScopeSelf {
		t.Fatalf("new roles must start with SELF, got %s", role.DataScope)
	}
	dept := department.Department{Code: "scope-" + suffix, Name: "Scope"}
	if err := departments.Save(ctx, &dept); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.SetRoleDataScope(ctx, role.Code, permission.DataScopeCustom, []uint{dept.ID}); err != nil {
		t.Fatalf("SetRoleDataScope: %v", err)
	}
	loaded, err := catalog.GetRoleByCode(ctx, role.Code)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DataScope != permission.DataScopeCustom || len(loaded.DepartmentIDs) != 1 || loaded.DepartmentIDs[0] != dept.ID {
		t.Fatalf("data scope not stored: %+v", loaded)
	}
	if _, err := catalog.SetRoleDataScope(ctx, role.Code, permission.DataScopeDept, nil); err != nil {
		t.Fatal(err)
	}
	if loaded, _ = catalog.GetRoleByCode(ctx, role.Code); loaded.DataScope != permission.DataScopeDept || len(loaded.DepartmentIDs) != 0 {
		t.Fatalf("switching away from CUSTOM must drop departments: %+v", loaded)
	}
}
