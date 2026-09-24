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

// AdminDepartmentHandler 部门管理处理器
type AdminDepartmentHandler struct {
	departments service.DepartmentService
	rbac        service.RBACService
	audit       service.AuditLogService
}

// NewAdminDepartmentHandler 创建部门管理处理器
func NewAdminDepartmentHandler(departments service.DepartmentService, rbac service.RBACService, audit service.AuditLogService) *AdminDepartmentHandler {
	return &AdminDepartmentHandler{departments: departments, rbac: rbac, audit: audit}
}

// List 返回整棵部门树（扁平列表，按 sortOrder 排序），每个部门标出是否在调用者的数据范围内
// GET /api/v1/admin/departments
func (h *AdminDepartmentHandler) List(c *gin.Context) {
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	items, err := h.departments.List(c.Request.Context(), scope)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// Create 新建部门
// POST /api/v1/admin/departments
func (h *AdminDepartmentHandler) Create(c *gin.Context) { h.save(c, false) }

// Update 修改部门
// PUT /api/v1/admin/departments/:id
func (h *AdminDepartmentHandler) Update(c *gin.Context) { h.save(c, true) }

func (h *AdminDepartmentHandler) save(c *gin.Context, update bool) {
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	var req dto.DepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	d := req.ToEntity()
	logType := audit_log.AuditLogTypeDepartmentCreate
	if update {
		id, err := parseUintParam(c, "id")
		if err != nil {
			response.BadRequestErr(c, err)
			return
		}
		d.ID = id
		logType = audit_log.AuditLogTypeDepartmentUpdate
	}
	err := h.departments.Save(ucontext.WithAuditContext(c), scope, &d)
	logAudit(c, h.audit, logType, d.Code, fmt.Sprintf("Save department %s", d.Code), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, d, response.InfDepartmentSaved)
}

// Delete 删除部门（有下级部门或成员时拒绝）
// DELETE /api/v1/admin/departments/:id
func (h *AdminDepartmentHandler) Delete(c *gin.Context) {
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	err = h.departments.Delete(ucontext.WithAuditContext(c), scope, id)
	logAudit(c, h.audit, audit_log.AuditLogTypeDepartmentDelete, c.Param("id"), fmt.Sprintf("Delete department %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfDepartmentDeleted)
}
