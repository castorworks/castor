package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"gorm.io/gorm"
)

type settingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 创建系统配置仓储
func NewSettingRepository(db *gorm.DB) setting.Repository {
	return &settingRepository{db: db}
}

func (r *settingRepository) Gets(ctx context.Context, category string) ([]setting.Setting, error) {
	var ms []models.SettingModel
	q := r.db.WithContext(ctx).Order("sort_order ASC, id ASC")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]setting.Setting, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *settingRepository) Get(ctx context.Context, id uint) (*setting.Setting, error) {
	var m models.SettingModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *settingRepository) GetByKey(ctx context.Context, key string) (*setting.Setting, error) {
	var m models.SettingModel
	if err := translateError(r.db.WithContext(ctx).Where("key = ?", key).First(&m).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *settingRepository) Create(ctx context.Context, item *setting.Setting) error {
	m := models.SettingModelFromEntity(item)
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

func (r *settingRepository) Update(ctx context.Context, item *setting.Setting) error {
	m := models.SettingModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *settingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.SettingModel{}, id).Error
}

func (r *settingRepository) GetByKeys(ctx context.Context, keys []string) ([]setting.Setting, error) {
	var ms []models.SettingModel
	if err := r.db.WithContext(ctx).Where("key IN ?", keys).Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]setting.Setting, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *settingRepository) BatchUpdate(ctx context.Context, settings []setting.Setting) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, s := range settings {
			updates := map[string]interface{}{
				"value": s.Value,
			}
			// 从 context 中提取操作者 ID 设置 updated_by
			if userID := ucontext.AuditUserIDFromContext(ctx); userID != 0 {
				updates["updated_by"] = userID
			}
			if err := tx.Model(&models.SettingModel{}).Where("key = ?", s.Key).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *settingRepository) GetPublicSettings(ctx context.Context) ([]setting.Setting, error) {
	var ms []models.SettingModel
	if err := r.db.WithContext(ctx).Where("is_public = ?", true).Order("sort_order ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]setting.Setting, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *settingRepository) GetByCategory(ctx context.Context, category setting.SettingCategory) ([]setting.Setting, error) {
	var ms []models.SettingModel
	if err := r.db.WithContext(ctx).Where("category = ?", category).Order("sort_order ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]setting.Setting, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

func (r *settingRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SettingModel{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
