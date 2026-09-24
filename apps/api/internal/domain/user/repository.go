package user

import (
	"context"

	"github.com/castorworks/castor/internal/domain/shared"
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
	// GetByEmail 根据邮箱获取用户；空邮箱视为“未绑定”，永远返回 shared.ErrNotFound
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByMobile 根据手机号获取用户；空手机号视为“未绑定”，永远返回 shared.ErrNotFound
	GetByMobile(ctx context.Context, mobile string) (*User, error)
	// Create 创建用户
	Create(ctx context.Context, user *User) error
	// Update 更新用户
	Update(ctx context.Context, user *User) error
	// Delete 删除用户
	Delete(ctx context.Context, id uint) error
	// CountByDepartment 各部门的成员数（没有成员的部门不出现在结果中）
	CountByDepartment(ctx context.Context) (map[uint]int64, error)
	// CountCreatedByMonth 最近 months 个自然月（含本月）每月新增的用户数，按月份升序
	CountCreatedByMonth(ctx context.Context, months int, opts ...query.Option) ([]shared.MonthlyCount, error)
}
