package service

import (
	"context"
	"sync"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/hyperits/gosuite/logger"
)

// AuditLogService 审计日志应用服务接口
type AuditLogService interface {
	// Gets 按调用者数据范围列出审计日志：只看得到范围内用户（按操作者）的操作
	Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]audit_log.AuditLog, int64, error)
	Log(ctx context.Context, log *audit_log.AuditLog) error
	LogAsync(log *audit_log.AuditLog)
	// DeleteBefore 删除指定时间之前的审计日志（数据保留策略）；会删到所有人的日志，要求全部数据范围
	DeleteBefore(ctx context.Context, scope permission.AccessScope, before time.Time) (int64, error)
	// Wait 等待所有异步审计日志写入完成
	Wait()
}

type auditLogService struct {
	auditLogRepo audit_log.Repository
	wg           sync.WaitGroup
}

// NewAuditLogService 创建审计日志应用服务
func NewAuditLogService(auditLogRepo audit_log.Repository) AuditLogService {
	return &auditLogService{
		auditLogRepo: auditLogRepo,
	}
}

func (svc *auditLogService) Gets(ctx context.Context, scope permission.AccessScope, page, size int, order string, opts ...query.Option) ([]audit_log.AuditLog, int64, error) {
	return svc.auditLogRepo.Gets(ctx, page, size, order, withUserScope(opts, scope, "operator_id")...)
}

func (svc *auditLogService) Log(ctx context.Context, log *audit_log.AuditLog) error {
	return svc.auditLogRepo.Create(ctx, log)
}

// LogAsync 异步记录审计日志（不依赖请求 context，内部创建独立超时 context）
func (svc *auditLogService) LogAsync(log *audit_log.AuditLog) {
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := svc.auditLogRepo.Create(ctx, log); err != nil {
			logger.Errorf("Failed to create audit log: type=%s, operator=%s, err=%v", log.LogType, log.Operator, err)
		}
	}()
}

// DeleteBefore 删除指定时间之前的审计日志
func (svc *auditLogService) DeleteBefore(ctx context.Context, scope permission.AccessScope, before time.Time) (int64, error) {
	if !scope.All {
		return 0, apperror.ErrDataScopeExceedsCaller
	}
	return svc.auditLogRepo.DeleteBefore(ctx, before)
}

// Wait 等待所有异步审计日志写入完成
func (svc *auditLogService) Wait() {
	svc.wg.Wait()
}
