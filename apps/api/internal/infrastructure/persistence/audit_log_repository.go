package persistence

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
)

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) audit_log.Repository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]audit_log.AuditLog, int64, error) {
	ms, total, err := PaginatedQuery[models.AuditLogModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	result := make([]audit_log.AuditLog, len(ms))
	for i, m := range ms {
		result[i] = *m.ToEntity()
	}
	return result, total, nil
}

func (r *auditLogRepository) Get(ctx context.Context, id uint) (*audit_log.AuditLog, error) {
	var m models.AuditLogModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *auditLogRepository) Create(ctx context.Context, item *audit_log.AuditLog) error {
	m := models.AuditLogModelFromEntity(item)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	return nil
}

func (r *auditLogRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&models.AuditLogModel{})
	return result.RowsAffected, result.Error
}
