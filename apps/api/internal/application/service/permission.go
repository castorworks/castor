package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/pkg/query"
)

var (
	ErrResourceNotFound     = apperror.ErrResourceNotFound
	ErrResourceCodeExists   = apperror.ErrResourceCodeExists
	ErrInvalidResource      = apperror.ErrInvalidResource
	ErrResourceConflict     = apperror.ErrResourcePermissionConflict
	ErrSystemResourceDelete = apperror.ErrSystemResourceDelete
	ErrRoleNotFound         = apperror.ErrRoleNotFound
	ErrRoleCodeExists       = apperror.ErrRoleCodeExists
	ErrInvalidRole          = apperror.ErrInvalidRole
	ErrSystemRoleDelete     = apperror.ErrSystemRoleDelete
)

var (
	resourceCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(?::[a-z0-9][a-z0-9-]*)+$`)
	roleCodePattern     = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// PermissionService manages the RBAC object, operation and role catalog.
// Assignment, hierarchy, constraints and sessions live in RBACService.
type PermissionService interface {
	GetResources(ctx context.Context, page, size int, category, module string, isEnabled *bool, search string) ([]permission.Resource, int64, error)
	GetResource(ctx context.Context, id uint) (*permission.Resource, error)
	CreateResource(ctx context.Context, resource *permission.Resource) error
	UpdateResource(ctx context.Context, resource *permission.Resource) error
	DeleteResource(ctx context.Context, id uint) error
	GetResourcesByCategory(ctx context.Context, category permission.ResourceCategory) ([]permission.Resource, error)
	GetResourcesByModule(ctx context.Context, module string) ([]permission.Resource, error)
	GetAllModules(ctx context.Context) ([]string, error)
	GetRoles(ctx context.Context) ([]permission.Role, error)
	GetRole(ctx context.Context, id uint) (*permission.Role, error)
	GetRoleByCode(ctx context.Context, code string) (*permission.Role, error)
	CreateRole(ctx context.Context, role *permission.Role) error
	UpdateRole(ctx context.Context, role *permission.Role) error
	DeleteRole(ctx context.Context, id uint) error
}

type permissionService struct {
	resourceRepo permission.ResourceRepository
	roleRepo     permission.RoleRepository
	authzRepo    permission.AuthorizationRepository
}

func NewPermissionService(resourceRepo permission.ResourceRepository, roleRepo permission.RoleRepository, authzRepo permission.AuthorizationRepository) PermissionService {
	return &permissionService{resourceRepo: resourceRepo, roleRepo: roleRepo, authzRepo: authzRepo}
}

func (s *permissionService) GetResources(ctx context.Context, page, size int, category, module string, isEnabled *bool, search string) ([]permission.Resource, int64, error) {
	var opts []query.Option
	if category != "" {
		opts = append(opts, query.Option{Condition: "category = ?", Args: []any{category}})
	}
	if module != "" {
		opts = append(opts, query.Option{Condition: "module = ?", Args: []any{module}})
	}
	if isEnabled != nil {
		opts = append(opts, query.Option{Condition: "is_enabled = ?", Args: []any{*isEnabled}})
	}
	if search != "" {
		like := "%" + search + "%"
		opts = append(opts, query.Option{Condition: "(name LIKE ? OR code LIKE ? OR path LIKE ?)", Args: []any{like, like, like}})
	}
	return s.resourceRepo.Gets(ctx, page, size, "sort_order ASC, id ASC", opts...)
}

func (s *permissionService) GetResource(ctx context.Context, id uint) (*permission.Resource, error) {
	return s.resourceRepo.Get(ctx, id)
}

func (s *permissionService) CreateResource(ctx context.Context, resource *permission.Resource) error {
	if err := normalizeResource(resource); err != nil {
		return err
	}
	resource.IsSystem = false
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		exists, err := s.resourceRepo.ExistsByCode(ctx, resource.Code)
		if err != nil {
			return err
		}
		if exists {
			return ErrResourceCodeExists
		}
		if err := s.validateResourceConflict(ctx, resource); err != nil {
			return err
		}
		return tx.CreateResource(ctx, resource)
	})
}

func (s *permissionService) UpdateResource(ctx context.Context, resource *permission.Resource) error {
	if resource == nil {
		return ErrInvalidResource
	}
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		existing, err := s.resourceRepo.Get(ctx, resource.ID)
		if err != nil {
			return ErrResourceNotFound
		}
		if err := normalizeResource(resource); err != nil {
			return err
		}
		resource.IsSystem = existing.IsSystem
		if resource.Code != existing.Code {
			exists, err := s.resourceRepo.ExistsByCode(ctx, resource.Code)
			if err != nil {
				return err
			}
			if exists {
				return ErrResourceCodeExists
			}
		}
		if err := s.validateResourceConflict(ctx, resource); err != nil {
			return err
		}
		if err := tx.UpdateResource(ctx, resource); err != nil {
			return err
		}
		return tx.PruneResourcePermissions(ctx, resource.ID, []string(resource.Actions))
	})
}

func (s *permissionService) DeleteResource(ctx context.Context, id uint) error {
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		resource, err := s.resourceRepo.Get(ctx, id)
		if err != nil {
			return ErrResourceNotFound
		}
		if resource.IsSystem {
			return ErrSystemResourceDelete
		}
		return tx.DeleteResource(ctx, id)
	})
}

func (s *permissionService) GetResourcesByCategory(ctx context.Context, category permission.ResourceCategory) ([]permission.Resource, error) {
	return s.resourceRepo.GetByCategory(ctx, category)
}

func (s *permissionService) GetResourcesByModule(ctx context.Context, module string) ([]permission.Resource, error) {
	return s.resourceRepo.GetByModule(ctx, module)
}

func (s *permissionService) GetAllModules(ctx context.Context) ([]string, error) {
	resources, err := s.resourceRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	modules := make([]string, 0)
	for _, resource := range resources {
		if resource.Module == "" {
			continue
		}
		if _, ok := seen[resource.Module]; !ok {
			seen[resource.Module] = struct{}{}
			modules = append(modules, resource.Module)
		}
	}
	return modules, nil
}

func (s *permissionService) GetRoles(ctx context.Context) ([]permission.Role, error) {
	return s.roleRepo.GetAll(ctx)
}

func (s *permissionService) GetRole(ctx context.Context, id uint) (*permission.Role, error) {
	return s.roleRepo.Get(ctx, id)
}

func (s *permissionService) GetRoleByCode(ctx context.Context, code string) (*permission.Role, error) {
	return s.roleRepo.GetByCode(ctx, canonicalRoleCode(code))
}

func (s *permissionService) CreateRole(ctx context.Context, role *permission.Role) error {
	if err := normalizeRole(role); err != nil {
		return err
	}
	role.IsSystem = false
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		exists, err := s.roleRepo.ExistsByCode(ctx, role.Code)
		if err != nil {
			return err
		}
		if exists {
			return ErrRoleCodeExists
		}
		return tx.CreateRole(ctx, role)
	})
}

func (s *permissionService) UpdateRole(ctx context.Context, role *permission.Role) error {
	if role == nil {
		return ErrInvalidRole
	}
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		existing, err := s.roleRepo.Get(ctx, role.ID)
		if err != nil {
			return ErrRoleNotFound
		}
		if err := normalizeRole(role); err != nil {
			return err
		}
		role.IsSystem = existing.IsSystem
		if role.Code != existing.Code {
			exists, err := s.roleRepo.ExistsByCode(ctx, role.Code)
			if err != nil {
				return err
			}
			if exists {
				return ErrRoleCodeExists
			}
		}
		if !existing.IsEnabled && role.IsEnabled {
			if err := s.validateRoleEnable(ctx, tx, role); err != nil {
				return err
			}
		}
		if err := tx.UpdateRole(ctx, role); err != nil {
			return err
		}
		if existing.IsEnabled && !role.IsEnabled {
			return tx.RevokeSessionsWithRole(ctx, role.ID)
		}
		return nil
	})
}

func (s *permissionService) validateRoleEnable(ctx context.Context, repo permission.AuthorizationRepository, changed *permission.Role) error {
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	for i := range roles {
		if roles[i].ID == changed.ID {
			roles[i] = *changed
		}
	}
	edges, err := repo.GetRoleHierarchies(ctx)
	if err != nil {
		return err
	}
	constraints, err := repo.GetConstraints(ctx)
	if err != nil {
		return err
	}
	assignments, err := repo.GetAllUserRoles(ctx)
	if err != nil {
		return err
	}
	byUser := make(map[uint][]uint)
	for _, assignment := range assignments {
		byUser[assignment.UserID] = append(byUser[assignment.UserID], assignment.RoleID)
	}
	for _, roleIDs := range byUser {
		if violates(roleIDs, roles, edges, constraints, permission.ConstraintTypeSSD) {
			return apperror.ErrSSDViolation
		}
	}
	sessions, err := repo.GetActiveSessions(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		roleIDs, err := repo.GetSessionRoleIDs(ctx, session.ID)
		if err != nil {
			return err
		}
		if violates(roleIDs, roles, edges, constraints, permission.ConstraintTypeDSD) {
			return apperror.ErrDSDViolation
		}
	}
	return nil
}

func (s *permissionService) DeleteRole(ctx context.Context, id uint) error {
	return s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		role, err := s.roleRepo.Get(ctx, id)
		if err != nil {
			return ErrRoleNotFound
		}
		if role.IsSystem {
			return ErrSystemRoleDelete
		}
		constraints, err := tx.GetConstraints(ctx)
		if err != nil {
			return err
		}
		for _, constraint := range constraints {
			for _, roleID := range constraint.RoleIDs {
				if roleID == id {
					return apperror.ErrRoleInUse
				}
			}
		}
		if err := tx.RevokeSessionsWithRole(ctx, id); err != nil {
			return err
		}
		return tx.DeleteRole(ctx, id)
	})
}

func normalizeResource(resource *permission.Resource) error {
	if resource == nil {
		return ErrInvalidResource
	}
	resource.Code = strings.ToLower(strings.TrimSpace(resource.Code))
	resource.Name = strings.TrimSpace(resource.Name)
	resource.Description = strings.TrimSpace(resource.Description)
	resource.Path = strings.TrimSpace(resource.Path)
	resource.Module = strings.TrimSpace(resource.Module)
	if len(resource.Path) > 1 {
		resource.Path = strings.TrimRight(resource.Path, "/")
	}
	if !resourceCodePattern.MatchString(resource.Code) ||
		utf8.RuneCountInString(resource.Code) > 100 ||
		utf8.RuneCountInString(resource.Name) < 2 || utf8.RuneCountInString(resource.Name) > 100 ||
		utf8.RuneCountInString(resource.Description) > 500 ||
		utf8.RuneCountInString(resource.Path) > 200 ||
		utf8.RuneCountInString(resource.Module) > 100 ||
		!strings.HasPrefix(resource.Path, "/api/v1/") ||
		strings.ContainsAny(resource.Path, "?# \t\r\n") ||
		(resource.Category != permission.CategoryAdmin && resource.Category != permission.CategoryUser) {
		return ErrInvalidResource
	}
	requested := make(map[string]struct{}, len(resource.Actions))
	for _, action := range resource.Actions {
		action = strings.ToUpper(strings.TrimSpace(action))
		if action == "" {
			return ErrInvalidResource
		}
		requested[action] = struct{}{}
	}
	resource.Actions = make(permission.StringSlice, 0, len(requested))
	for _, action := range permission.AllActions {
		if _, ok := requested[action]; ok {
			resource.Actions = append(resource.Actions, action)
			delete(requested, action)
		}
	}
	if len(resource.Actions) == 0 || len(requested) != 0 {
		return ErrInvalidResource
	}
	return nil
}

func (s *permissionService) validateResourceConflict(ctx context.Context, candidate *permission.Resource) error {
	resources, err := s.resourceRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	if resourcePermissionConflict(resources, candidate) {
		return ErrResourceConflict
	}
	return nil
}

func resourcePermissionConflict(resources []permission.Resource, candidate *permission.Resource) bool {
	actions := make(map[string]struct{}, len(candidate.Actions))
	for _, action := range candidate.Actions {
		actions[action] = struct{}{}
	}
	for _, resource := range resources {
		if resource.ID == candidate.ID || resource.Path != candidate.Path {
			continue
		}
		for _, action := range resource.Actions {
			if _, ok := actions[action]; ok {
				return true
			}
		}
	}
	return false
}

func normalizeRole(role *permission.Role) error {
	if role == nil {
		return ErrInvalidRole
	}
	role.Code = strings.ToLower(strings.TrimSpace(role.Code))
	role.Name = strings.TrimSpace(role.Name)
	role.Description = strings.TrimSpace(role.Description)
	if !roleCodePattern.MatchString(role.Code) ||
		utf8.RuneCountInString(role.Code) < 2 || utf8.RuneCountInString(role.Code) > 50 ||
		utf8.RuneCountInString(role.Name) < 2 || utf8.RuneCountInString(role.Name) > 100 ||
		utf8.RuneCountInString(role.Description) > 500 {
		return ErrInvalidRole
	}
	return nil
}

func canonicalRoleCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
