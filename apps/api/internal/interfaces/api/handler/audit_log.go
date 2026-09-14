package handler

import (
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
}

// NewAdminAuditLogHandler 创建管理员审计日志处理器
func NewAdminAuditLogHandler(svc service.AuditLogService) *AdminAuditLogHandler {
	return &AdminAuditLogHandler{AuditLogService: svc}
}

// Gets 获取审计日志列表
func (h *AdminAuditLogHandler) Gets(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(audit_log.AuditLog{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.AuditLogService.Gets(ctx, page, size, order, opts...)
	})
}

// DeleteBefore 按保留天数清理审计日志
func (h *AdminAuditLogHandler) DeleteBefore(c *gin.Context) {
	var req dto.AuditLogDeleteBeforeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	before := time.Now().AddDate(0, 0, -req.RetentionDays)
	deleted, err := h.AuditLogService.DeleteBefore(c.Request.Context(), before)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, &dto.AuditLogDeleteBeforeResp{Deleted: deleted})
}
