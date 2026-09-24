package handler

import (
	"fmt"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
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
	logType, verb := audit_log.AuditLogTypeMenuCreate, "Create"
	if update {
		id, err := parseUintParam(c, "id")
		if err != nil {
			response.BadRequestErr(c, err)
			return
		}
		m.ID = id
		logType, verb = audit_log.AuditLogTypeMenuUpdate, "Update"
	}
	err := h.menus.Save(ucontext.WithAuditContext(c), &m)
	logAudit(c, h.audit, logType, m.Code, fmt.Sprintf("%s menu %s (%s, %d permissions)", verb, m.Code, m.Path, len(m.Permissions)), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	err = h.menus.Delete(ucontext.WithAuditContext(c), id)
	logAudit(c, h.audit, audit_log.AuditLogTypeMenuDelete, c.Param("id"), fmt.Sprintf("Delete menu %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
