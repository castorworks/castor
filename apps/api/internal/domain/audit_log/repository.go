package audit_log

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/pkg/query"
)

// Repository 审计日志仓储接口
type Repository interface {
	// Gets 分页查询审计日志列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]AuditLog, int64, error)
	// Get 根据 ID 获取审计日志
	Get(ctx context.Context, id uint) (*AuditLog, error)
	// Create 创建审计日志
	Create(ctx context.Context, auditLog *AuditLog) error
	// DeleteBefore 删除指定时间之前的审计日志（数据保留策略）
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
