package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/google/uuid"
)

type RolePermissionDetail struct {
	ResourceID   uint     `json:"resourceId"`
	ResourceCode string   `json:"resourceCode"`
	ResourceName string   `json:"resourceName"`
	ResourcePath string   `json:"resourcePath"`
	Actions      []string `json:"actions"`
	IsSystem     bool     `json:"isSystem"`
}

type AccessSnapshot struct {
	SessionID       string                           `json:"sessionId,omitempty"`
	Username        string                           `json:"username"`
	AssignedRoles   []permission.Role                `json:"assignedRoles"`
	AuthorizedRoles []permission.Role                `json:"authorizedRoles"`
	ActiveRoles     []permission.Role                `json:"activeRoles"`
	Permissions     []permission.EffectivePermission `json:"permissions"`
}

// SessionOrigin 描述打开授权会话的那次登录，只用于在线会话的展示与审计，不参与授权判断。
type SessionOrigin struct {
	IP        string
	UserAgent string
	Remember  bool // 用户登录时勾选了"记住我"
}

// 会话来源字段的存储上限，与 rbac_sessions 列宽一致。
const (
	maxSessionIPLength        = 64
	maxSessionUserAgentLength = 512
)

type RBACService interface {
	GetRolePermissions(ctx context.Context, roleCode string) ([]RolePermissionDetail, error)
	SetRolePermissions(ctx context.Context, roleCode string, grants []permission.PermissionGrant) error
	GetRoleJuniors(ctx context.Context, roleCode string) ([]permission.Role, error)
	SetRoleJuniors(ctx context.Context, roleCode string, juniorRoleCodes []string) error
	GetConstraints(ctx context.Context) ([]permission.SeparationConstraint, error)
	CreateConstraint(ctx context.Context, constraint *permission.SeparationConstraint) error
	UpdateConstraint(ctx context.Context, id uint, constraint *permission.SeparationConstraint) error
	DeleteConstraint(ctx context.Context, id uint) error
	GetUserRoleCodes(ctx context.Context, username string) ([]string, error)
	// GetRoleUsernames 角色的成员中位于 scope 内的用户名（范围外的成员不出现，也不计数）
	GetRoleUsernames(ctx context.Context, scope permission.AccessScope, roleCode string) ([]string, error)
	AddUserRole(ctx context.Context, username, roleCode string, system bool) error
	EnsureCanAssignRole(ctx context.Context, callerUsername, targetUsername, roleCode string) error
	EnsureCanSetRolePermissions(ctx context.Context, callerUsername, roleCode string, grants []permission.PermissionGrant) error
	// EnsureCanManageUser 要求目标用户的全部授权权限都在调用者自己的授权范围内，
	// 用于改密码/状态/过期时间、删除、强制下线等"可接管账号"的操作。
	EnsureCanManageUser(ctx context.Context, callerID, targetID uint) error
	// EnsureCanSetRoleDataScope 防止通过改角色数据范围放大自己或他人可见的数据。
	EnsureCanSetRoleDataScope(ctx context.Context, callerUsername, roleCode string, scope permission.DataScope, departmentIDs []uint) error
	// SessionScope 是会话激活角色的数据范围，列表过滤与单条可见性都以它为准。
	SessionScope(ctx context.Context, sessionID string, userID uint) (permission.AccessScope, error)
	EnsureUserVisible(ctx context.Context, scope permission.AccessScope, username string) error
	RemoveUserRole(ctx context.Context, username, roleCode string) error
	GetUserAccess(ctx context.Context, username string) (*AccessSnapshot, error)
	CreateSession(ctx context.Context, userID uint, ttl time.Duration, origin SessionOrigin) (*AccessSnapshot, error)
	// TouchSession 记录会话仍在使用（令牌刷新时调用），供在线会话列表展示。
	TouchSession(ctx context.Context, sessionID string) error
	GetSessionAccess(ctx context.Context, sessionID string, userID uint) (*AccessSnapshot, error)
	SetActiveRoles(ctx context.Context, sessionID string, userID uint, roleCodes []string) (*AccessSnapshot, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeUserSessions(ctx context.Context, userID uint) error
	DeleteUserAuthorization(ctx context.Context, userID uint) error
	AuthorizeSession(ctx context.Context, sessionID string, userID uint, object, action string) (bool, error)
}

type rbacService struct {
	repo         permission.AuthorizationRepository
	roleRepo     permission.RoleRepository
	resourceRepo permission.ResourceRepository
	userRepo     user.Repository
	departments  department.Repository
}

func NewRBACService(repo permission.AuthorizationRepository, roleRepo permission.RoleRepository, resourceRepo permission.ResourceRepository, userRepo user.Repository, departments department.Repository) RBACService {
	return &rbacService{repo: repo, roleRepo: roleRepo, resourceRepo: resourceRepo, userRepo: userRepo, departments: departments}
}

func (s *rbacService) GetRolePermissions(ctx context.Context, roleCode string) ([]RolePermissionDetail, error) {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return nil, apperror.ErrRoleNotFound
	}
	rows, err := s.repo.GetRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	resources, err := s.resourceRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	resourceByID := make(map[uint]permission.Resource, len(resources))
	for _, resource := range resources {
		resourceByID[resource.ID] = resource
	}
	details := make(map[uint]*RolePermissionDetail)
	order := make([]uint, 0)
	for _, row := range rows {
		resource, ok := resourceByID[row.ResourceID]
		if !ok {
			continue
		}
		detail, exists := details[row.ResourceID]
		if !exists {
			detail = &RolePermissionDetail{ResourceID: resource.ID, ResourceCode: resource.Code, ResourceName: resource.Name, ResourcePath: resource.Path, IsSystem: row.IsSystem}
			details[row.ResourceID] = detail
			order = append(order, row.ResourceID)
		}
		detail.Actions = append(detail.Actions, row.Action)
		detail.IsSystem = detail.IsSystem && row.IsSystem
	}
	result := make([]RolePermissionDetail, 0, len(order))
	for _, id := range order {
		result = append(result, *details[id])
	}
	return result, nil
}

func (s *rbacService) SetRolePermissions(ctx context.Context, roleCode string, grants []permission.PermissionGrant) error {
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
		if err != nil {
			return apperror.ErrRoleNotFound
		}
		resources, err := s.resourceRepo.GetAll(ctx)
		if err != nil {
			return err
		}
		resourceByID := make(map[uint]permission.Resource, len(resources))
		for _, resource := range resources {
			resourceByID[resource.ID] = resource
		}
		seen := make(map[string]struct{})
		rows := make([]permission.RolePermission, 0)
		for _, grant := range grants {
			resource, ok := resourceByID[grant.ResourceID]
			if !ok {
				return apperror.ErrResourceNotFound
			}
			supported := make(map[string]struct{}, len(resource.Actions))
			for _, action := range resource.Actions {
				supported[action] = struct{}{}
			}
			for _, action := range grant.Actions {
				action = strings.ToUpper(strings.TrimSpace(action))
				if _, ok := supported[action]; !ok {
					return apperror.ErrRolePermissionInvalid
				}
				key := stringID(grant.ResourceID) + "\x00" + action
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				rows = append(rows, permission.RolePermission{RoleID: role.ID, ResourceID: grant.ResourceID, Action: action})
			}
		}
		return tx.ReplaceRolePermissions(ctx, role.ID, rows)
	})
}

func (s *rbacService) GetRoleJuniors(ctx context.Context, roleCode string) ([]permission.Role, error) {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return nil, apperror.ErrRoleNotFound
	}
	edges, err := s.repo.GetRoleHierarchies(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := roleMap(roles)
	result := make([]permission.Role, 0)
	for _, edge := range edges {
		if edge.SeniorRoleID == role.ID {
			if junior, ok := byID[edge.JuniorRoleID]; ok {
				result = append(result, junior)
			}
		}
	}
	sortRoles(result)
	return result, nil
}

func (s *rbacService) SetRoleJuniors(ctx context.Context, roleCode string, juniorRoleCodes []string) error {
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		senior, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
		if err != nil {
			return apperror.ErrRoleNotFound
		}
		roles, err := s.roleRepo.GetAll(ctx)
		if err != nil {
			return err
		}
		byCode := make(map[string]permission.Role, len(roles))
		for _, role := range roles {
			byCode[role.Code] = role
		}
		juniorIDs := make([]uint, 0, len(juniorRoleCodes))
		seen := make(map[uint]struct{})
		for _, code := range juniorRoleCodes {
			code = strings.ToLower(strings.TrimSpace(code))
			junior, ok := byCode[code]
			if !ok {
				return apperror.ErrRoleNotFound
			}
			if junior.ID == senior.ID {
				return apperror.ErrInvalidRoleHierarchy
			}
			if _, ok := seen[junior.ID]; !ok {
				seen[junior.ID] = struct{}{}
				juniorIDs = append(juniorIDs, junior.ID)
			}
		}
		edges, err := tx.GetRoleHierarchies(ctx)
		if err != nil {
			return err
		}
		candidate := make([]permission.RoleHierarchy, 0, len(edges)+len(juniorIDs))
		for _, edge := range edges {
			if edge.SeniorRoleID != senior.ID {
				candidate = append(candidate, edge)
			}
		}
		for _, id := range juniorIDs {
			candidate = append(candidate, permission.RoleHierarchy{SeniorRoleID: senior.ID, JuniorRoleID: id})
		}
		if hasHierarchyCycle(candidate) {
			return apperror.ErrRoleHierarchyCycle
		}
		if err := s.validateExistingRelations(ctx, tx, roles, candidate, nil); err != nil {
			return err
		}
		return tx.ReplaceRoleJuniors(ctx, senior.ID, juniorIDs)
	})
}

func (s *rbacService) GetConstraints(ctx context.Context) ([]permission.SeparationConstraint, error) {
	return s.repo.GetConstraints(ctx)
}

func validateConstraintShape(constraint *permission.SeparationConstraint) error {
	if constraint == nil {
		return apperror.ErrInvalidConstraint
	}
	constraint.Code = strings.ToLower(strings.TrimSpace(constraint.Code))
	constraint.Name = strings.TrimSpace(constraint.Name)
	constraint.Description = strings.TrimSpace(constraint.Description)
	constraint.Type = permission.ConstraintType(strings.ToUpper(strings.TrimSpace(string(constraint.Type))))
	if constraint.Type != permission.ConstraintTypeSSD && constraint.Type != permission.ConstraintTypeDSD {
		return apperror.ErrInvalidConstraint
	}
	set := make(map[uint]struct{}, len(constraint.RoleIDs))
	for _, id := range constraint.RoleIDs {
		if id == 0 {
			return apperror.ErrInvalidConstraint
		}
		set[id] = struct{}{}
	}
	constraint.RoleIDs = constraint.RoleIDs[:0]
	for id := range set {
		constraint.RoleIDs = append(constraint.RoleIDs, id)
	}
	sort.Slice(constraint.RoleIDs, func(i, j int) bool { return constraint.RoleIDs[i] < constraint.RoleIDs[j] })
	if len(constraint.RoleIDs) < 2 || constraint.Cardinality < 2 || constraint.Cardinality > len(constraint.RoleIDs) ||
		!roleCodePattern.MatchString(constraint.Code) || utf8.RuneCountInString(constraint.Code) < 2 || utf8.RuneCountInString(constraint.Code) > 64 ||
		utf8.RuneCountInString(constraint.Name) < 2 || utf8.RuneCountInString(constraint.Name) > 100 ||
		utf8.RuneCountInString(constraint.Description) > 500 {
		return apperror.ErrInvalidConstraint
	}
	return nil
}

func (s *rbacService) CreateConstraint(ctx context.Context, constraint *permission.SeparationConstraint) error {
	if err := validateConstraintShape(constraint); err != nil {
		return err
	}
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		roles, err := s.roleRepo.GetAll(ctx)
		if err != nil {
			return err
		}
		if !containsRoleIDs(roles, constraint.RoleIDs) {
			return apperror.ErrRoleNotFound
		}
		existing, err := tx.GetConstraints(ctx)
		if err != nil {
			return err
		}
		for _, item := range existing {
			if item.Code == constraint.Code {
				return apperror.ErrInvalidConstraint
			}
		}
		edges, err := tx.GetRoleHierarchies(ctx)
		if err != nil {
			return err
		}
		candidate := append(existing, *constraint)
		if err := s.validateExistingRelations(ctx, tx, roles, edges, candidate); err != nil {
			return err
		}
		return tx.CreateConstraint(ctx, constraint)
	})
}

func (s *rbacService) UpdateConstraint(ctx context.Context, id uint, constraint *permission.SeparationConstraint) error {
	if err := validateConstraintShape(constraint); err != nil {
		return err
	}
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		current, err := tx.GetConstraint(ctx, id)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return apperror.ErrRecordNotFound
			}
			return err
		}
		constraint.ID, constraint.CreatedAt, constraint.CreatedBy = id, current.CreatedAt, current.CreatedBy
		roles, err := s.roleRepo.GetAll(ctx)
		if err != nil {
			return err
		}
		if !containsRoleIDs(roles, constraint.RoleIDs) {
			return apperror.ErrRoleNotFound
		}
		constraints, err := tx.GetConstraints(ctx)
		if err != nil {
			return err
		}
		for i := range constraints {
			if constraints[i].ID != id && constraints[i].Code == constraint.Code {
				return apperror.ErrInvalidConstraint
			}
			if constraints[i].ID == id {
				constraints[i] = *constraint
			}
		}
		edges, err := tx.GetRoleHierarchies(ctx)
		if err != nil {
			return err
		}
		if err := s.validateExistingRelations(ctx, tx, roles, edges, constraints); err != nil {
			return err
		}
		return tx.UpdateConstraint(ctx, constraint)
	})
}

func (s *rbacService) DeleteConstraint(ctx context.Context, id uint) error {
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		return tx.DeleteConstraint(ctx, id)
	})
}

func (s *rbacService) GetUserRoleCodes(ctx context.Context, username string) ([]string, error) {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	assignments, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := roleMap(roles)
	codes := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		if role, ok := byID[assignment.RoleID]; ok {
			codes = append(codes, role.Code)
		}
	}
	sort.Strings(codes)
	return codes, nil
}

func (s *rbacService) GetRoleUsernames(ctx context.Context, scope permission.AccessScope, roleCode string) ([]string, error) {
	role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
	if err != nil {
		return nil, apperror.ErrRoleNotFound
	}
	ids, err := s.repo.GetRoleUserIDs(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		u, err := s.userRepo.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if scope.Contains(u.ID, u.DepartmentID) {
			result = append(result, u.Username)
		}
	}
	sort.Strings(result)
	return result, nil
}

func (s *rbacService) AddUserRole(ctx context.Context, username, roleCode string, system bool) error {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return err
	}
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		if err := tx.LockUsers(ctx, u.ID); err != nil {
			return err
		}
		u, err = s.userRepo.GetByUsername(ctx, username)
		if err != nil {
			return err
		}
		role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
		if err != nil || !role.IsEnabled {
			return apperror.ErrRoleNotFound
		}
		assignments, err := tx.GetUserRoles(ctx, u.ID)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.RoleID == role.ID {
				return nil
			}
		}
		roles, edges, constraints, err := s.loadModel(ctx, tx)
		if err != nil {
			return err
		}
		ids := make([]uint, 0, len(assignments)+1)
		for _, assignment := range assignments {
			ids = append(ids, assignment.RoleID)
		}
		ids = append(ids, role.ID)
		if violates(ids, roles, edges, constraints, permission.ConstraintTypeSSD) {
			return apperror.ErrSSDViolation
		}
		return tx.AddUserRole(ctx, &permission.UserRole{UserID: u.ID, RoleID: role.ID, IsSystem: system})
	})
}

func (s *rbacService) RemoveUserRole(ctx context.Context, username, roleCode string) error {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		if err := tx.LockUsers(ctx, u.ID); err != nil {
			return err
		}
		u, err = s.userRepo.GetByUsername(ctx, username)
		if err != nil {
			return err
		}
		role, err := s.roleRepo.GetByCode(ctx, canonicalRoleCode(roleCode))
		if err != nil {
			return apperror.ErrRoleNotFound
		}
		assignments, err := tx.GetUserRoles(ctx, u.ID)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.RoleID == role.ID && assignment.IsSystem {
				return apperror.ErrSystemRoleDelete
			}
		}
		if err := tx.DeleteUserRole(ctx, u.ID, role.ID); err != nil {
			return err
		}
		return tx.RevokeUserSessions(ctx, u.ID)
	}); err != nil {
		return err
	}
	return nil
}

func (s *rbacService) GetUserAccess(ctx context.Context, username string) (*AccessSnapshot, error) {
	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return s.accessForRoleIDs(ctx, "", u, nil, false)
}

func (s *rbacService) CreateSession(ctx context.Context, userID uint, ttl time.Duration, origin SessionOrigin) (*AccessSnapshot, error) {
	if ttl <= 0 {
		return nil, apperror.ErrAuthorizationSessionExpired
	}
	var u *user.User
	var active []uint
	now := time.Now()
	session := &permission.AuthorizationSession{
		ID: uuid.NewString(), UserID: userID,
		IpAddr:    truncateRunes(origin.IP, maxSessionIPLength),
		UserAgent: truncateRunes(origin.UserAgent, maxSessionUserAgentLength),
		Remember:  origin.Remember,
		CreatedAt: now, LastActiveAt: now, ExpiresAt: now.Add(ttl), Version: 1,
	}
	if err := s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		if err := tx.LockUsers(ctx, userID); err != nil {
			return err
		}
		var err error
		u, err = s.userRepo.Get(ctx, userID)
		if err != nil {
			return err
		}
		if err := validateUserState(u); err != nil {
			return err
		}
		session.Username = u.Username
		assignments, err := tx.GetUserRoles(ctx, userID)
		if err != nil {
			return err
		}
		roles, edges, constraints, err := s.loadModel(ctx, tx)
		if err != nil {
			return err
		}
		direct := assignmentRoleIDs(assignments)
		active = make([]uint, 0, len(direct))
		for _, id := range direct {
			candidate := append(append([]uint(nil), active...), id)
			if !violates(candidate, roles, edges, constraints, permission.ConstraintTypeDSD) {
				active = candidate
			}
		}
		return tx.CreateSession(ctx, session, active)
	}); err != nil {
		return nil, err
	}
	return s.accessForRoleIDs(ctx, session.ID, u, active, true)
}

func (s *rbacService) GetSessionAccess(ctx context.Context, sessionID string, userID uint) (*AccessSnapshot, error) {
	session, err := s.validSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	u, err := s.userRepo.Get(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.GetSessionRoleIDs(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	return s.accessForRoleIDs(ctx, session.ID, u, active, true)
}

func (s *rbacService) SetActiveRoles(ctx context.Context, sessionID string, userID uint, roleCodes []string) (*AccessSnapshot, error) {
	var u *user.User
	var active []uint
	if err := s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		if err := tx.LockUsers(ctx, userID); err != nil {
			return err
		}
		if err := tx.LockSession(ctx, sessionID); err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return apperror.ErrAuthorizationSessionNotFound
			}
			return err
		}
		session, err := s.validSessionWithRepository(ctx, tx, sessionID, userID)
		if err != nil {
			return err
		}
		u, err = s.userRepo.Get(ctx, session.UserID)
		if err != nil {
			return err
		}
		if err := validateUserState(u); err != nil {
			return err
		}
		roles, edges, constraints, err := s.loadModel(ctx, tx)
		if err != nil {
			return err
		}
		assignments, err := tx.GetUserRoles(ctx, u.ID)
		if err != nil {
			return err
		}
		authorized := closureSet(assignmentRoleIDs(assignments), edges, enabledRoleSet(roles))
		byCode := make(map[string]permission.Role, len(roles))
		active = make([]uint, 0, len(roleCodes))
		seen := make(map[uint]struct{})
		for _, role := range roles {
			byCode[role.Code] = role
		}
		for _, code := range roleCodes {
			code = strings.ToLower(strings.TrimSpace(code))
			role, ok := byCode[code]
			if !ok || !role.IsEnabled {
				return apperror.ErrRoleNotFound
			}
			if _, ok := authorized[role.ID]; !ok {
				return apperror.ErrRoleNotAuthorized
			}
			if _, ok := seen[role.ID]; !ok {
				seen[role.ID] = struct{}{}
				active = append(active, role.ID)
			}
		}
		if violates(active, roles, edges, constraints, permission.ConstraintTypeDSD) {
			return apperror.ErrDSDViolation
		}
		return tx.ReplaceSessionRoles(ctx, session.ID, active)
	}); err != nil {
		return nil, err
	}
	return s.accessForRoleIDs(ctx, sessionID, u, active, true)
}

func (s *rbacService) RevokeSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return apperror.ErrAuthorizationSessionNotFound
	}
	return s.repo.RevokeSession(ctx, sessionID)
}

func (s *rbacService) TouchSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return apperror.ErrAuthorizationSessionNotFound
	}
	return s.repo.TouchSession(ctx, sessionID, time.Now())
}

// truncateRunes 按字符截断，避免把多字节字符截成非法 UTF-8。
func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func (s *rbacService) RevokeUserSessions(ctx context.Context, userID uint) error {
	return s.repo.RevokeUserSessions(ctx, userID)
}

func (s *rbacService) DeleteUserAuthorization(ctx context.Context, userID uint) error {
	return s.repo.WithTx(ctx, func(tx permission.AuthorizationRepository) error {
		if err := tx.LockRoles(ctx); err != nil {
			return err
		}
		if err := tx.LockUsers(ctx, userID); err != nil {
			return err
		}
		if err := tx.RevokeUserSessions(ctx, userID); err != nil {
			return err
		}
		return tx.DeleteUserRoles(ctx, userID)
	})
}

func (s *rbacService) AuthorizeSession(ctx context.Context, sessionID string, userID uint, object, action string) (bool, error) {
	access, err := s.GetSessionAccess(ctx, sessionID, userID)
	if err != nil {
		return false, err
	}
	// The middleware supplies Gin's route template. Match it exactly so a grant
	// for /objects/:id cannot authorize a separate /objects/export endpoint.
	for _, grant := range access.Permissions {
		if grant.ResourcePath == object && grant.Action == action {
			return true, nil
		}
	}
	return false, nil
}

func (s *rbacService) validSession(ctx context.Context, sessionID string, userID uint) (*permission.AuthorizationSession, error) {
	return s.validSessionWithRepository(ctx, s.repo, sessionID, userID)
}

func (s *rbacService) validSessionWithRepository(ctx context.Context, repo permission.AuthorizationRepository, sessionID string, userID uint) (*permission.AuthorizationSession, error) {
	if sessionID == "" {
		return nil, apperror.ErrAuthorizationSessionNotFound
	}
	session, err := repo.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, apperror.ErrAuthorizationSessionNotFound
		}
		return nil, err
	}
	if session.UserID != userID {
		return nil, apperror.ErrAuthorizationSessionNotFound
	}
	if session.RevokedAt != nil {
		return nil, apperror.ErrAuthorizationSessionRevoked
	}
	if !session.ExpiresAt.After(time.Now()) {
		return nil, apperror.ErrAuthorizationSessionExpired
	}
	return session, nil
}

func (s *rbacService) accessForRoleIDs(ctx context.Context, sessionID string, u *user.User, activeIDs []uint, sessionMode bool) (*AccessSnapshot, error) {
	if err := validateUserState(u); err != nil {
		return nil, err
	}
	roles, edges, constraints, err := s.loadModel(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	assignments, err := s.repo.GetUserRoles(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	directIDs := assignmentRoleIDs(assignments)
	enabled := enabledRoleSet(roles)
	authorizedSet := closureSet(directIDs, edges, enabled)
	if !sessionMode {
		permissions, err := s.effectivePermissions(ctx, authorizedSet, roles)
		if err != nil {
			return nil, err
		}
		return &AccessSnapshot{
			SessionID:       sessionID,
			Username:        u.Username,
			AssignedRoles:   rolesForIDs(directIDs, roles),
			AuthorizedRoles: rolesForSet(authorizedSet, roles),
			ActiveRoles:     []permission.Role{},
			Permissions:     permissions,
		}, nil
	}
	for _, id := range activeIDs {
		if _, ok := authorizedSet[id]; !ok {
			return nil, apperror.ErrRoleNotAuthorized
		}
	}
	if violates(activeIDs, roles, edges, constraints, permission.ConstraintTypeDSD) {
		return nil, apperror.ErrDSDViolation
	}
	activeClosure := closureSet(activeIDs, edges, enabled)
	permissions, err := s.effectivePermissions(ctx, activeClosure, roles)
	if err != nil {
		return nil, err
	}
	return &AccessSnapshot{
		SessionID:       sessionID,
		Username:        u.Username,
		AssignedRoles:   rolesForIDs(directIDs, roles),
		AuthorizedRoles: rolesForSet(authorizedSet, roles),
		ActiveRoles:     rolesForIDs(activeIDs, roles),
		Permissions:     permissions,
	}, nil
}

func (s *rbacService) effectivePermissions(ctx context.Context, roleIDs map[uint]struct{}, roles []permission.Role) ([]permission.EffectivePermission, error) {
	rows, err := s.repo.GetAllRolePermissions(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := s.resourceRepo.GetAllEnabled(ctx)
	if err != nil {
		return nil, err
	}
	resourceByID := make(map[uint]permission.Resource, len(resources))
	for _, resource := range resources {
		resourceByID[resource.ID] = resource
	}
	roleByID := roleMap(roles)
	seen := make(map[string]struct{})
	result := make([]permission.EffectivePermission, 0)
	for _, row := range rows {
		if _, ok := roleIDs[row.RoleID]; !ok {
			continue
		}
		role, roleOK := roleByID[row.RoleID]
		resource, resourceOK := resourceByID[row.ResourceID]
		if !roleOK || !role.IsEnabled || !resourceOK || !resource.Actions.Contains(row.Action) {
			continue
		}
		key := resource.Path + "\x00" + row.Action
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, permission.EffectivePermission{RoleID: role.ID, RoleCode: role.Code, ResourceID: resource.ID, ResourceCode: resource.Code, ResourcePath: resource.Path, Action: row.Action})
	}
	return result, nil
}

func (s *rbacService) loadModel(ctx context.Context, repo permission.AuthorizationRepository) ([]permission.Role, []permission.RoleHierarchy, []permission.SeparationConstraint, error) {
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	edges, err := repo.GetRoleHierarchies(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	constraints, err := repo.GetConstraints(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return roles, edges, constraints, nil
}

func (s *rbacService) validateExistingRelations(ctx context.Context, repo permission.AuthorizationRepository, roles []permission.Role, edges []permission.RoleHierarchy, override []permission.SeparationConstraint) error {
	constraints := override
	if constraints == nil {
		var err error
		constraints, err = repo.GetConstraints(ctx)
		if err != nil {
			return err
		}
	}
	assignments, err := repo.GetAllUserRoles(ctx)
	if err != nil {
		return err
	}
	byUser := make(map[uint][]uint)
	for _, assignment := range assignments {
		byUser[assignment.UserID] = append(byUser[assignment.UserID], assignment.RoleID)
	}
	for _, ids := range byUser {
		if violates(ids, roles, edges, constraints, permission.ConstraintTypeSSD) {
			return apperror.ErrSSDViolation
		}
	}
	sessions, err := repo.GetActiveSessions(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		ids, err := repo.GetSessionRoleIDs(ctx, session.ID)
		if err != nil {
			return err
		}
		if violates(ids, roles, edges, constraints, permission.ConstraintTypeDSD) {
			return apperror.ErrDSDViolation
		}
	}
	return nil
}

func validateUserState(u *user.User) error {
	if !u.Enable {
		return apperror.ErrUserDisabled
	}
	if u.Locked {
		return apperror.ErrUserLocked
	}
	if !u.AccountExpireDate.IsZero() && !u.AccountExpireDate.After(time.Now()) {
		return apperror.ErrAccountExpired
	}
	if !u.CredentialExpireDate.IsZero() && !u.CredentialExpireDate.After(time.Now()) {
		return apperror.ErrCredentialExpired
	}
	return nil
}

func violates(seed []uint, roles []permission.Role, edges []permission.RoleHierarchy, constraints []permission.SeparationConstraint, kind permission.ConstraintType) bool {
	closure := closureSet(seed, edges, enabledRoleSet(roles))
	for _, constraint := range constraints {
		if !constraint.IsEnabled || constraint.Type != kind {
			continue
		}
		count := 0
		for _, roleID := range constraint.RoleIDs {
			if _, ok := closure[roleID]; ok {
				count++
			}
		}
		if count >= constraint.Cardinality {
			return true
		}
	}
	return false
}

func closureSet(seed []uint, edges []permission.RoleHierarchy, allowed map[uint]struct{}) map[uint]struct{} {
	children := make(map[uint][]uint)
	for _, edge := range edges {
		children[edge.SeniorRoleID] = append(children[edge.SeniorRoleID], edge.JuniorRoleID)
	}
	result := make(map[uint]struct{})
	stack := append([]uint(nil), seed...)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, ok := result[id]; ok {
			continue
		}
		result[id] = struct{}{}
		stack = append(stack, children[id]...)
	}
	return result
}

func hasHierarchyCycle(edges []permission.RoleHierarchy) bool {
	children := make(map[uint][]uint)
	for _, edge := range edges {
		children[edge.SeniorRoleID] = append(children[edge.SeniorRoleID], edge.JuniorRoleID)
	}
	state := make(map[uint]uint8)
	var visit func(uint) bool
	visit = func(id uint) bool {
		if state[id] == 1 {
			return true
		}
		if state[id] == 2 {
			return false
		}
		state[id] = 1
		for _, child := range children[id] {
			if visit(child) {
				return true
			}
		}
		state[id] = 2
		return false
	}
	for id := range children {
		if visit(id) {
			return true
		}
	}
	return false
}

func roleMap(roles []permission.Role) map[uint]permission.Role {
	result := make(map[uint]permission.Role, len(roles))
	for _, role := range roles {
		result[role.ID] = role
	}
	return result
}

func enabledRoleSet(roles []permission.Role) map[uint]struct{} {
	result := make(map[uint]struct{}, len(roles))
	for _, role := range roles {
		if role.IsEnabled {
			result[role.ID] = struct{}{}
		}
	}
	return result
}

func containsRoleIDs(roles []permission.Role, ids []uint) bool {
	byID := roleMap(roles)
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			return false
		}
	}
	return true
}

func assignmentRoleIDs(assignments []permission.UserRole) []uint {
	result := make([]uint, len(assignments))
	for i := range assignments {
		result[i] = assignments[i].RoleID
	}
	return result
}

func rolesForIDs(ids []uint, roles []permission.Role) []permission.Role {
	byID := roleMap(roles)
	result := make([]permission.Role, 0, len(ids))
	seen := make(map[uint]struct{})
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		if role, ok := byID[id]; ok {
			seen[id] = struct{}{}
			result = append(result, role)
		}
	}
	sortRoles(result)
	return result
}

func rolesForSet(ids map[uint]struct{}, roles []permission.Role) []permission.Role {
	return rolesForIDs(setIDs(ids), roles)
}

func setIDs(set map[uint]struct{}) []uint {
	result := make([]uint, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	return result
}

func sortRoles(roles []permission.Role) {
	sort.Slice(roles, func(i, j int) bool { return roles[i].Code < roles[j].Code })
}

func stringID(id uint) string {
	if id == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for id > 0 {
		buf = append(buf, byte('0'+id%10))
		id /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
