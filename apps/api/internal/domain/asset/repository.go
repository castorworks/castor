package asset

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/query"
)

// Repository 资产仓储接口
type Repository interface {
	// ========================
	// 基础 CRUD 操作
	// ========================

	// Create 创建资产记录
	Create(ctx context.Context, asset *Asset) error

	// Update 更新资产记录
	Update(ctx context.Context, asset *Asset) error

	// Delete 删除资产记录
	Delete(ctx context.Context, id uint) error

	// ========================
	// 单条查询
	// ========================

	// Get 根据 ID 获取资产
	Get(ctx context.Context, id uint) (*Asset, error)

	// GetByObjectKey 根据对象键获取资产
	GetByObjectKey(ctx context.Context, objectKey string) (*Asset, error)

	// GetByHash 根据文件哈希获取资产（用于去重）
	GetByHash(ctx context.Context, hash string) (*Asset, error)

	// ========================
	// 列表查询
	// ========================

	// Gets 分页查询资产列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]Asset, int64, error)

	// GetsByCategory 按分类查询资产列表
	GetsByCategory(ctx context.Context, category AssetCategory, page, size int, order string) ([]Asset, int64, error)

	// GetsByStatus 按状态查询资产列表
	GetsByStatus(ctx context.Context, status AssetStatus, page, size int, order string) ([]Asset, int64, error)

	// GetsByFolderID 按文件夹 ID 查询资产列表
	GetsByFolderID(ctx context.Context, folderID uint, page, size int, order string) ([]Asset, int64, error)

	// GetsByFolderPath 按文件夹路径查询资产列表
	GetsByFolderPath(ctx context.Context, folderPath string, page, size int, order string) ([]Asset, int64, error)

	// GetsByCreatedBy 按创建者查询资产列表
	GetsByCreatedBy(ctx context.Context, createdBy uint, page, size int, order string) ([]Asset, int64, error)

	// GetsByTags 按标签模糊查询资产列表
	GetsByTags(ctx context.Context, tag string, page, size int, order string) ([]Asset, int64, error)

	// GetPublicAssets 获取公开资产列表
	GetPublicAssets(ctx context.Context, page, size int, order string) ([]Asset, int64, error)

	// ========================
	// 按对象键批量操作
	// ========================

	// DeleteByObjectKey 根据对象键删除资产记录
	DeleteByObjectKey(ctx context.Context, objectKey string) error

	// BatchDeleteByObjectKeys 批量删除资产记录
	BatchDeleteByObjectKeys(ctx context.Context, objectKeys []string) error

	// ========================
	// 统计查询
	// ========================

	// CountByCategory 统计各分类资产数量
	CountByCategory(ctx context.Context) (map[AssetCategory]int64, error)

	// CountByStatus 统计各状态资产数量
	CountByStatus(ctx context.Context) (map[AssetStatus]int64, error)

	// SumSize 统计资产总大小
	SumSize(ctx context.Context) (int64, error)

	// GetCombinedStats 合并统计查询（分类、状态、总大小一次查询）
	GetCombinedStats(ctx context.Context) (categoryStats map[AssetCategory]int64, statusStats map[AssetStatus]int64, totalSize int64, err error)

	// SumSizeByCreatedBy 统计指定用户资产总大小
	SumSizeByCreatedBy(ctx context.Context, createdBy uint) (int64, error)

	// ========================
	// 更新操作
	// ========================

	// UpdateStatus 更新资产状态
	UpdateStatus(ctx context.Context, id uint, status AssetStatus) error

	// UpdateIsPublic 更新资产公开状态
	UpdateIsPublic(ctx context.Context, id uint, isPublic bool) error

	// IncrementDownloadCount 增加下载次数
	IncrementDownloadCount(ctx context.Context, id uint) error

	// IncrementViewCount 增加查看次数
	IncrementViewCount(ctx context.Context, id uint) error

	// BatchUpdateStatus 批量更新资产状态
	BatchUpdateStatus(ctx context.Context, ids []uint, status AssetStatus) error

	// MoveToFolder 移动资产到指定文件夹
	MoveToFolder(ctx context.Context, id uint, folderID *uint, folderPath string) error
}
