package persistence

import (
	"context"
	"os"
	"slices"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
)

// departmentTestSchema 让本文件的表与其它包的集成测试互不干扰。
const departmentTestSchema = "department_repo_test"

// 需要一次性的空库，绝不能指向开发库或生产库。
func openDepartmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("DROP SCHEMA IF EXISTS " + departmentTestSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA " + departmentTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}
	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, departmentTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UserModel{}, &models.RoleModel{}, &models.DepartmentModel{}, &models.RoleDepartmentModel{}, &models.LoginHistoryModel{}); err != nil {
		t.Fatal(err)
	}
	// 与初始结构相同的外键：有成员的部门删不掉。
	if err := db.Exec(`ALTER TABLE users ADD CONSTRAINT fk_users_department FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE RESTRICT`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDepartmentsRolesAndUserScope(t *testing.T) {
	db := openDepartmentTestDB(t)
	ctx := context.Background()
	depts := NewDepartmentRepository(db)
	root := department.Department{Code: "hq", Name: "HQ", IsEnabled: true}
	if err := depts.Save(ctx, &root); err != nil {
		t.Fatal(err)
	}
	rd := department.Department{ParentID: &root.ID, Code: "rd", Name: "R&D", IsEnabled: true}
	sales := department.Department{ParentID: &root.ID, Code: "sales", Name: "Sales", IsEnabled: true}
	for _, d := range []*department.Department{&rd, &sales} {
		if err := depts.Save(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	users := []models.UserModel{
		{Username: "alice", AccountSource: "INTERNAL", DepartmentID: &rd.ID},
		{Username: "bob", AccountSource: "INTERNAL", DepartmentID: &sales.ID},
		{Username: "carol", AccountSource: "INTERNAL"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	userRepo := NewUserRepository(db)

	// 用户表自身按部门过滤；本人（carol，无部门）始终可见。
	scope := query.UserScope("id", false, users[2].ID, []uint{rd.ID})
	list, total, err := userRepo.Gets(ctx, 1, 10, "id asc", *scope)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, u := range list {
		names = append(names, u.Username)
	}
	if total != 2 || !slices.Equal(names, []string{"alice", "carol"}) {
		t.Fatalf("scoped users = %v (total %d), want alice + carol", names, total)
	}

	// 其它按用户归属的表经子查询过滤。
	history := NewLoginHistoryRepository(db)
	for _, u := range users {
		if err := history.Create(ctx, &login_history.LoginHistory{UserID: u.ID, Username: u.Username, Success: true}); err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err := history.Gets(ctx, 1, 10, "id asc", *query.UserScope("user_id", false, 0, []uint{sales.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || rows[0].Username != "bob" {
		t.Fatalf("scoped login history = %+v, want only bob", rows)
	}

	counts, err := userRepo.CountByDepartment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts[rd.ID] != 1 || counts[sales.ID] != 1 || counts[root.ID] != 0 {
		t.Fatalf("CountByDepartment = %v", counts)
	}

	// 有成员的部门被外键拦住；有下级的部门同样删不掉。
	if err := depts.Delete(ctx, rd.ID); err == nil {
		t.Fatal("a department with members must not be deletable")
	}
	if err := depts.Delete(ctx, root.ID); err == nil {
		t.Fatal("a department with children must not be deletable")
	}

	// CUSTOM 范围的部门随角色读写；改成其它范围时部门清空。
	roles := NewRoleRepository(db)
	role := &permission.Role{Code: "scoped", Name: "Scoped", IsEnabled: true, DataScope: permission.DataScopeCustom, DepartmentIDs: []uint{sales.ID, rd.ID}}
	if err := roles.Create(ctx, role); err != nil {
		t.Fatal(err)
	}
	loaded, err := roles.GetByCode(ctx, "scoped")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DataScope != permission.DataScopeCustom || !slices.Equal(loaded.DepartmentIDs, permission.NormalizeIDs([]uint{sales.ID, rd.ID})) {
		t.Fatalf("custom scope not persisted: %+v", loaded)
	}
	loaded.DataScope, loaded.DepartmentIDs = permission.DataScopeDept, nil
	if err := roles.Update(ctx, loaded); err != nil {
		t.Fatal(err)
	}
	all, err := roles.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].DataScope != permission.DataScopeDept || len(all[0].DepartmentIDs) != 0 {
		t.Fatalf("switching away from CUSTOM must drop departments: %+v", all)
	}

	// 部门删除时，CUSTOM 引用随之删除，不阻塞删除。
	if err := roles.Update(ctx, &permission.Role{ID: all[0].ID, Code: "scoped", Name: "Scoped", IsEnabled: true, DataScope: permission.DataScopeCustom, DepartmentIDs: []uint{root.ID}}); err != nil {
		t.Fatal(err)
	}
	empty := department.Department{ParentID: &root.ID, Code: "empty", Name: "Empty"}
	if err := depts.Save(ctx, &empty); err != nil {
		t.Fatal(err)
	}
	if err := roles.Update(ctx, &permission.Role{ID: all[0].ID, Code: "scoped", Name: "Scoped", IsEnabled: true, DataScope: permission.DataScopeCustom, DepartmentIDs: []uint{empty.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := depts.Delete(ctx, empty.ID); err != nil {
		t.Fatalf("deleting a department referenced by a custom scope: %v", err)
	}
	after, err := roles.GetByCode(ctx, "scoped")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.DepartmentIDs) != 0 {
		t.Fatalf("custom scope must drop the deleted department: %+v", after.DepartmentIDs)
	}
}
