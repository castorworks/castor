package service

import (
	"context"
	"strings"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
)

// permKey identifies a single (resource path, HTTP action) permission.
func permKey(path, action string) string {
	return path + "\x00" + strings.ToUpper(strings.TrimSpace(action))
}

// callerPermissionSet returns the caller's full authorized permission set as a
// set of "path\x00ACTION" keys. It is the ceiling for anything the caller may
// grant to others: a full admin authorizes every resource and therefore may
// grant anything, while a delegated account is bounded by what it holds.
func (s *rbacService) callerPermissionSet(ctx context.Context, callerUsername string) (map[string]struct{}, error) {
	snap, err := s.GetUserAccess(ctx, callerUsername)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(snap.Permissions))
	for _, p := range snap.Permissions {
		set[permKey(p.ResourcePath, p.Action)] = struct{}{}
	}
	return set, nil
}

// roleEffectivePermissionSet returns the effective (hierarchy-closed) permission
// set that assigning the given role would confer.
func (s *rbacService) roleEffectivePermissionSet(ctx context.Context, roleCode string) (map[string]struct{}, error) {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return nil, apperror.ErrRoleNotFound
	}
	roles, edges, _, err := s.loadModel(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	closure := closureSet([]uint{role.ID}, edges, enabledRoleSet(roles))
	perms, err := s.effectivePermissions(ctx, closure, roles)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[permKey(p.ResourcePath, p.Action)] = struct{}{}
	}
	return set, nil
}

// EnsureCanAssignRole verifies the caller may grant roleCode to targetUsername:
// the role's effective permissions must be a subset of the caller's own. This
// blocks a delegated role manager from assigning a role (to anyone, including
// themselves) that carries permissions the caller does not hold.
func (s *rbacService) EnsureCanAssignRole(ctx context.Context, callerUsername, targetUsername, roleCode string) error {
	callerSet, err := s.callerPermissionSet(ctx, callerUsername)
	if err != nil {
		return err
	}
	roleSet, err := s.roleEffectivePermissionSet(ctx, roleCode)
	if err != nil {
		return err
	}
	for key := range roleSet {
		if _, ok := callerSet[key]; !ok {
			return apperror.ErrGrantExceedsCaller
		}
	}
	// 数据范围同理：角色授给目标用户后带来的范围（按目标所在部门展开）不得超出调用者。
	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		return err
	}
	target, err := s.userRepo.GetByUsername(ctx, targetUsername)
	if err != nil {
		return err
	}
	callerScope, err := s.authorizedScope(ctx, caller)
	if err != nil {
		return err
	}
	roleScope, err := s.roleScopeFor(ctx, target, roleCode)
	if err != nil {
		return err
	}
	if !roleScope.Within(callerScope) {
		return apperror.ErrDataScopeExceedsCaller
	}
	return nil
}

// EnsureCanSetRolePermissions verifies the caller may replace roleCode's grants:
// system roles are seed-managed and rejected outright, and every granted
// (resource, action) must be within the caller's own permission set.
func (s *rbacService) EnsureCanSetRolePermissions(ctx context.Context, callerUsername, roleCode string, grants []permission.PermissionGrant) error {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return apperror.ErrRoleNotFound
	}
	if role.IsSystem {
		return apperror.ErrCannotModifySystemRole
	}
	callerSet, err := s.callerPermissionSet(ctx, callerUsername)
	if err != nil {
		return err
	}
	resources, err := s.resourceRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	pathByID := make(map[uint]string, len(resources))
	for _, r := range resources {
		pathByID[r.ID] = r.Path
	}
	for _, grant := range grants {
		path, ok := pathByID[grant.ResourceID]
		if !ok {
			return apperror.ErrResourceNotFound
		}
		for _, action := range grant.Actions {
			if _, ok := callerSet[permKey(path, action)]; !ok {
				return apperror.ErrGrantExceedsCaller
			}
		}
	}
	return nil
}

// EnsureCanManageUser verifies the caller may take over targetID's account: every
// permission the target is authorized for must also be authorized for the caller.
// Resetting a password, re-enabling a locked account or kicking sessions of a more
// privileged user would otherwise let a delegated user manager assume permissions it
// was never granted. Account state (disabled, locked, expired) is deliberately not
// checked, so a manager can still unlock an account within its own ceiling.
func (s *rbacService) EnsureCanManageUser(ctx context.Context, callerID, targetID uint) error {
	if callerID == targetID {
		return nil
	}
	callerSet, err := s.authorizedPermissionSet(ctx, callerID)
	if err != nil {
		return err
	}
	targetSet, err := s.authorizedPermissionSet(ctx, targetID)
	if err != nil {
		return err
	}
	for key := range targetSet {
		if _, ok := callerSet[key]; !ok {
			return apperror.ErrTargetUserExceedsCaller
		}
	}
	// 接管一个数据范围更宽的账号，同样等于越过自己的数据范围。
	caller, err := s.userRepo.Get(ctx, callerID)
	if err != nil {
		return err
	}
	target, err := s.userRepo.Get(ctx, targetID)
	if err != nil {
		return err
	}
	callerScope, err := s.authorizedScope(ctx, caller)
	if err != nil {
		return err
	}
	targetScope, err := s.authorizedScope(ctx, target)
	if err != nil {
		return err
	}
	if !targetScope.Within(callerScope) {
		return apperror.ErrTargetUserExceedsCaller
	}
	return nil
}

// authorizedPermissionSet returns every permission userID is authorized for through its
// assigned roles and their juniors, regardless of the user's account state.
func (s *rbacService) authorizedPermissionSet(ctx context.Context, userID uint) (map[string]struct{}, error) {
	roles, edges, _, err := s.loadModel(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	assignments, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	authorized := closureSet(assignmentRoleIDs(assignments), edges, enabledRoleSet(roles))
	perms, err := s.effectivePermissions(ctx, authorized, roles)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[permKey(p.ResourcePath, p.Action)] = struct{}{}
	}
	return set, nil
}
