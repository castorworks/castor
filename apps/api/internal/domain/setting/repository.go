package setting

import (
	"context"
)

// Repository 系统配置仓储接口
type Repository interface {
	// Gets 获取配置列表，可按分类筛选
	Gets(ctx context.Context, category string) ([]Setting, error)
	// Get 根据 ID 获取配置
	Get(ctx context.Context, id uint) (*Setting, error)
	// GetByKey 根据 Key 获取配置
	GetByKey(ctx context.Context, key string) (*Setting, error)
	// Create 创建配置
	Create(ctx context.Context, setting *Setting) error
	// Update 更新配置
	Update(ctx context.Context, setting *Setting) error
	// Delete 删除配置
	Delete(ctx context.Context, id uint) error
	// GetByKeys 批量获取配置
	GetByKeys(ctx context.Context, keys []string) ([]Setting, error)
	// BatchUpdate 批量更新配置值
	BatchUpdate(ctx context.Context, settings []Setting) error
	// GetPublicSettings 获取公开配置
	GetPublicSettings(ctx context.Context) ([]Setting, error)
	// GetByCategory 按分类获取配置
	GetByCategory(ctx context.Context, category SettingCategory) ([]Setting, error)
	// ExistsByKey 检查 Key 是否存在
	ExistsByKey(ctx context.Context, key string) (bool, error)
}
