package handler

import (
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	gi18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	menus service.MenuService
	rbac  service.RBACService
	audit service.AuditLogService
}

func NewMenuHandler(menus service.MenuService, rbac service.RBACService, audit service.AuditLogService) *MenuHandler {
	return &MenuHandler{menus: menus, rbac: rbac, audit: audit}
}
func (h *MenuHandler) List(c *gin.Context) {
	data, err := h.menus.Catalog(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, data)
}
func (h *MenuHandler) Create(c *gin.Context) { h.save(c, false) }
func (h *MenuHandler) Update(c *gin.Context) { h.save(c, true) }
func (h *MenuHandler) save(c *gin.Context, update bool) {
	var req dto.MenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	m := req.ToEntity()
	if update {
		id, err := parseUintParam(c, "id")
		if err != nil {
			response.BadRequestErr(c, err)
			return
		}
		m.ID = id
	}
	if err := h.menus.Save(ucontext.WithAuditContext(c), &m); err != nil {
		response.HandleError(c, err)
		return
	}
	h.audit.LogAsync(&audit_log.AuditLog{LogType: audit_log.AuditLogTypeAddPermission, Operator: ucontext.GetUsername(c), OperatorID: ucontext.GetUserID(c), Target: m.Code, Details: gi18n.MustGetMessage(c, response.InfMenuSaved), IpAddr: c.ClientIP(), Success: true})
	var result dto.MenuResp
	result.FromEntity(&m)
	key := response.InfCreateSuccess
	if update {
		key = response.InfUpdateSuccess
	}
	response.SuccessI18n(c, result, key)
}
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	if err = h.menus.Delete(ucontext.WithAuditContext(c), id); err != nil {
		response.HandleError(c, err)
		return
	}
	h.audit.LogAsync(&audit_log.AuditLog{LogType: audit_log.AuditLogTypeDeletePermission, Operator: ucontext.GetUsername(c), OperatorID: ucontext.GetUserID(c), Target: c.Param("id"), Details: gi18n.MustGetMessage(c, response.InfMenuDeleted), IpAddr: c.ClientIP(), Success: true})
	response.SuccessI18n(c, nil, response.InfDeleteSuccess)
}
func (h *MenuHandler) Navigation(c *gin.Context) {
	access, err := h.rbac.GetSessionAccess(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	data, err := h.menus.Navigation(c.Request.Context(), access.Permissions)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, data)
}
