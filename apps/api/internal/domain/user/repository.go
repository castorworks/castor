package user

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/query"
)

// Repository 用户仓储接口
// 定义领域层对数据持久化的抽象，具体实现在 infrastructure 层
type Repository interface {
	// Gets 分页查询用户列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]User, int64, error)
	// Get 根据 ID 获取用户
	Get(ctx context.Context, id uint) (*User, error)
	// GetByUsername 根据用户名获取用户
	GetByUsername(ctx context.Context, username string) (*User, error)
	// Create 创建用户
	Create(ctx context.Context, user *User) error
	// Update 更新用户
	Update(ctx context.Context, user *User) error
	// Delete 删除用户
	Delete(ctx context.Context, id uint) error
}
