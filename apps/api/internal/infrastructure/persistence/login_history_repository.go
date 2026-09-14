package persistence

import (
	"context"

	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
)

type loginHistoryRepository struct {
	db *gorm.DB
}

// NewLoginHistoryRepository 创建登录历史仓储实现
func NewLoginHistoryRepository(db *gorm.DB) login_history.Repository {
	repo := &loginHistoryRepository{db: db}
	return repo
}

func (r *loginHistoryRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]login_history.LoginHistory, int64, error) {
	ms, total, err := PaginatedQuery[models.LoginHistoryModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	result := make([]login_history.LoginHistory, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, total, nil
}

func (r *loginHistoryRepository) Get(ctx context.Context, id uint) (*login_history.LoginHistory, error) {
	var m models.LoginHistoryModel
	err := translateError(r.db.WithContext(ctx).First(&m, id).Error)
	return m.ToEntity(), err
}

func (r *loginHistoryRepository) Create(ctx context.Context, item *login_history.LoginHistory) error {
	m := models.LoginHistoryModelFromEntity(item)
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

func (r *loginHistoryRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.LoginHistoryModel{}, id).Error
}

func (r *loginHistoryRepository) BatchDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Delete(&models.LoginHistoryModel{}, ids).Error
}
