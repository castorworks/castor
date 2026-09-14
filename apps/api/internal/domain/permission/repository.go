package permission

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/query"
)

// ResourceRepository 资源仓储接口
type ResourceRepository interface {
	// GetAll 获取所有资源（包括禁用资源）
	GetAll(ctx context.Context) ([]Resource, error)
	// GetAllEnabled 获取所有启用的资源
	GetAllEnabled(ctx context.Context) ([]Resource, error)
	// Gets 分页查询资源列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]Resource, int64, error)
	// Get 根据 ID 获取资源
	Get(ctx context.Context, id uint) (*Resource, error)
	// GetByCode 根据编码获取资源
	GetByCode(ctx context.Context, code string) (*Resource, error)
	// GetByPath 根据路径获取资源
	GetByPath(ctx context.Context, path string) (*Resource, error)
	// GetByCategory 根据分类获取资源
	GetByCategory(ctx context.Context, category ResourceCategory) ([]Resource, error)
	// GetByModule 根据模块获取资源
	GetByModule(ctx context.Context, module string) ([]Resource, error)
	// GetByIDs 根据 ID 列表获取资源
	GetByIDs(ctx context.Context, ids []uint) ([]Resource, error)
	// Create 创建资源
	Create(ctx context.Context, resource *Resource) error
	// BatchCreate 批量创建资源
	BatchCreate(ctx context.Context, resources []Resource) error
	// Update 更新资源
	Update(ctx context.Context, resource *Resource) error
	// Delete 删除资源
	Delete(ctx context.Context, id uint) error
	// ExistsByCode 检查编码是否存在
	ExistsByCode(ctx context.Context, code string) (bool, error)
}

// RoleRepository 角色仓储接口
type RoleRepository interface {
	// GetAll 获取所有角色
	GetAll(ctx context.Context) ([]Role, error)
	// GetAllEnabled 获取所有启用的角色
	GetAllEnabled(ctx context.Context) ([]Role, error)
	// Get 根据 ID 获取角色
	Get(ctx context.Context, id uint) (*Role, error)
	// GetByCode 根据编码获取角色
	GetByCode(ctx context.Context, code string) (*Role, error)
	// Create 创建角色
	Create(ctx context.Context, role *Role) error
	// BatchCreate 批量创建角色
	BatchCreate(ctx context.Context, roles []Role) error
	// Update 更新角色
	Update(ctx context.Context, role *Role) error
	// Delete 删除角色
	Delete(ctx context.Context, id uint) error
	// ExistsByCode 检查编码是否存在
	ExistsByCode(ctx context.Context, code string) (bool, error)
}

// AuthorizationRepository owns every RBAC relation. WithTx returns a repository
// bound to one database transaction so validation and mutation are atomic.
type AuthorizationRepository interface {
	WithTx(ctx context.Context, fn func(AuthorizationRepository) error) error
	LockUsers(ctx context.Context, userIDs ...uint) error
	LockRoles(ctx context.Context) error
	LockSession(ctx context.Context, sessionID string) error

	CreateResource(ctx context.Context, resource *Resource) error
	UpdateResource(ctx context.Context, resource *Resource) error
	DeleteResource(ctx context.Context, id uint) error
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uint) error

	GetAllRolePermissions(ctx context.Context) ([]RolePermission, error)
	GetRolePermissions(ctx context.Context, roleID uint) ([]RolePermission, error)
	ReplaceRolePermissions(ctx context.Context, roleID uint, permissions []RolePermission) error
	ReconcileSystemRolePermissions(ctx context.Context, roleID uint, permissions []RolePermission) error
	PruneResourcePermissions(ctx context.Context, resourceID uint, allowedActions []string) error

	GetAllUserRoles(ctx context.Context) ([]UserRole, error)
	GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error)
	GetRoleUserIDs(ctx context.Context, roleID uint) ([]uint, error)
	AddUserRole(ctx context.Context, assignment *UserRole) error
	DeleteUserRole(ctx context.Context, userID, roleID uint) error
	DeleteUserRoles(ctx context.Context, userID uint) error

	GetRoleHierarchies(ctx context.Context) ([]RoleHierarchy, error)
	ReplaceRoleJuniors(ctx context.Context, seniorRoleID uint, juniorRoleIDs []uint) error

	GetConstraints(ctx context.Context) ([]SeparationConstraint, error)
	GetConstraint(ctx context.Context, id uint) (*SeparationConstraint, error)
	CreateConstraint(ctx context.Context, constraint *SeparationConstraint) error
	UpdateConstraint(ctx context.Context, constraint *SeparationConstraint) error
	DeleteConstraint(ctx context.Context, id uint) error

	CreateSession(ctx context.Context, session *AuthorizationSession, activeRoleIDs []uint) error
	GetSession(ctx context.Context, id string) (*AuthorizationSession, error)
	GetSessionRoleIDs(ctx context.Context, sessionID string) ([]uint, error)
	ReplaceSessionRoles(ctx context.Context, sessionID string, roleIDs []uint) error
	GetActiveSessions(ctx context.Context) ([]AuthorizationSession, error)
	RevokeSession(ctx context.Context, id string) error
	RevokeUserSessions(ctx context.Context, userID uint) error
	RevokeSessionsWithRole(ctx context.Context, roleID uint) error
}
