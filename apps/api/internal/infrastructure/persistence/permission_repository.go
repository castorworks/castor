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

// rolesWithDepartments 转换角色并补上 CUSTOM 数据范围指定的部门（一次查询取全）。
func (r *roleRepository) rolesWithDepartments(ctx context.Context, rows []models.RoleModel) ([]permission.Role, error) {
	result := make([]permission.Role, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]uint, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var links []models.RoleDepartmentModel
	if err := r.db.WithContext(ctx).Where("role_id IN ?", ids).Order("department_id").Find(&links).Error; err != nil {
		return nil, err
	}
	byRole := make(map[uint][]uint, len(rows))
	for _, link := range links {
		byRole[link.RoleID] = append(byRole[link.RoleID], link.DepartmentID)
	}
	for i := range rows {
		result[i] = *rows[i].ToEntity()
		if depts := byRole[rows[i].ID]; depts != nil {
			result[i].DepartmentIDs = depts
		}
	}
	return result, nil
}

func (r *roleRepository) roleWithDepartments(ctx context.Context, row models.RoleModel) (*permission.Role, error) {
	roles, err := r.rolesWithDepartments(ctx, []models.RoleModel{row})
	if err != nil {
		return nil, err
	}
	return &roles[0], nil
}

// saveDepartments 让 CUSTOM 范围的部门集合恰好等于 role.DepartmentIDs；其它范围不保留部门。
func saveDepartments(tx *gorm.DB, role *permission.Role, roleID uint) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleDepartmentModel{}).Error; err != nil {
		return err
	}
	if role.DataScope != permission.DataScopeCustom || len(role.DepartmentIDs) == 0 {
		return nil
	}
	links := make([]models.RoleDepartmentModel, len(role.DepartmentIDs))
	for i, id := range role.DepartmentIDs {
		links[i] = models.RoleDepartmentModel{RoleID: roleID, DepartmentID: id}
	}
	return tx.Omit("Role", "Department").Create(&links).Error
}

func (r *roleRepository) GetAll(ctx context.Context) ([]permission.Role, error) {
	var rows []models.RoleModel
	if err := r.db.WithContext(ctx).Order("code").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.rolesWithDepartments(ctx, rows)
}

func (r *roleRepository) GetAllEnabled(ctx context.Context) ([]permission.Role, error) {
	var rows []models.RoleModel
	if err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Order("code").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.rolesWithDepartments(ctx, rows)
}

func (r *roleRepository) Get(ctx context.Context, id uint) (*permission.Role, error) {
	var row models.RoleModel
	if err := translateError(r.db.WithContext(ctx).First(&row, id).Error); err != nil {
		return row.ToEntity(), err
	}
	return r.roleWithDepartments(ctx, row)
}

func (r *roleRepository) GetByCode(ctx context.Context, code string) (*permission.Role, error) {
	var row models.RoleModel
	if err := translateError(r.db.WithContext(ctx).Where("code = ?", code).First(&row).Error); err != nil {
		return row.ToEntity(), err
	}
	return r.roleWithDepartments(ctx, row)
}

func (r *roleRepository) Create(ctx context.Context, role *permission.Role) error {
	row := models.RoleModelFromEntity(role)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		if err := saveDepartments(tx, role, row.ID); err != nil {
			return err
		}
		depts := role.DepartmentIDs
		*role = *row.ToEntity()
		if role.DataScope == permission.DataScopeCustom && depts != nil {
			role.DepartmentIDs = depts
		}
		return nil
	})
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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		if err := saveDepartments(tx, role, row.ID); err != nil {
			return err
		}
		depts := role.DepartmentIDs
		*role = *row.ToEntity()
		if role.DataScope == permission.DataScopeCustom && depts != nil {
			role.DepartmentIDs = depts
		}
		return nil
	})
}

func (r *roleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RoleModel{}, id).Error
}

func (r *roleRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}
