package persistence

import (
	"context"
	"fmt"

	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实现
func NewUserRepository(db *gorm.DB) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]user.User, int64, error) {
	models, total, err := PaginatedQuery[models.UserModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	result := make([]user.User, len(models))
	for i, m := range models {
		result[i] = *m.ToEntity()
	}
	return result, total, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var m models.UserModel
	if err := translateError(r.db.WithContext(ctx).Where("username = ?", username).First(&m).Error); err != nil {
		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}
	return m.ToEntity(), nil
}

func (r *userRepository) Get(ctx context.Context, id uint) (*user.User, error) {
	var m models.UserModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	return m.ToEntity(), nil
}

func (r *userRepository) Create(ctx context.Context, item *user.User) error {
	m := models.UserModelFromEntity(item)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	// 回写自增 ID 和时间戳
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	item.CreatedBy = m.CreatedBy
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *userRepository) Update(ctx context.Context, item *user.User) error {
	m := models.UserModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update user %d: %w", item.ID, err)
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.UserModel{}, id).Error; err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	return nil
}
