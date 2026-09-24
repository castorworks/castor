package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
)

// These repositories expose mutable database state to the real session access
// and authorization services, without a second policy store or reload step.
type authorizationState struct {
	permission.AuthorizationRepository
	session     permission.AuthorizationSession
	active      []uint
	assignments []permission.UserRole
	grants      []permission.RolePermission
	edges       []permission.RoleHierarchy
	constraints []permission.SeparationConstraint
	sessionErr  error
	grantErr    error
}

func (r *authorizationState) GetSession(context.Context, string) (*permission.AuthorizationSession, error) {
	return &r.session, r.sessionErr
}
func (r *authorizationState) GetSessionRoleIDs(context.Context, string) ([]uint, error) {
	return r.active, nil
}
func (r *authorizationState) GetUserRoles(context.Context, uint) ([]permission.UserRole, error) {
	return r.assignments, nil
}
func (r *authorizationState) GetAllRolePermissions(context.Context) ([]permission.RolePermission, error) {
	return r.grants, r.grantErr
}
func (r *authorizationState) GetRoleHierarchies(context.Context) ([]permission.RoleHierarchy, error) {
	return r.edges, nil
}
func (r *authorizationState) GetConstraints(context.Context) ([]permission.SeparationConstraint, error) {
	return r.constraints, nil
}

type authorizationRoles struct {
	permission.RoleRepository
	roles []permission.Role
}

func (r *authorizationRoles) GetAll(context.Context) ([]permission.Role, error) { return r.roles, nil }

type authorizationResources struct {
	permission.ResourceRepository
	resources []permission.Resource
}

func (r *authorizationResources) GetAllEnabled(context.Context) ([]permission.Resource, error) {
	var enabled []permission.Resource
	for _, resource := range r.resources {
		if resource.IsEnabled {
			enabled = append(enabled, resource)
		}
	}
	return enabled, nil
}

type authorizationUser struct {
	user.Repository
	account user.User
}

func (r *authorizationUser) Get(context.Context, uint) (*user.User, error) { return &r.account, nil }

const reportRoute = "/api/v1/admin/reports/:id"

type authorizationFixture struct {
	svc       RBACService
	repo      *authorizationState
	roles     *authorizationRoles
	resources *authorizationResources
	users     *authorizationUser
}

func newAuthorizationFixture() *authorizationFixture {
	f := &authorizationFixture{
		repo: &authorizationState{
			session:     permission.AuthorizationSession{ID: "session", UserID: 7, ExpiresAt: time.Now().Add(time.Hour)},
			active:      []uint{1},
			assignments: []permission.UserRole{{UserID: 7, RoleID: 1}},
			grants:      []permission.RolePermission{{RoleID: 1, ResourceID: 1, Action: "GET"}},
		},
		roles:     &authorizationRoles{roles: []permission.Role{{ID: 1, Code: "reader", IsEnabled: true}}},
		resources: &authorizationResources{resources: []permission.Resource{{ID: 1, Path: reportRoute, Actions: permission.StringSlice{"GET"}, IsEnabled: true}}},
		users:     &authorizationUser{account: user.User{ID: 7, Username: "reader", Enable: true}},
	}
	f.svc = NewRBACService(f.repo, f.roles, f.resources, f.users, &memDepartments{})
	return f
}

func TestAuthorizeSessionUsesEffectivePermissions(t *testing.T) {
	dbErr := errors.New("database read failed")
	tests := []struct {
		name    string
		change  func(*authorizationFixture)
		route   string
		method  string
		allowed bool
		err     error
	}{
		{name: "direct grant", allowed: true},
		{name: "HTTP method must match", method: "POST"},
		{name: "HTTP method is case sensitive", method: "get"},
		{name: "static route is separate from parameter route", route: "/api/v1/admin/reports/export"},
		{name: "concrete URL is not a route template", route: "/api/v1/admin/reports/42"},
		{name: "parameter names must match", route: "/api/v1/admin/reports/:reportId"},
		{name: "route prefix grants nothing", route: "/api/v1/admin/reports"},
		{name: "trailing slash is a different template", route: reportRoute + "/"},
		{name: "empty active roles", change: func(f *authorizationFixture) { f.repo.active = nil }},
		{name: "unassigned active role", change: func(f *authorizationFixture) { f.repo.assignments = nil }, err: apperror.ErrRoleNotAuthorized},
		{name: "disabled active role", change: func(f *authorizationFixture) { f.roles.roles[0].IsEnabled = false }, err: apperror.ErrRoleNotAuthorized},
		{name: "disabled resource", change: func(f *authorizationFixture) { f.resources.resources[0].IsEnabled = false }},
		{name: "deleted resource", change: func(f *authorizationFixture) { f.resources.resources = nil }},
		{name: "action removed from resource", change: func(f *authorizationFixture) { f.resources.resources[0].Actions = permission.StringSlice{"POST"} }},
		{name: "permission revoked", change: func(f *authorizationFixture) { f.repo.grants = nil }},
		{name: "admin has no implicit bypass", change: func(f *authorizationFixture) { f.roles.roles[0].Code = "admin"; f.repo.grants = nil }},
		{name: "wildcard is not expanded", change: func(f *authorizationFixture) { f.resources.resources[0].Path = "/api/v1/admin/*" }},
		{name: "missing session", change: func(f *authorizationFixture) { f.repo.sessionErr = shared.ErrNotFound }, err: apperror.ErrAuthorizationSessionNotFound},
		{name: "wrong session owner", change: func(f *authorizationFixture) { f.repo.session.UserID = 8 }, err: apperror.ErrAuthorizationSessionNotFound},
		{name: "expired session", change: func(f *authorizationFixture) { f.repo.session.ExpiresAt = time.Now().Add(-time.Hour) }, err: apperror.ErrAuthorizationSessionExpired},
		{name: "revoked session", change: func(f *authorizationFixture) { now := time.Now(); f.repo.session.RevokedAt = &now }, err: apperror.ErrAuthorizationSessionRevoked},
		{name: "disabled user", change: func(f *authorizationFixture) { f.users.account.Enable = false }, err: apperror.ErrUserDisabled},
		{name: "locked user", change: func(f *authorizationFixture) { f.users.account.Locked = true }, err: apperror.ErrUserLocked},
		{name: "expired user", change: func(f *authorizationFixture) { f.users.account.AccountExpireDate = time.Now().Add(-time.Hour) }, err: apperror.ErrAccountExpired},
		{name: "expired credentials", change: func(f *authorizationFixture) { f.users.account.CredentialExpireDate = time.Now().Add(-time.Hour) }, err: apperror.ErrCredentialExpired},
		{name: "database error fails closed", change: func(f *authorizationFixture) { f.repo.grantErr = dbErr }, err: dbErr},
		{name: "assigned but inactive role grants nothing", change: func(f *authorizationFixture) {
			f.roles.roles = append(f.roles.roles, permission.Role{ID: 2, Code: "approver", IsEnabled: true})
			f.repo.assignments = append(f.repo.assignments, permission.UserRole{UserID: 7, RoleID: 2})
			f.repo.grants[0].RoleID = 2
		}},
		{name: "inherited permission", allowed: true, change: func(f *authorizationFixture) {
			f.roles.roles = append(f.roles.roles, permission.Role{ID: 2, Code: "junior", IsEnabled: true})
			f.repo.edges = []permission.RoleHierarchy{{SeniorRoleID: 1, JuniorRoleID: 2}}
			f.repo.grants[0].RoleID = 2
		}},
		{name: "DSD includes inherited roles", err: apperror.ErrDSDViolation, change: func(f *authorizationFixture) {
			f.roles.roles = append(f.roles.roles, permission.Role{ID: 2, Code: "junior", IsEnabled: true})
			f.repo.edges = []permission.RoleHierarchy{{SeniorRoleID: 1, JuniorRoleID: 2}}
			f.repo.constraints = []permission.SeparationConstraint{{Type: permission.ConstraintTypeDSD, Cardinality: 2, RoleIDs: []uint{1, 2}, IsEnabled: true}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newAuthorizationFixture()
			if tt.change != nil {
				tt.change(f)
			}
			route, method := tt.route, tt.method
			if route == "" {
				route = reportRoute
			}
			if method == "" {
				method = "GET"
			}
			allowed, err := f.svc.AuthorizeSession(context.Background(), "session", 7, route, method)
			if allowed != tt.allowed || !errors.Is(err, tt.err) {
				t.Fatalf("got allowed=%v err=%v; want allowed=%v err=%v", allowed, err, tt.allowed, tt.err)
			}
		})
	}
}

func TestAuthorizeSessionTraversesDeepHierarchyAndObservesChanges(t *testing.T) {
	f := newAuthorizationFixture()
	for id := uint(2); id <= 16; id++ {
		f.roles.roles = append(f.roles.roles, permission.Role{ID: id, Code: fmt.Sprintf("role-%d", id), IsEnabled: true})
		f.repo.edges = append(f.repo.edges, permission.RoleHierarchy{SeniorRoleID: id - 1, JuniorRoleID: id})
	}
	f.repo.grants[0].RoleID = 16
	// Two service instances read the same authoritative state on every request.
	instances := []RBACService{f.svc, NewRBACService(f.repo, f.roles, f.resources, f.users, &memDepartments{})}
	check := func(want bool) {
		t.Helper()
		for i, svc := range instances {
			allowed, err := svc.AuthorizeSession(context.Background(), "session", 7, reportRoute, "GET")
			if err != nil || allowed != want {
				t.Fatalf("instance %d: allowed=%v err=%v; want %v", i, allowed, err, want)
			}
		}
	}
	check(true)
	f.roles.roles[7].IsEnabled = false
	check(false)
	f.roles.roles[7].IsEnabled = true
	check(true)
	edges := f.repo.edges
	f.repo.edges = nil
	check(false)
	f.repo.edges = edges
	check(true)
	grants := f.repo.grants
	f.repo.grants = nil
	check(false)
	f.repo.grants = grants
	check(true)
	f.resources.resources[0].IsEnabled = false
	check(false)
}
