package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// ========== DictTypeRepository ==========

type dictTypeRepository struct {
	db *gorm.DB
}

// NewDictTypeRepository 创建字典类型仓储
func NewDictTypeRepository(db *gorm.DB) dictionary.DictTypeRepository {
	return &dictTypeRepository{db: db}
}

func (r *dictTypeRepository) Gets(ctx context.Context) ([]dictionary.DictType, error) {
	var ms []models.DictTypeModel
	if err := r.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]dictionary.DictType, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *dictTypeRepository) Get(ctx context.Context, id uint) (*dictionary.DictType, error) {
	var m models.DictTypeModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *dictTypeRepository) GetByCode(ctx context.Context, code string) (*dictionary.DictType, error) {
	var m models.DictTypeModel
	if err := translateError(r.db.WithContext(ctx).Where("code = ?", code).First(&m).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *dictTypeRepository) Create(ctx context.Context, item *dictionary.DictType) error {
	m := models.DictTypeModelFromEntity(item)
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

func (r *dictTypeRepository) Update(ctx context.Context, item *dictionary.DictType) error {
	m := models.DictTypeModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *dictTypeRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.DictTypeModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *dictTypeRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.DictTypeModel{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ========== DictItemRepository ==========

type dictItemRepository struct {
	db *gorm.DB
}

// NewDictItemRepository 创建字典项仓储
func NewDictItemRepository(db *gorm.DB) dictionary.DictItemRepository {
	return &dictItemRepository{db: db}
}

func (r *dictItemRepository) Gets(ctx context.Context, typeCode string) ([]dictionary.DictItem, error) {
	var ms []models.DictItemModel
	q := r.db.WithContext(ctx).Order("sort_order ASC, id ASC")
	if typeCode != "" {
		q = q.Where("type_code = ?", typeCode)
	}
	if err := q.Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]dictionary.DictItem, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *dictItemRepository) Get(ctx context.Context, id uint) (*dictionary.DictItem, error) {
	var m models.DictItemModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *dictItemRepository) GetByTypeCode(ctx context.Context, typeCode string) ([]dictionary.DictItem, error) {
	var ms []models.DictItemModel
	if err := r.db.WithContext(ctx).Where("type_code = ?", typeCode).Order("sort_order ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]dictionary.DictItem, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *dictItemRepository) GetEnabledByTypeCode(ctx context.Context, typeCode string) ([]dictionary.DictItem, error) {
	var ms []models.DictItemModel
	if err := r.db.WithContext(ctx).Where("type_code = ? AND is_enabled = ?", typeCode, true).Order("sort_order ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]dictionary.DictItem, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *dictItemRepository) Create(ctx context.Context, item *dictionary.DictItem) error {
	m := models.DictItemModelFromEntity(item)
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

func (r *dictItemRepository) Update(ctx context.Context, item *dictionary.DictItem) error {
	m := models.DictItemModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *dictItemRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.DictItemModel{}, id).Error
}

func (r *dictItemRepository) DeleteByTypeCode(ctx context.Context, typeCode string) error {
	return r.db.WithContext(ctx).Where("type_code = ?", typeCode).Delete(&models.DictItemModel{}).Error
}

func (r *dictItemRepository) BatchCreate(ctx context.Context, items []dictionary.DictItem) error {
	ms := make([]models.DictItemModel, len(items))
	for i := range items {
		ms[i] = *models.DictItemModelFromEntity(&items[i])
	}
	if err := r.db.WithContext(ctx).Create(&ms).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].ID = ms[i].ID
		items[i].CreatedAt = ms[i].CreatedAt
		items[i].UpdatedAt = ms[i].UpdatedAt
		items[i].CreatedBy = ms[i].CreatedBy
		items[i].UpdatedBy = ms[i].UpdatedBy
	}
	return nil
}

func (r *dictItemRepository) ExistsByTypeCodeAndValue(ctx context.Context, typeCode, value string, excludeID uint) (bool, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&models.DictItemModel{}).Where("type_code = ? AND value = ?", typeCode, value)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *dictItemRepository) GetAllEnabled(ctx context.Context) ([]dictionary.DictItem, error) {
	var ms []models.DictItemModel
	if err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Order("type_code ASC, sort_order ASC, id ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]dictionary.DictItem, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}
