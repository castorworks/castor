package service

import (
	"context"
	"errors"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
)

// SessionScope 返回会话当前激活角色（含其下级角色）展开后的数据范围，用于列表过滤和
// 单条记录的可见性判断——与权限一样遵循 RBAC3 的最小激活原则。
func (s *rbacService) SessionScope(ctx context.Context, sessionID string, userID uint) (permission.AccessScope, error) {
	session, err := s.validSession(ctx, sessionID, userID)
	if err != nil {
		return permission.AccessScope{}, err
	}
	u, err := s.userRepo.Get(ctx, session.UserID)
	if err != nil {
		return permission.AccessScope{}, err
	}
	active, err := s.repo.GetSessionRoleIDs(ctx, session.ID)
	if err != nil {
		return permission.AccessScope{}, err
	}
	return s.scopeForRoles(ctx, u, active)
}

// authorizedScope 是用户全部已分配角色（含下级角色）的数据范围，用作授出与接管时的上限，
// 与 callerPermissionSet 采用同一口径。
func (s *rbacService) authorizedScope(ctx context.Context, u *user.User) (permission.AccessScope, error) {
	assignments, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return permission.AccessScope{}, err
	}
	return s.scopeForRoles(ctx, u, assignmentRoleIDs(assignments))
}

// scopeForRoles 对用户 u 展开一组角色的数据范围：取并集，任一角色为 ALL 即全部；
// 未知或未声明的范围按 SELF 处理（失败即收紧）。
func (s *rbacService) scopeForRoles(ctx context.Context, u *user.User, seed []uint) (permission.AccessScope, error) {
	roles, edges, _, err := s.loadModel(ctx, s.repo)
	if err != nil {
		return permission.AccessScope{}, err
	}
	byID := roleMap(roles)
	scope := permission.AccessScope{UserID: u.ID}
	var departments []uint
	var tree []department.Department
	treeLoaded := false
	for id := range closureSet(seed, edges, enabledRoleSet(roles)) {
		role := byID[id]
		switch role.DataScope {
		case permission.DataScopeAll:
			return permission.AccessScope{All: true, UserID: u.ID}, nil
		case permission.DataScopeDept:
			if u.DepartmentID != nil {
				departments = append(departments, *u.DepartmentID)
			}
		case permission.DataScopeDeptTree:
			if u.DepartmentID == nil {
				continue
			}
			if !treeLoaded {
				if tree, err = s.departments.List(ctx); err != nil {
					return permission.AccessScope{}, err
				}
				treeLoaded = true
			}
			departments = append(departments, department.Subtree(tree, *u.DepartmentID)...)
		case permission.DataScopeCustom:
			departments = append(departments, role.DepartmentIDs...)
		}
	}
	scope.DepartmentIDs = permission.NormalizeIDs(departments)
	return scope, nil
}

// roleScopeFor 是把单个角色授给用户 u 后，该角色为 u 带来的数据范围。
func (s *rbacService) roleScopeFor(ctx context.Context, u *user.User, roleCode string) (permission.AccessScope, error) {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return permission.AccessScope{}, apperror.ErrRoleNotFound
		}
		return permission.AccessScope{}, err
	}
	return s.scopeForRoles(ctx, u, []uint{role.ID})
}

// EnsureCanSetRoleDataScope 检查调用者能否把角色的数据范围改为 scope：系统角色由种子管理，
// 一律拒绝；SELF 只会收窄，总是允许；CUSTOM 的部门必须都在调用者自己的范围内；
// ALL 与按成员所在部门展开的 DEPT / DEPT_AND_CHILDREN 无法逐一核对成员，要求调用者本身是 ALL。
func (s *rbacService) EnsureCanSetRoleDataScope(ctx context.Context, callerUsername, roleCode string, scope permission.DataScope, departmentIDs []uint) error {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return apperror.ErrRoleNotFound
	}
	if role.IsSystem {
		return apperror.ErrCannotModifySystemRole
	}
	if scope == permission.DataScopeSelf {
		return nil
	}
	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		return err
	}
	callerScope, err := s.authorizedScope(ctx, caller)
	if err != nil {
		return err
	}
	if scope == permission.DataScopeCustom {
		if (permission.AccessScope{DepartmentIDs: departmentIDs}).Within(callerScope) {
			return nil
		}
		return apperror.ErrDataScopeExceedsCaller
	}
	if !callerScope.All {
		return apperror.ErrDataScopeExceedsCaller
	}
	return nil
}

// EnsureUserVisible 确认用户名对应的用户在 scope 内；不存在与范围外同样返回 ErrRecordNotFound，
// 按用户名操作的授权接口据此避免暴露或改动范围外的账号。
func (s *rbacService) EnsureUserVisible(ctx context.Context, scope permission.AccessScope, username string) error {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if errors.Is(err, shared.ErrNotFound) {
		return apperror.ErrRecordNotFound
	}
	if err != nil {
		return err
	}
	if !scope.Contains(u.ID, u.DepartmentID) {
		return apperror.ErrRecordNotFound
	}
	return nil
}
