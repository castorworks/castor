package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
)

func withID(id uint, d department.Department) department.Department {
	d.ID = id
	return d
}

func newDepartmentFixture(t *testing.T, depth int) (*departmentService, *memDepartments, *mockUserRepo) {
	t.Helper()
	repo := &memDepartments{}
	var parent *uint
	for i := 1; i <= depth; i++ {
		d := department.Department{ParentID: parent, Code: "d" + string(rune('a'+i-1)), Name: "D"}
		d.ID = uint(i)
		repo.items = append(repo.items, d)
		id := d.ID
		parent = &id
	}
	extraRoot := department.Department{Code: "x", Name: "X"}
	extraRoot.ID = 20
	extraChild := department.Department{ParentID: deptID(20), Code: "y", Name: "Y"}
	extraChild.ID = 21
	repo.items = append(repo.items, extraRoot, extraChild)
	users := newMockUserRepo()
	return &departmentService{repo: repo, users: users}, repo, users
}

func TestDepartmentService_Save(t *testing.T) {
	t.Parallel()
	all := permission.AccessScope{All: true}
	ctx := context.Background()
	cases := []struct {
		name  string
		scope permission.AccessScope
		dept  department.Department
		want  error
	}{
		{"root department", all, department.Department{Code: "new", Name: "New"}, nil},
		{"child department", all, department.Department{ParentID: deptID(2), Code: "new", Name: "New"}, nil},
		{"code must be lowercase", all, department.Department{Code: "New", Name: "New"}, apperror.ErrInvalidDepartment},
		{"name is required", all, department.Department{Code: "new", Name: "  "}, apperror.ErrInvalidDepartment},
		{"code must be unique", all, department.Department{Code: "db", Name: "Dup"}, apperror.ErrDepartmentConflict},
		{"parent must exist", all, department.Department{ParentID: deptID(99), Code: "new", Name: "New"}, apperror.ErrDepartmentHierarchy},
		{"no ninth level", all, department.Department{ParentID: deptID(8), Code: "new", Name: "New"}, apperror.ErrDepartmentHierarchy},
		{"cannot move under a descendant", all, withID(2, department.Department{ParentID: deptID(5), Code: "db", Name: "D"}), apperror.ErrDepartmentHierarchy},
		{"cannot be its own parent", all, withID(3, department.Department{ParentID: deptID(3), Code: "dc", Name: "D"}), apperror.ErrDepartmentHierarchy},
		// 部门 2 带着 7 层子树（第 2～8 层）挂到第 2 层的 21 下，最深处会到第 9 层。
		{"moving a subtree may not push it past 8 levels", all, withID(2, department.Department{ParentID: deptID(21), Code: "db", Name: "D"}), apperror.ErrDepartmentHierarchy},
		{"moving a shallow subtree elsewhere is fine", all, withID(8, department.Department{ParentID: deptID(21), Code: "dh", Name: "D"}), nil},
		{"scoped caller cannot create a root", permission.AccessScope{DepartmentIDs: []uint{2}}, department.Department{Code: "new", Name: "New"}, apperror.ErrDataScopeExceedsCaller},
		{"scoped caller creates inside the scope", permission.AccessScope{DepartmentIDs: []uint{2}}, department.Department{ParentID: deptID(2), Code: "new", Name: "New"}, nil},
		{"scoped caller cannot attach outside the scope", permission.AccessScope{DepartmentIDs: []uint{2}}, department.Department{ParentID: deptID(1), Code: "new", Name: "New"}, apperror.ErrDataScopeExceedsCaller},
		{"scoped caller cannot edit outside the scope", permission.AccessScope{DepartmentIDs: []uint{2}}, withID(1, department.Department{Code: "da", Name: "Renamed"}), apperror.ErrDepartmentNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc, _, _ := newDepartmentFixture(t, 8)
			d := tc.dept
			err := svc.Save(ctx, tc.scope, &d)
			if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
				t.Fatalf("Save() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestDepartmentService_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	all := permission.AccessScope{All: true}

	svc, repo, users := newDepartmentFixture(t, 3)
	if err := svc.Delete(ctx, all, 2); !errors.Is(err, apperror.ErrDepartmentHasChildren) {
		t.Fatalf("deleting a department with children: %v", err)
	}
	users.users["eve"] = &user.User{ID: 9, Username: "eve", DepartmentID: deptID(3)}
	if err := svc.Delete(ctx, all, 3); !errors.Is(err, apperror.ErrDepartmentHasMembers) {
		t.Fatalf("deleting a department with members: %v", err)
	}
	delete(users.users, "eve")
	if err := svc.Delete(ctx, permission.AccessScope{DepartmentIDs: []uint{1}}, 3); !errors.Is(err, apperror.ErrDepartmentNotFound) {
		t.Fatalf("deleting outside the scope: %v", err)
	}
	if err := svc.Delete(ctx, all, 3); err != nil {
		t.Fatalf("deleting an empty leaf: %v", err)
	}
	if len(repo.items) != 4 {
		t.Fatalf("expected 4 departments left, got %d", len(repo.items))
	}

	list, err := svc.List(ctx, permission.AccessScope{DepartmentIDs: []uint{2}})
	if err != nil {
		t.Fatal(err)
	}
	if list[0].InScope || !list[1].InScope {
		t.Fatalf("InScope flags wrong: %+v", list)
	}
}
