package asset

import (
	"context"
	"time"

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

	// Delete 删除资产记录；仍被引用时由外键拒绝
	Delete(ctx context.Context, id uint) error

	// BatchDelete 批量删除资产记录（全有或全无）；任意一条仍被引用时由外键拒绝
	BatchDelete(ctx context.Context, ids []uint) error

	// ========================
	// 查询
	// ========================

	// Get 根据 ID 获取资产
	Get(ctx context.Context, id uint) (*Asset, error)

	// GetByObjectKey 根据对象键获取资产
	GetByObjectKey(ctx context.Context, objectKey string) (*Asset, error)

	// GetByHash 根据文件哈希获取资产（用于去重）
	GetByHash(ctx context.Context, hash string) (*Asset, error)

	// Gets 分页查询资产列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]Asset, int64, error)

	// GetPublicAssets 获取公开资产列表（仅资产库中生效的公开资产）
	GetPublicAssets(ctx context.Context, page, size int, order string) ([]Asset, int64, error)

	// GetCombinedStats 合并统计查询（分类、状态、总大小一次查询）
	GetCombinedStats(ctx context.Context) (categoryStats map[AssetCategory]int64, statusStats map[AssetStatus]int64, totalSize int64, err error)

	// ========================
	// 更新操作
	// ========================

	// UpdateStatus 更新资产状态
	UpdateStatus(ctx context.Context, id uint, status AssetStatus) error

	// BatchUpdateStatus 批量更新资产状态
	BatchUpdateStatus(ctx context.Context, ids []uint, status AssetStatus) error

	// UpdateScope 更新资产归属范围
	UpdateScope(ctx context.Context, id uint, scope AssetScope) error

	// UpdateIsPublic 更新资产公开状态
	UpdateIsPublic(ctx context.Context, id uint, isPublic bool) error

	// UpdateFolderPath 移动资产到指定虚拟文件夹
	UpdateFolderPath(ctx context.Context, id uint, folderPath string) error

	// IncrementDownloadCount 增加下载次数
	IncrementDownloadCount(ctx context.Context, id uint) error

	// ========================
	// 引用
	// ========================

	// AddReference 登记 ref 对资产的引用；重复登记是幂等的，只更新显示名
	AddReference(ctx context.Context, assetID uint, ref Reference, name string) error

	// RemoveReference 解除 ref 对资产的引用；引用不存在时不报错
	RemoveReference(ctx context.Context, assetID uint, ref Reference) error

	// GetReferences 列出资产的全部引用
	GetReferences(ctx context.Context, assetID uint) ([]Reference, error)

	// CountReferences 统计资产被引用的次数
	CountReferences(ctx context.Context, assetID uint) (int64, error)

	// GetReferencedIDs 返回 ids 中仍被引用的资产 ID
	GetReferencedIDs(ctx context.Context, ids []uint) ([]uint, error)

	// GetIDsByReference 返回挂在 ref 这个业务字段上的全部资产 ID
	GetIDsByReference(ctx context.Context, ref Reference) ([]uint, error)

	// GetReferenceName 返回 ref 引用该资产时登记的显示名；未引用返回 shared.ErrNotFound
	GetReferenceName(ctx context.Context, assetID uint, ref Reference) (string, error)

	// GetAttached 批量读取若干业务记录在 field 上引用的资产，按登记顺序排列
	GetAttached(ctx context.Context, field Field, ownerIDs []uint) ([]Attached, error)

	// GetIDsByOwner 返回某个业务对象引用的全部资产 ID（去重）
	GetIDsByOwner(ctx context.Context, ownerType string, ownerID uint) ([]uint, error)

	// GetOrphanAttachments 返回在 before 之前最后变动、且没有任何引用的业务附件（最多 limit 条）
	GetOrphanAttachments(ctx context.Context, before time.Time, limit int) ([]Asset, error)

	// Touch 刷新资产的更新时间，让刚被去重复用的孤儿附件重新获得一个完整的宽限期
	Touch(ctx context.Context, id uint) error

	// RemoveOwnerReferences 解除某个业务对象的全部引用
	RemoveOwnerReferences(ctx context.Context, ownerType string, ownerID uint) error
}
