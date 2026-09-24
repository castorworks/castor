package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
)

// guardRepo backs the escalation-guard tests with mutable role/permission state.
type guardRepo struct {
	permission.AuthorizationRepository
	assignments map[uint][]permission.UserRole
	grants      []permission.RolePermission
}

func (r *guardRepo) GetUserRoles(_ context.Context, id uint) ([]permission.UserRole, error) {
	return r.assignments[id], nil
}
func (r *guardRepo) GetAllRolePermissions(context.Context) ([]permission.RolePermission, error) {
	return r.grants, nil
}
func (r *guardRepo) GetRoleHierarchies(context.Context) ([]permission.RoleHierarchy, error) {
	return nil, nil
}
func (r *guardRepo) GetConstraints(context.Context) ([]permission.SeparationConstraint, error) {
	return nil, nil
}

type guardRoles struct {
	permission.RoleRepository
	roles []permission.Role
}

func (r *guardRoles) GetAll(context.Context) ([]permission.Role, error) { return r.roles, nil }
func (r *guardRoles) GetByCode(_ context.Context, code string) (*permission.Role, error) {
	for i := range r.roles {
		if strings.EqualFold(r.roles[i].Code, code) {
			return &r.roles[i], nil
		}
	}
	return nil, apperror.ErrRoleNotFound
}

type guardResources struct {
	permission.ResourceRepository
	resources []permission.Resource
}

func (r *guardResources) GetAll(context.Context) ([]permission.Resource, error) {
	return r.resources, nil
}
func (r *guardResources) GetAllEnabled(context.Context) ([]permission.Resource, error) {
	return r.resources, nil
}

type guardUsers struct {
	user.Repository
	byName map[string]*user.User
}

func (r *guardUsers) GetByUsername(_ context.Context, name string) (*user.User, error) {
	if u, ok := r.byName[name]; ok {
		return u, nil
	}
	return nil, apperror.ErrRecordNotFound
}

func (r *guardUsers) Get(_ context.Context, id uint) (*user.User, error) {
	for _, u := range r.byName {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, apperror.ErrRecordNotFound
}

// caller "manager" holds GET on /reports only. Role "editor" additionally holds
// POST; role "viewer" holds only GET.
func newGuardService() RBACService {
	const route = "/api/v1/admin/reports/:id"
	repo := &guardRepo{
		assignments: map[uint][]permission.UserRole{7: {{UserID: 7, RoleID: 1}}},
		grants: []permission.RolePermission{
			{RoleID: 1, ResourceID: 1, Action: "GET"}, // manager role
			{RoleID: 2, ResourceID: 1, Action: "GET"}, // editor role
			{RoleID: 2, ResourceID: 1, Action: "POST"},
			{RoleID: 3, ResourceID: 1, Action: "GET"}, // viewer role
		},
	}
	roles := &guardRoles{roles: []permission.Role{
		{ID: 1, Code: "manager", IsEnabled: true},
		{ID: 2, Code: "editor", IsEnabled: true},
		{ID: 3, Code: "viewer", IsEnabled: true},
		{ID: 4, Code: "admin", IsEnabled: true, IsSystem: true},
	}}
	resources := &guardResources{resources: []permission.Resource{
		{ID: 1, Path: route, Actions: permission.StringSlice{"GET", "POST"}, IsEnabled: true},
	}}
	users := &guardUsers{byName: map[string]*user.User{
		"manager": {ID: 7, Username: "manager", Enable: true},
		"editor":  {ID: 8, Username: "editor", Enable: true},
		"viewer":  {ID: 9, Username: "viewer", Enable: true},
		"target":  {ID: 10, Username: "target", Enable: true},
	}}
	return NewRBACService(repo, roles, resources, users, &memDepartments{})
}

func TestEnsureCanAssignRole(t *testing.T) {
	svc := newGuardService()
	ctx := context.Background()

	if err := svc.EnsureCanAssignRole(ctx, "manager", "target", "editor"); !errors.Is(err, apperror.ErrGrantExceedsCaller) {
		t.Fatalf("granting a role with POST (caller lacks it) should be rejected, got %v", err)
	}
	// Self-assignment of an over-privileged role is equally blocked.
	if err := svc.EnsureCanAssignRole(ctx, "manager", "manager", "editor"); !errors.Is(err, apperror.ErrGrantExceedsCaller) {
		t.Fatalf("self-assigning an over-privileged role should be rejected, got %v", err)
	}
	if err := svc.EnsureCanAssignRole(ctx, "manager", "target", "viewer"); err != nil {
		t.Fatalf("granting a role within the caller's permissions should be allowed, got %v", err)
	}
}

func TestEnsureCanSetRolePermissions(t *testing.T) {
	svc := newGuardService()
	ctx := context.Background()

	if err := svc.EnsureCanSetRolePermissions(ctx, "manager", "admin", nil); !errors.Is(err, apperror.ErrCannotModifySystemRole) {
		t.Fatalf("editing a system role must be rejected, got %v", err)
	}
	over := []permission.PermissionGrant{{ResourceID: 1, Actions: []string{"POST"}}}
	if err := svc.EnsureCanSetRolePermissions(ctx, "manager", "editor", over); !errors.Is(err, apperror.ErrGrantExceedsCaller) {
		t.Fatalf("granting POST beyond the caller must be rejected, got %v", err)
	}
	within := []permission.PermissionGrant{{ResourceID: 1, Actions: []string{"GET"}}}
	if err := svc.EnsureCanSetRolePermissions(ctx, "manager", "editor", within); err != nil {
		t.Fatalf("granting within the caller's permissions should be allowed, got %v", err)
	}
}

func TestEnsureCanManageUser(t *testing.T) {
	svc := newGuardService()
	repo := svc.(*rbacService).repo.(*guardRepo)
	// 8 = editor (GET+POST), 9 = viewer (GET), 10 = no roles
	repo.assignments[8] = []permission.UserRole{{UserID: 8, RoleID: 2}}
	repo.assignments[9] = []permission.UserRole{{UserID: 9, RoleID: 3}}
	ctx := context.Background()

	if err := svc.EnsureCanManageUser(ctx, 7, 8); !errors.Is(err, apperror.ErrTargetUserExceedsCaller) {
		t.Fatalf("managing a user who holds POST (caller lacks it) must be rejected, got %v", err)
	}
	if err := svc.EnsureCanManageUser(ctx, 7, 9); err != nil {
		t.Fatalf("managing a user within the caller's permissions should be allowed, got %v", err)
	}
	if err := svc.EnsureCanManageUser(ctx, 7, 10); err != nil {
		t.Fatalf("managing a user without roles should be allowed, got %v", err)
	}
	if err := svc.EnsureCanManageUser(ctx, 8, 7); err != nil {
		t.Fatalf("a more privileged caller may manage a less privileged user, got %v", err)
	}
	if err := svc.EnsureCanManageUser(ctx, 7, 7); err != nil {
		t.Fatalf("managing oneself never exceeds oneself, got %v", err)
	}
}
