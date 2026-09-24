package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminSessionHandler 在线会话处理器
type AdminSessionHandler struct {
	SessionService  service.SessionService
	AuditLogService service.AuditLogService
	RBAC            service.RBACService
}

// NewAdminSessionHandler 创建在线会话处理器
func NewAdminSessionHandler(sessionService service.SessionService, auditLogService service.AuditLogService, rbac service.RBACService) *AdminSessionHandler {
	return &AdminSessionHandler{SessionService: sessionService, AuditLogService: auditLogService, RBAC: rbac}
}

// Gets 分页列出在线会话
// GET /api/v1/admin/sessions
func (h *AdminSessionHandler) Gets(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	current := ucontext.GetAuthorizationSessionID(c)
	GenericGets(c, reflect.TypeOf(permission.AuthorizationSession{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.SessionService.Gets(ctx, scope, current, page, size, order, opts...)
	})
}

// Revoke 强制下线：撤销会话，绑定它的令牌随即失效
// DELETE /api/v1/admin/sessions/:id
func (h *AdminSessionHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.BadRequestI18n(c, response.ErrInvalidID)
		return
	}
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	session, err := h.SessionService.Revoke(c.Request.Context(), scope, ucontext.GetUserID(c), id)
	target := id
	if session != nil {
		target = session.Username
	}
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeSessionRevoke, target, fmt.Sprintf("Revoke session %s", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfSessionRevoked)
}
