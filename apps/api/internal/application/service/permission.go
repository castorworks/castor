package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/pkg/query"
)

var (
	ErrResourceNotFound           = apperror.ErrResourceNotFound
	ErrResourceCodeExists         = apperror.ErrResourceCodeExists
	ErrInvalidResource            = apperror.ErrInvalidResource
	ErrResourceConflict           = apperror.ErrResourcePermissionConflict
	ErrSystemResourceDelete       = apperror.ErrSystemResourceDelete
	ErrCannotModifySystemResource = apperror.ErrCannotModifySystemResource
	ErrRoleNotFound               = apperror.ErrRoleNotFound
	ErrRoleCodeExists             = apperror.ErrRoleCodeExists
	ErrInvalidRole                = apperror.ErrInvalidRole
	ErrSystemRoleDelete           = apperror.ErrSystemRoleDelete
)

// defaultResourceOrder 让资源目录默认按人工维护的展示顺序返回，
// 客户端显式传 order 时以客户端为准。
const defaultResourceOrder = "sort_order asc, id asc"

var (
	resourceCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(?::[a-z0-9][a-z0-9-]*)+$`)
	roleCodePattern     = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// PermissionService manages the RBAC object, operation and role catalog.
// Assignment, hierarchy, constraints and sessions live in RBACService.
type PermissionService interface {
	GetResources(ctx context.Context, page, size int, order string, opts ...query.Option) ([]permission.Resource, int64, error)
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
	// SetRoleDataScope 设置角色的数据范围；CUSTOM 至少指定一个存在的部门，其它范围不保留部门。
	// 越权检查（RBACService.EnsureCanSetRoleDataScope）由调用方先做。
	SetRoleDataScope(ctx context.Context, roleCode string, scope permission.DataScope, departmentIDs []uint) (*permission.Role, error)
}

type permissionService struct {
	resourceRepo permission.ResourceRepository
	roleRepo     permission.RoleRepository
	authzRepo    permission.AuthorizationRepository
	departments  department.Repository
}

func NewPermissionService(resourceRepo permission.ResourceRepository, roleRepo permission.RoleRepository, authzRepo permission.AuthorizationRepository, departments department.Repository) PermissionService {
	return &permissionService{resourceRepo: resourceRepo, roleRepo: roleRepo, authzRepo: authzRepo, departments: departments}
}

func (s *permissionService) SetRoleDataScope(ctx context.Context, roleCode string, scope permission.DataScope, departmentIDs []uint) (*permission.Role, error) {
	if !scope.Valid() {
		return nil, apperror.ErrInvalidDataScope
	}
	ids := permission.NormalizeIDs(departmentIDs)
	if scope != permission.DataScopeCustom {
		ids = []uint{}
	} else if len(ids) == 0 {
		return nil, apperror.ErrInvalidDataScope
	}
	var role *permission.Role
	err := s.authzRepo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		var err error
		role, err = s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
		if err != nil {
			return ErrRoleNotFound
		}
		if role.IsSystem {
			return apperror.ErrCannotModifySystemRole
		}
		if len(ids) > 0 {
			all, err := s.departments.List(ctx)
			if err != nil {
				return err
			}
			known := make(map[uint]bool, len(all))
			for _, d := range all {
				known[d.ID] = true
			}
			for _, id := range ids {
				if !known[id] {
					return apperror.ErrDepartmentNotFound
				}
			}
		}
		role.DataScope, role.DepartmentIDs = scope, ids
		// 必须经事务内的仓储写：LockRoles 已在本事务里锁住 rbac_roles，
		// 换一个连接去 UPDATE 同一行会一直等这把锁，形成跨连接的自锁死。
		return tx.SetRoleDataScope(ctx, role.ID, scope, ids)
	})
	return role, err
}

// GetResources 走统一的分页/筛选/排序契约：筛选与排序由 handler 的 GenericGets
// 从 `{field}-{op}=` 与 `order=` 解析并按实体字段白名单校验后传入，
// 服务层不再自己拼条件，避免绕开 query.NormalizeOrder 的排序注入防护。
func (s *permissionService) GetResources(ctx context.Context, page, size int, order string, opts ...query.Option) ([]permission.Resource, int64, error) {
	if strings.TrimSpace(order) == "" {
		order = defaultResourceOrder
	}
	return s.resourceRepo.Gets(ctx, page, size, order, opts...)
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
		// System resources are the RBAC anchor: their Path/Actions are the
		// identity that existing grants resolve against. Allowing them to be
		// repointed would let a resource-catalog editor redirect their own
		// grant onto a privileged route (privilege escalation), so freeze both.
		if existing.IsSystem {
			if resource.Path != existing.Path || !equalActions(resource.Actions, existing.Actions) {
				return ErrCannotModifySystemResource
			}
		}
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
	// 新角色默认只看本人：更宽的范围要经 SetRoleDataScope 的越权检查单独授予。
	role.DataScope, role.DepartmentIDs = permission.DataScopeSelf, nil
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

// equalActions reports whether two action lists contain the same set of actions,
// independent of order or duplicates.
func equalActions(a, b permission.StringSlice) bool {
	set := make(map[string]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	seen := make(map[string]struct{}, len(b))
	for _, x := range b {
		if _, ok := set[x]; !ok {
			return false
		}
		seen[x] = struct{}{}
	}
	return len(seen) == len(set)
}
