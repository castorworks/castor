package dictionary

import (
	"context"
)

// DictTypeRepository 字典类型仓储接口
type DictTypeRepository interface {
	// Gets 获取所有字典类型
	Gets(ctx context.Context) ([]DictType, error)
	// Get 根据 ID 获取字典类型
	Get(ctx context.Context, id uint) (*DictType, error)
	// GetByCode 根据编码获取字典类型
	GetByCode(ctx context.Context, code string) (*DictType, error)
	// Create 创建字典类型
	Create(ctx context.Context, dictType *DictType) error
	// Update 更新字典类型
	Update(ctx context.Context, dictType *DictType) error
	// Delete 删除字典类型
	Delete(ctx context.Context, id uint) error
	// ExistsByCode 检查编码是否存在
	ExistsByCode(ctx context.Context, code string) (bool, error)
}

// DictItemRepository 字典项仓储接口
type DictItemRepository interface {
	// Gets 获取字典项列表
	Gets(ctx context.Context, typeCode string) ([]DictItem, error)
	// Get 根据 ID 获取字典项
	Get(ctx context.Context, id uint) (*DictItem, error)
	// GetByTypeCode 根据类型编码获取字典项
	GetByTypeCode(ctx context.Context, typeCode string) ([]DictItem, error)
	// GetEnabledByTypeCode 获取启用的字典项
	GetEnabledByTypeCode(ctx context.Context, typeCode string) ([]DictItem, error)
	// GetAllEnabled 获取所有启用的字典项
	GetAllEnabled(ctx context.Context) ([]DictItem, error)
	// Create 创建字典项
	Create(ctx context.Context, dictItem *DictItem) error
	// Update 更新字典项
	Update(ctx context.Context, dictItem *DictItem) error
	// Delete 删除字典项
	Delete(ctx context.Context, id uint) error
	// DeleteByTypeCode 根据类型编码删除字典项
	DeleteByTypeCode(ctx context.Context, typeCode string) error
	// BatchCreate 批量创建字典项
	BatchCreate(ctx context.Context, items []DictItem) error
	// ExistsByTypeCodeAndValue 检查同类型下是否存在相同值
	ExistsByTypeCodeAndValue(ctx context.Context, typeCode, value string, excludeID uint) (bool, error)
}
