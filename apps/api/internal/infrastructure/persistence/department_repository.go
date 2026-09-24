package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

type departmentRepository struct{ db *gorm.DB }

func NewDepartmentRepository(db *gorm.DB) department.Repository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) WithTx(ctx context.Context, fn func(department.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&departmentRepository{db: tx}) })
}

func (r *departmentRepository) Lock(ctx context.Context) error {
	// 串行化整棵树的写入，包括空表上的第一次插入。
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", int64(0x44455054)).Error
}

func (r *departmentRepository) List(ctx context.Context) ([]department.Department, error) {
	var rows []models.DepartmentModel
	if err := r.db.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]department.Department, len(rows))
	for i := range rows {
		result[i] = rows[i].ToEntity()
	}
	return result, nil
}

func (r *departmentRepository) Save(ctx context.Context, d *department.Department) error {
	row := models.DepartmentModelFromEntity(d)
	q := r.db.WithContext(ctx).Omit("Parent")
	var err error
	if d.ID == 0 {
		err = q.Create(row).Error
	} else {
		err = q.Save(row).Error
	}
	if err != nil {
		return err
	}
	*d = row.ToEntity()
	return nil
}

func (r *departmentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.DepartmentModel{}, id).Error
}
