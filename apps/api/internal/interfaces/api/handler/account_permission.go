package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

type AccountPermissionHandler struct {
	rbac  service.RBACService
	audit service.AuditLogService
}

func NewAccountPermissionHandler(rbac service.RBACService, auditLogService service.AuditLogService) *AccountPermissionHandler {
	return &AccountPermissionHandler{rbac: rbac, audit: auditLogService}
}

func (h *AccountPermissionHandler) GetMyRoles(c *gin.Context) {
	h.getAccess(c)
}

func (h *AccountPermissionHandler) GetMyPermissions(c *gin.Context) {
	h.getAccess(c)
}

func (h *AccountPermissionHandler) PutActiveRoles(c *gin.Context) {
	var request struct {
		RoleCodes []string `json:"roleCodes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	access, err := h.rbac.SetActiveRoles(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c), request.RoleCodes)
	// 激活角色改变的是调用者自己的有效权限，等同于一次提权/降权。
	logAudit(c, h.audit, audit_log.AuditLogTypeActiveRolesChange, ucontext.GetUsername(c),
		fmt.Sprintf("Set active roles: %s", joinAuditKeys(request.RoleCodes)), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}

func (h *AccountPermissionHandler) getAccess(c *gin.Context) {
	access, err := h.rbac.GetSessionAccess(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}
