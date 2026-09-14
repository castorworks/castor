package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

type menuRepository struct{ db *gorm.DB }

func NewMenuRepository(db *gorm.DB) menu.Repository { return &menuRepository{db: db} }
func (r *menuRepository) WithTx(ctx context.Context, fn func(menu.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&menuRepository{db: tx}) })
}
func (r *menuRepository) Lock(ctx context.Context) error {
	// Serializes even the first insertion into an empty menu catalog.
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", int64(0x4d454e55)).Error
}
func (r *menuRepository) List(ctx context.Context) ([]menu.Menu, error) {
	var rows []models.MenuModel
	if err := r.db.WithContext(ctx).Preload("Permissions").Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]menu.Menu, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ToEntity())
	}
	return result, nil
}
func (r *menuRepository) Save(ctx context.Context, m *menu.Menu) error {
	row := models.MenuModelFromEntity(m)
	q := r.db.WithContext(ctx).Omit("Parent", "Permissions")
	var err error
	if m.ID == 0 {
		err = q.Create(row).Error
	} else {
		err = q.Save(row).Error
	}
	if err != nil {
		return err
	}
	if err = r.db.WithContext(ctx).Where("menu_id = ?", row.ID).Delete(&models.MenuPermissionModel{}).Error; err != nil {
		return err
	}
	refs := make([]models.MenuPermissionModel, 0, len(m.Permissions))
	for _, p := range m.Permissions {
		refs = append(refs, models.MenuPermissionModel{MenuID: row.ID, ResourceID: p.ResourceID, Action: p.Action})
	}
	if len(refs) > 0 {
		if err = r.db.WithContext(ctx).Create(&refs).Error; err != nil {
			return err
		}
	}
	refsEntity := m.Permissions
	*m = row.ToEntity()
	m.Permissions = refsEntity
	return nil
}
func (r *menuRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.MenuModel{}, id).Error
}
