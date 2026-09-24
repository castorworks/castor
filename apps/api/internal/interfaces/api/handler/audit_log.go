package handler

import (
	"fmt"
	"reflect"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
)

// AdminAuditLogHandler 管理员审计日志处理器
type AdminAuditLogHandler struct {
	AuditLogService service.AuditLogService
	RBAC            service.RBACService
	Dictionary      service.DictionaryService
}

// NewAdminAuditLogHandler 创建管理员审计日志处理器
func NewAdminAuditLogHandler(svc service.AuditLogService, rbac service.RBACService, dictionary service.DictionaryService) *AdminAuditLogHandler {
	return &AdminAuditLogHandler{AuditLogService: svc, RBAC: rbac, Dictionary: dictionary}
}

// Gets 获取审计日志列表
func (h *AdminAuditLogHandler) Gets(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericGets(c, reflect.TypeOf(audit_log.AuditLog{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.AuditLogService.Gets(ctx, scope, page, size, order, opts...)
	})
}

// DeleteBefore 按保留天数清理审计日志
func (h *AdminAuditLogHandler) DeleteBefore(c *gin.Context) {
	var req dto.AuditLogDeleteBeforeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	before := time.Now().AddDate(0, 0, -req.RetentionDays)
	deleted, err := h.AuditLogService.DeleteBefore(c.Request.Context(), scope, before)
	// 清理审计日志会抹掉证据链，因此清理动作本身必须先留下一条审计记录。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAuditLogCleanup, fmt.Sprintf("retentionDays=%d", req.RetentionDays),
		fmt.Sprintf("Clean up audit logs older than %d days (%d deleted)", req.RetentionDays, deleted), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, &dto.AuditLogDeleteBeforeResp{Deleted: deleted})
}

// Export 按审计日志列表相同的筛选与数据范围导出
// GET /api/v1/admin/audit-logs/export
func (h *AdminAuditLogHandler) Export(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	x := newExportContext(c, h.Dictionary)
	columns := []exportColumn[audit_log.AuditLog]{
		{"ColumnTime", func(l audit_log.AuditLog) string { return x.time(l.CreatedAt) }},
		{"ColumnOperator", func(l audit_log.AuditLog) string { return l.Operator }},
		{"ColumnLogType", func(l audit_log.AuditLog) string { return x.label("audit_log_type", string(l.LogType)) }},
		{"ColumnTarget", func(l audit_log.AuditLog) string { return l.Target }},
		{"ColumnDetails", func(l audit_log.AuditLog) string { return l.Details }},
		{"ColumnIpAddr", func(l audit_log.AuditLog) string { return l.IpAddr }},
		{"ColumnResult", func(l audit_log.AuditLog) string { return x.result(l.Success) }},
	}
	exportList(c, reflect.TypeOf(audit_log.AuditLog{}), "audit-logs", columns, func(ctx *gin.Context, page, size int, order string, opts ...query.Option) ([]audit_log.AuditLog, int64, error) {
		return h.AuditLogService.Gets(ctx, scope, page, size, order, opts...)
	}, func(rows int, err error) {
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAuditLogExport, "-", fmt.Sprintf("Export %d audit logs", rows), err)
	})
}
