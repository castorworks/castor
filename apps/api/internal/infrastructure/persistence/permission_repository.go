package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type resourceRepository struct{ db *gorm.DB }

func NewResourceRepository(db *gorm.DB) permission.ResourceRepository {
	return &resourceRepository{db: db}
}

func resourcesFromModels(rows []models.ResourceModel) []permission.Resource {
	result := make([]permission.Resource, len(rows))
	for i := range rows {
		result[i] = *rows[i].ToEntity()
	}
	return result
}

func (r *resourceRepository) GetAll(ctx context.Context) ([]permission.Resource, error) {
	var rows []models.ResourceModel
	if err := r.db.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return resourcesFromModels(rows), nil
}

func (r *resourceRepository) GetAllEnabled(ctx context.Context) ([]permission.Resource, error) {
	var rows []models.ResourceModel
	if err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return resourcesFromModels(rows), nil
}

func (r *resourceRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]permission.Resource, int64, error) {
	rows, total, err := PaginatedQuery[models.ResourceModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	return resourcesFromModels(rows), total, nil
}

func (r *resourceRepository) Get(ctx context.Context, id uint) (*permission.Resource, error) {
	var row models.ResourceModel
	err := translateError(r.db.WithContext(ctx).First(&row, id).Error)
	return row.ToEntity(), err
}

func (r *resourceRepository) GetByCode(ctx context.Context, code string) (*permission.Resource, error) {
	var row models.ResourceModel
	err := translateError(r.db.WithContext(ctx).Where("code = ?", code).First(&row).Error)
	return row.ToEntity(), err
}

func (r *resourceRepository) GetByPath(ctx context.Context, path string) (*permission.Resource, error) {
	var row models.ResourceModel
	err := translateError(r.db.WithContext(ctx).Where("path = ?", path).First(&row).Error)
	return row.ToEntity(), err
}

func (r *resourceRepository) GetByCategory(ctx context.Context, category permission.ResourceCategory) ([]permission.Resource, error) {
	var rows []models.ResourceModel
	if err := r.db.WithContext(ctx).Where("category = ? AND is_enabled = ?", category, true).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return resourcesFromModels(rows), nil
}

func (r *resourceRepository) GetByModule(ctx context.Context, module string) ([]permission.Resource, error) {
	var rows []models.ResourceModel
	if err := r.db.WithContext(ctx).Where("module = ? AND is_enabled = ?", module, true).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return resourcesFromModels(rows), nil
}

func (r *resourceRepository) GetByIDs(ctx context.Context, ids []uint) ([]permission.Resource, error) {
	if len(ids) == 0 {
		return []permission.Resource{}, nil
	}
	var rows []models.ResourceModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	return resourcesFromModels(rows), nil
}

func (r *resourceRepository) Create(ctx context.Context, resource *permission.Resource) error {
	row := models.ResourceModelFromEntity(resource)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	*resource = *row.ToEntity()
	return nil
}

func (r *resourceRepository) BatchCreate(ctx context.Context, resources []permission.Resource) error {
	if len(resources) == 0 {
		return nil
	}
	rows := make([]models.ResourceModel, len(resources))
	for i := range resources {
		rows[i] = *models.ResourceModelFromEntity(&resources[i])
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoNothing: true}).CreateInBatches(&rows, 100).Error; err != nil {
		return err
	}
	for i := range resources {
		row, err := r.GetByCode(ctx, resources[i].Code)
		if err != nil {
			return err
		}
		resources[i] = *row
	}
	return nil
}

func (r *resourceRepository) Update(ctx context.Context, resource *permission.Resource) error {
	row := models.ResourceModelFromEntity(resource)
	if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return err
	}
	*resource = *row.ToEntity()
	return nil
}

func (r *resourceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ResourceModel{}, id).Error
}

func (r *resourceRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ResourceModel{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) permission.RoleRepository {
	return &roleRepository{db: db}
}

func rolesFromModels(rows []models.RoleModel) []permission.Role {
	result := make([]permission.Role, len(rows))
	for i := range rows {
		result[i] = *rows[i].ToEntity()
	}
	return result
}

func (r *roleRepository) GetAll(ctx context.Context) ([]permission.Role, error) {
	var rows []models.RoleModel
	if err := r.db.WithContext(ctx).Order("code").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rolesFromModels(rows), nil
}

func (r *roleRepository) GetAllEnabled(ctx context.Context) ([]permission.Role, error) {
	var rows []models.RoleModel
	if err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Order("code").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rolesFromModels(rows), nil
}

func (r *roleRepository) Get(ctx context.Context, id uint) (*permission.Role, error) {
	var row models.RoleModel
	err := translateError(r.db.WithContext(ctx).First(&row, id).Error)
	return row.ToEntity(), err
}

func (r *roleRepository) GetByCode(ctx context.Context, code string) (*permission.Role, error) {
	var row models.RoleModel
	err := translateError(r.db.WithContext(ctx).Where("code = ?", code).First(&row).Error)
	return row.ToEntity(), err
}

func (r *roleRepository) Create(ctx context.Context, role *permission.Role) error {
	row := models.RoleModelFromEntity(role)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	*role = *row.ToEntity()
	return nil
}

func (r *roleRepository) BatchCreate(ctx context.Context, roles []permission.Role) error {
	if len(roles) == 0 {
		return nil
	}
	rows := make([]models.RoleModel, len(roles))
	for i := range roles {
		rows[i] = *models.RoleModelFromEntity(&roles[i])
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoNothing: true}).CreateInBatches(&rows, 100).Error; err != nil {
		return err
	}
	for i := range roles {
		row, err := r.GetByCode(ctx, roles[i].Code)
		if err != nil {
			return err
		}
		roles[i] = *row
	}
	return nil
}

func (r *roleRepository) Update(ctx context.Context, role *permission.Role) error {
	row := models.RoleModelFromEntity(role)
	if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return err
	}
	*role = *row.ToEntity()
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RoleModel{}, id).Error
}

func (r *roleRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}
