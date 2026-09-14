package login_history

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/query"
)

// Repository 登录历史仓储接口
type Repository interface {
	// Gets 分页查询登录历史列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]LoginHistory, int64, error)
	// Get 根据 ID 获取登录历史
	Get(ctx context.Context, id uint) (*LoginHistory, error)
	// Create 创建登录历史记录
	Create(ctx context.Context, item *LoginHistory) error
	// Delete 删除登录历史记录
	Delete(ctx context.Context, id uint) error
	// BatchDelete 批量删除登录历史记录
	BatchDelete(ctx context.Context, ids []uint) error
}
