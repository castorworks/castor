package persistence

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return r.BatchDelete(ctx, []uint{id})
}

func (r *assetRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	// 不连带删除引用：asset_references.asset_id 的外键（ON DELETE RESTRICT）正是"被引用不可删"的最终防线。
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.AssetModel{}).Error
}

// ========================
// 查询
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

func (r *assetRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order, opts...)
}

func (r *assetRepository) GetPublicAssets(ctx context.Context, page, size int, order string) ([]asset.Asset, int64, error) {
	return filteredPaginatedQuery[asset.Asset, models.AssetModel](ctx, r.db, page, size, order,
		*query.NewOption("is_public = ?", true),
		*query.NewOption("status = ?", asset.StatusActive),
		*query.NewOption("scope = ?", asset.ScopeLibrary),
	)
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

// ========================
// 更新操作
// ========================

func (r *assetRepository) UpdateStatus(ctx context.Context, id uint, status asset.AssetStatus) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("status", status).Error
}

func (r *assetRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id IN ?", ids).Update("status", status).Error
}

func (r *assetRepository) UpdateScope(ctx context.Context, id uint, scope asset.AssetScope) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("scope", scope).Error
}

func (r *assetRepository) UpdateIsPublic(ctx context.Context, id uint, isPublic bool) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("is_public", isPublic).Error
}

func (r *assetRepository) UpdateFolderPath(ctx context.Context, id uint, folderPath string) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).Update("folder_path", folderPath).Error
}

func (r *assetRepository) IncrementDownloadCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

// ========================
// 引用
// ========================

func (r *assetRepository) AddReference(ctx context.Context, assetID uint, ref asset.Reference, name string) error {
	m := &models.AssetReferenceModel{AssetID: assetID, OwnerType: ref.OwnerType, OwnerID: ref.OwnerID, Field: ref.Field, Name: name}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "owner_type"}, {Name: "owner_id"}, {Name: "field"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(m).Error
}

func (r *assetRepository) RemoveReference(ctx context.Context, assetID uint, ref asset.Reference) error {
	return r.db.WithContext(ctx).
		Where("asset_id = ? AND owner_type = ? AND owner_id = ? AND field = ?", assetID, ref.OwnerType, ref.OwnerID, ref.Field).
		Delete(&models.AssetReferenceModel{}).Error
}

func (r *assetRepository) GetReferences(ctx context.Context, assetID uint) ([]asset.Reference, error) {
	var ms []models.AssetReferenceModel
	if err := r.db.WithContext(ctx).Where("asset_id = ?", assetID).Order("id ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	refs := make([]asset.Reference, len(ms))
	for i, m := range ms {
		refs[i] = m.ToEntity()
	}
	return refs, nil
}

func (r *assetRepository) CountReferences(ctx context.Context, assetID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AssetReferenceModel{}).Where("asset_id = ?", assetID).Count(&count).Error
	return count, err
}

func (r *assetRepository) GetReferencedIDs(ctx context.Context, ids []uint) ([]uint, error) {
	var referenced []uint
	if len(ids) == 0 {
		return referenced, nil
	}
	err := r.db.WithContext(ctx).Model(&models.AssetReferenceModel{}).
		Where("asset_id IN ?", ids).Distinct().Pluck("asset_id", &referenced).Error
	return referenced, err
}

func (r *assetRepository) GetIDsByReference(ctx context.Context, ref asset.Reference) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&models.AssetReferenceModel{}).
		Where("owner_type = ? AND owner_id = ? AND field = ?", ref.OwnerType, ref.OwnerID, ref.Field).
		Pluck("asset_id", &ids).Error
	return ids, err
}

func (r *assetRepository) GetReferenceName(ctx context.Context, assetID uint, ref asset.Reference) (string, error) {
	var m models.AssetReferenceModel
	err := translateError(r.db.WithContext(ctx).
		Where("asset_id = ? AND owner_type = ? AND owner_id = ? AND field = ?", assetID, ref.OwnerType, ref.OwnerID, ref.Field).
		First(&m).Error)
	return m.Name, err
}

func (r *assetRepository) GetAttached(ctx context.Context, field asset.Field, ownerIDs []uint) ([]asset.Attached, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	var refs []models.AssetReferenceModel
	if err := r.db.WithContext(ctx).
		Where("owner_type = ? AND field = ? AND owner_id IN ?", field.OwnerType, field.Name, ownerIDs).
		Order("id ASC").Find(&refs).Error; err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}
	assetIDs := make([]uint, len(refs))
	for i, ref := range refs {
		assetIDs[i] = ref.AssetID
	}
	var ms []models.AssetModel
	if err := r.db.WithContext(ctx).Where("id IN ?", assetIDs).Find(&ms).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint]*asset.Asset, len(ms))
	for _, m := range ms {
		byID[m.ID] = m.ToEntity()
	}
	attached := make([]asset.Attached, 0, len(refs))
	for _, ref := range refs {
		if item, ok := byID[ref.AssetID]; ok {
			attached = append(attached, asset.Attached{OwnerID: ref.OwnerID, Name: ref.Name, Asset: *item})
		}
	}
	return attached, nil
}

func (r *assetRepository) GetIDsByOwner(ctx context.Context, ownerType string, ownerID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&models.AssetReferenceModel{}).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Distinct().Pluck("asset_id", &ids).Error
	return ids, err
}

func (r *assetRepository) GetOrphanAttachments(ctx context.Context, before time.Time, limit int) ([]asset.Asset, error) {
	var ms []models.AssetModel
	err := r.db.WithContext(ctx).
		Where("scope = ? AND updated_at < ?", asset.ScopeAttachment, before).
		Where("NOT EXISTS (SELECT 1 FROM asset_references r WHERE r.asset_id = assets.id)").
		Order("id ASC").Limit(limit).Find(&ms).Error
	if err != nil {
		return nil, err
	}
	return convertPagedModels[asset.Asset, models.AssetModel](ms), nil
}

func (r *assetRepository) Touch(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AssetModel{}).Where("id = ?", id).
		UpdateColumn("updated_at", time.Now()).Error
}

func (r *assetRepository) RemoveOwnerReferences(ctx context.Context, ownerType string, ownerID uint) error {
	return r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Delete(&models.AssetReferenceModel{}).Error
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
