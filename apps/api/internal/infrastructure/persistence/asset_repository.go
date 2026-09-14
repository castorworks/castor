package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
)

type assetRepository struct {
	db *gorm.DB
}

// NewAssetRepository 创建资产仓储实现
func NewAssetRepository(db *gorm.DB) asset.Repository {
	return &assetRepository{db: db}
}

// ========================
// 基础 CRUD 操作
// ========================

func (r *assetRepository) Create(ctx context.Context, item *asset.Asset) error {
	m := models.AssetModelFromEntity(item)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	item.CreatedBy = m.CreatedBy
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *assetRepository) Update(ctx context.Context, item *asset.Asset) error {
	m := models.AssetModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *assetRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AssetModel{}, id).Error
}

// ========================
// 单条查询
// ========================

func (r *assetRepository) Get(ctx context.Context, id uint) (*asset.Asset, error) {
	var m models.AssetModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *assetRepository) GetByObjectKey(ctx context.Context, objectKey string) (*asset.Asset, error) {
	var m models.AssetModel
	if err := translateError(r.db.WithContext(ctx).Where("object_key = ?", objectKey).First(&m).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *assetRepository) GetByHash(ctx context.Context, hash string) (*asset.Asset, error) {
	var m models.AssetModel
	if err := translateError(r.db.WithContext(ctx).Where("hash = ?", hash).First(&m).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

// ========================
// 列表查询
// ========================

func (r *assetRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, opts...)
}

func (r *assetRepository) GetsByCategory(ctx context.Context, category asset.AssetCategory, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("category = ?", category))
}

func (r *assetRepository) GetsByStatus(ctx context.Context, status asset.AssetStatus, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("status = ?", status))
}

func (r *assetRepository) GetsByFolderID(ctx context.Context, folderID uint, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("folder_id = ?", folderID))
}

func (r *assetRepository) GetsByFolderPath(ctx context.Context, folderPath string, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("folder_path = ?", folderPath))
}

func (r *assetRepository) GetsByCreatedBy(ctx context.Context, createdBy uint, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("created_by = ?", createdBy))
}

func (r *assetRepository) GetsByTags(ctx context.Context, tag string, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, *query.NewOption("tags LIKE ?", "%"+tag+"%"))
}

func (r *assetRepository) GetPublicAssets(ctx context.Context, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order,
		*query.NewOption("is_public = ?", true),
		*query.NewOption("status = ?", asset.StatusActive),
	)
}

// ========================
// 按对象键批量操作
// ========================

func (r *assetRepository) DeleteByObjectKey(ctx context.Context, objectKey string) error {
	return r.db.WithContext(ctx).Where("object_key = ?", objectKey).Delete(&models.AssetModel{}).Error
}

func (r *assetRepository) BatchDeleteByObjectKeys(ctx context.Context, objectKeys []string) error {
	return r.db.WithContext(ctx).Where("object_key IN ?", objectKeys).Delete(&models.AssetModel{}).Error
}

// ========================
// 统计查询
// ========================

func (r *assetRepository) CountByCategory(ctx context.Context) (map[asset.AssetCategory]int64, error) {
	type result struct {
		Category asset.AssetCategory
		Count    int64
	}
	var results []result
	if err := r.db.WithContext(ctx).Model(&models.AssetModel{}).
		Select("category, COUNT(*) as count").Group("category").Find(&results).Error; err != nil {
		return nil, err
	}
	stats := make(map[asset.AssetCategory]int64)
	for _, r := range results {
		stats[r.Category] = r.Count
	}
	return stats, nil
}

func (r *assetRepository) CountByStatus(ctx context.Context) (map[asset.AssetStatus]int64, error) {
	type result struct {
		Status asset.AssetStatus
		Count  int64
	}
	var results []result
	if err := r.db.WithContext(ctx).Model(&models.AssetModel{}).
		Select("status, COUNT(*) as count").Group("status").Find(&results).Error; err != nil {
		return nil, err
	}
	stats := make(map[asset.AssetStatus]int64)
	for _, r := range results {
		stats[r.Status] = r.Count
	}
	return stats, nil
}

func (r *assetRepository) SumSize(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.AssetModel{}).
		Select("COALESCE(SUM(size), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *assetRepository) GetCombinedStats(ctx context.Context) (map[asset.AssetCategory]int64, map[asset.AssetStatus]int64, int64, error) {
	type combinedResult struct {
		Category asset.AssetCategory
		Status   asset.AssetStatus
		Count    int64
		Size     int64
	}
	var results []combinedResult
	if err := r.db.WithContext(ctx).Model(&models.AssetModel{}).
		Select("category, status, COUNT(*) as count, COALESCE(SUM(size), 0) as size").
		Group("category, status").Find(&results).Error; err != nil {
		return nil, nil, 0, err
	}
	categoryStats := make(map[asset.AssetCategory]int64)
	statusStats := make(map[asset.AssetStatus]int64)
	var totalSize int64
	for _, r := range results {
		categoryStats[r.Category] += r.Count
		statusStats[r.Status] += r.Count
		totalSize += r.Size
	}
	return categoryStats, statusStats, totalSize, nil
}

func (r *assetRepository) SumSizeByCreatedBy(ctx context.Context, createdBy uint) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.AssetModel{}).
		Where("created_by = ?", createdBy).Select("COALESCE(SUM(size), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// ========================
// 更新操作
// ========================

func (r *assetRepository) UpdateStatus(ctx context.Context, id uint, status asset.AssetStatus) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("status", status).Error
}

func (r *assetRepository) UpdateIsPublic(ctx context.Context, id uint, isPublic bool) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("is_public", isPublic).Error
}

func (r *assetRepository) IncrementDownloadCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

func (r *assetRepository) IncrementViewCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (r *assetRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id IN ?", ids).Update("status", status).Error
}

func (r *assetRepository) MoveToFolder(ctx context.Context, id uint, folderID *uint, folderPath string) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"folder_id": folderID, "folder_path": folderPath}).Error
}

// ========================
// 通用分页辅助
// ========================

// convertPagedModels 将持久化模型列表转换为 Domain 实体列表
func convertPagedModels[E any, M interface{ ToEntity() *E }](models []M) []E {
	result := make([]E, len(models))
	for i, m := range models {
		result[i] = *m.ToEntity()
	}
	return result
}

// filteredPaginatedQuery 带 Domain/Model 类型转换的分页查询
func filteredPaginatedQuery[E any, M interface{ ToEntity() *E }](
	ctx context.Context, db *gorm.DB, page, size int, order string, opts ...query.Option,
) ([]E, int64, error) {
	if order == "" {
		order = "created_at DESC"
	}
	ms, total, err := PaginatedQuery[M](ctx, db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	return convertPagedModels[E, M](ms), total, nil
}
