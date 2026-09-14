package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AdminAuthorizationHandler 管理员授权处理器
type AdminAuthorizationHandler struct {
	permissionService service.PermissionService
	menus             service.MenuService
	rbac              service.RBACService
	auditLogService   service.AuditLogService
}

// NewAdminAuthorizationHandler 创建管理员授权处理器
func NewAdminAuthorizationHandler(
	permissionService service.PermissionService,
	menus service.MenuService,
	rbac service.RBACService,
	auditLogService service.AuditLogService,
) *AdminAuthorizationHandler {
	return &AdminAuthorizationHandler{
		permissionService: permissionService,
		menus:             menus,
		rbac:              rbac,
		auditLogService:   auditLogService,
	}
}

// ============================================
// 资源管理 API
// ============================================

// GetResources 获取资源列表
// GET /resources
func (h *AdminAuthorizationHandler) GetResources(c *gin.Context) {
	var req dto.ResourceQueryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	resources, total, err := h.permissionService.GetResources(c.Request.Context(), req.Page, req.PageSize, req.Category, req.Module, req.IsEnabled, req.Search)
	if err != nil {
		log.Err(c, err).Msg("Failed to get resources")
		response.HandleError(c, err)
		return
	}

	response.Success(c, dto.ResourceListResp{
		Items:    dto.ToResourceRespList(resources),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// GetResource 获取单个资源
// GET /resources/:id
func (h *AdminAuthorizationHandler) GetResource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	resource, err := h.permissionService.GetResource(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, dto.ToResourceResp(resource))
}

// CreateResource 创建资源
// POST /resources
func (h *AdminAuthorizationHandler) CreateResource(c *gin.Context) {
	var req dto.ResourceCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	resource := &permission.Resource{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Path:        req.Path,
		Actions:     req.Actions,
		Category:    req.Category,
		Module:      req.Module,
		SortOrder:   req.SortOrder,
		IsSystem:    false,
		IsEnabled:   true,
	}

	if err := h.permissionService.CreateResource(c.Request.Context(), resource); err != nil {
		h.logAudit(c, audit_log.AuditLogTypeAddPermission, req.Code,
			fmt.Sprintf("Failed to create resource: %s", err.Error()), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeAddPermission, req.Code,
		fmt.Sprintf("Created resource: %s (%s)", req.Name, req.Path), true)
	response.Success(c, dto.ToResourceResp(resource))
}

// UpdateResource 更新资源
// PUT /resources/:id
func (h *AdminAuthorizationHandler) UpdateResource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	var req dto.ResourceUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	// 获取现有资源
	resource, err := h.permissionService.GetResource(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 更新字段
	resource.Name = req.Name
	resource.Description = req.Description
	resource.Path = req.Path
	resource.Actions = req.Actions
	resource.Category = req.Category
	resource.Module = req.Module
	resource.SortOrder = req.SortOrder
	if req.IsEnabled != nil {
		resource.IsEnabled = *req.IsEnabled
	}

	if err := h.permissionService.UpdateResource(c.Request.Context(), resource); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, dto.ToResourceResp(resource), response.InfUpdateSuccess)
}

// DeleteResource 删除资源
// DELETE /resources/:id
func (h *AdminAuthorizationHandler) DeleteResource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.permissionService.DeleteResource(c.Request.Context(), id); err != nil {
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeDeletePermission, fmt.Sprintf("%d", id),
		"Deleted resource", true)
	response.SuccessI18n(c, gin.H{"deleted": true}, response.InfDeleteSuccess)
}

// GetResourceModules 获取所有模块
// GET /resources/modules
func (h *AdminAuthorizationHandler) GetResourceModules(c *gin.Context) {
	modules, err := h.permissionService.GetAllModules(c.Request.Context())
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}

	response.Success(c, modules)
}

// ============================================
// 角色管理 API
// ============================================

// GetRoles 获取所有角色
// GET /roles
func (h *AdminAuthorizationHandler) GetRoles(c *gin.Context) {
	roles, err := h.permissionService.GetRoles(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 获取每个角色的用户数
	result := make([]dto.RoleResp, len(roles))
	for i, role := range roles {
		result[i] = dto.ToRoleResp(&role)
		users, _ := h.rbac.GetRoleUsernames(c.Request.Context(), role.Code)
		result[i].UserCount = len(users)
	}

	response.Success(c, result)
}

// GetRole 获取角色详情
// GET /roles/:role
func (h *AdminAuthorizationHandler) GetRole(c *gin.Context) {
	roleCode := c.Param("role")
	if roleCode == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	role, err := h.permissionService.GetRoleByCode(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	users, err := h.rbac.GetRoleUsernames(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if users == nil {
		users = []string{}
	}

	permissions, err := h.rbac.GetRolePermissions(c.Request.Context(), roleCode)
	if err != nil {
		log.Err(c, err).Str("role", roleCode).Msg("Failed to get role permissions")
		response.HandleError(c, err)
		return
	}

	permissionResp := make([]dto.RolePermissionResp, len(permissions))
	for i, p := range permissions {
		permissionResp[i] = dto.RolePermissionResp{
			ResourceID:   p.ResourceID,
			ResourceCode: p.ResourceCode,
			ResourceName: p.ResourceName,
			ResourcePath: p.ResourcePath,
			Actions:      p.Actions,
			IsSystem:     p.IsSystem,
		}
	}

	roleResp := dto.ToRoleResp(role)
	roleResp.UserCount = len(users)
	response.Success(c, dto.RoleDetailResp{
		RoleResp:    roleResp,
		Users:       users,
		Permissions: permissionResp,
	})
}

// CreateRole 创建角色
// POST /roles
func (h *AdminAuthorizationHandler) CreateRole(c *gin.Context) {
	var req dto.RoleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	role := &permission.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
		IsEnabled:   true,
	}

	if err := h.permissionService.CreateRole(c.Request.Context(), role); err != nil {
		h.logAudit(c, audit_log.AuditLogTypeAddRole, req.Code,
			fmt.Sprintf("Failed to create role: %s", err.Error()), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeAddRole, req.Code,
		fmt.Sprintf("Created role: %s", req.Name), true)
	response.SuccessI18n(c, dto.ToRoleResp(role), response.InfRoleCreated)
}

// UpdateRole 更新角色
// PUT /roles/:role
func (h *AdminAuthorizationHandler) UpdateRole(c *gin.Context) {
	roleCode := c.Param("role")
	if roleCode == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	var req dto.RoleUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	role, err := h.permissionService.GetRoleByCode(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if req.IsEnabled != nil {
		role.IsEnabled = *req.IsEnabled
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := h.permissionService.UpdateRole(c.Request.Context(), role); err != nil {
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeUpdateRole, roleCode,
		fmt.Sprintf("Updated role: %s", role.Name), true)
	response.SuccessI18n(c, dto.ToRoleResp(role), response.InfRoleUpdated)
}

// DeleteRole 删除角色
// DELETE /roles/:role
func (h *AdminAuthorizationHandler) DeleteRole(c *gin.Context) {
	roleCode := c.Param("role")
	if roleCode == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	// 获取角色
	role, err := h.permissionService.GetRoleByCode(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if err := h.permissionService.DeleteRole(c.Request.Context(), role.ID); err != nil {
		h.logAudit(c, audit_log.AuditLogTypeDeleteRole, roleCode,
			fmt.Sprintf("Failed to delete role: %s", err.Error()), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeDeleteRole, roleCode,
		fmt.Sprintf("Deleted role: %s", roleCode), true)
	response.SuccessI18n(c, gin.H{"deleted": true, "role": roleCode}, response.InfRoleDeleted)
}

// GetRolePermissions 获取角色的权限
// GET /roles/:role/permissions
func (h *AdminAuthorizationHandler) GetRolePermissions(c *gin.Context) {
	roleCode := c.Param("role")
	if roleCode == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	permissions, err := h.rbac.GetRolePermissions(c.Request.Context(), roleCode)
	if err != nil {
		log.Err(c, err).Str("role", roleCode).Msg("Failed to get role permissions")
		response.HandleError(c, err)
		return
	}

	catalog, err := h.menus.Catalog(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"grants": permissions, "menus": catalog.Menus, "resources": catalog.Resources})
}

// SetRolePermissions 设置角色权限（完全替换）
// PUT /roles/:role/permissions
func (h *AdminAuthorizationHandler) SetRolePermissions(c *gin.Context) {
	roleCode := c.Param("role")
	var req dto.RolePermissionSetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.rbac.SetRolePermissions(c.Request.Context(), roleCode, req.Grants); err != nil {
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeAddPermission, roleCode,
		fmt.Sprintf("Set permissions: %v", req.Grants), true)
	response.SuccessI18n(c, gin.H{"updated": true}, response.InfPermissionUpdated)
}

// GetRoleUsers 获取角色下的用户
// GET /roles/:role/users
func (h *AdminAuthorizationHandler) GetRoleUsers(c *gin.Context) {
	roleCode := c.Param("role")
	users, err := h.rbac.GetRoleUsernames(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, users)
}

// ============================================
// 用户角色管理 API
// ============================================

// GetUserRoles 获取用户的角色
// GET /authorization/users/:username/roles
func (h *AdminAuthorizationHandler) GetUserRoles(c *gin.Context) {
	username := c.Param("username")
	roles, err := h.rbac.GetUserRoleCodes(c.Request.Context(), username)
	if err != nil {
		log.Err(c, err).Str("username", username).Msg("Failed to get user roles")
		response.HandleError(c, err)
		return
	}
	response.Success(c, roles)
}

// AddUserRole 为用户添加角色
// POST /authorization/users/:username/roles
func (h *AdminAuthorizationHandler) AddUserRole(c *gin.Context) {
	username := c.Param("username")
	var req dto.RoleForUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	ctx := c.Request.Context()

	if err := h.rbac.AddUserRole(ctx, username, req.Role, false); err != nil {
		log.Err(c, err).Str("username", username).Str("role", req.Role).Msg("Failed to add user role")
		h.logAudit(c, audit_log.AuditLogTypeAddRoleForUser, username,
			fmt.Sprintf("Failed to add role %s: %s", req.Role, err.Error()), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeAddRoleForUser, username,
		fmt.Sprintf("Added role %s", req.Role), true)
	response.SuccessI18n(c, gin.H{"added": true}, response.InfRoleUpdated)
}

// DeleteUserRole 删除用户的角色
// DELETE /authorization/users/:username/roles/:role
func (h *AdminAuthorizationHandler) DeleteUserRole(c *gin.Context) {
	username := c.Param("username")
	roleCode := c.Param("role")

	if roleCode == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	ctx := c.Request.Context()

	// 删除用户角色分配并撤销授权会话
	if err := h.rbac.RemoveUserRole(ctx, username, roleCode); err != nil {
		log.Err(c, err).Str("username", username).Str("role", roleCode).Msg("Failed to delete user role")
		h.logAudit(c, audit_log.AuditLogTypeDeleteRoleForUser, username,
			fmt.Sprintf("Failed to delete role %s: %s", roleCode, err.Error()), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypeDeleteRoleForUser, username,
		fmt.Sprintf("Deleted role %s", roleCode), true)
	response.SuccessI18n(c, gin.H{"deleted": true}, response.InfRoleUpdated)
}

// GetUserPermissions 获取用户的所有权限（包括通过角色继承的）
// GET /authorization/users/:username/permissions
func (h *AdminAuthorizationHandler) GetUserPermissions(c *gin.Context) {
	username := c.Param("username")
	access, err := h.rbac.GetUserAccess(c.Request.Context(), username)
	if err != nil {
		log.Err(c, err).Str("username", username).Msg("Failed to get user permissions")
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}

// ============================================
// 元数据查询 API
// ============================================

// GetSubjects 获取所有可授权角色
// GET /authorization/metadata/subjects
func (h *AdminAuthorizationHandler) GetSubjects(c *gin.Context) {
	roles, err := h.permissionService.GetRoles(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	subjects := make([]string, len(roles))
	for i := range roles {
		subjects[i] = roles[i].Code
	}
	response.Success(c, subjects)
}

// GetObjects 获取所有资源对象
// GET /authorization/metadata/objects
func (h *AdminAuthorizationHandler) GetObjects(c *gin.Context) {
	adminResources, err := h.permissionService.GetResourcesByCategory(c.Request.Context(), permission.CategoryAdmin)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	userResources, err := h.permissionService.GetResourcesByCategory(c.Request.Context(), permission.CategoryUser)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	resources := append(adminResources, userResources...)
	objects := make([]string, len(resources))
	for i := range resources {
		objects[i] = resources[i].Path
	}
	response.Success(c, objects)
}

// GetActions 获取所有操作
// GET /authorization/metadata/actions
func (h *AdminAuthorizationHandler) GetActions(c *gin.Context) {
	response.Success(c, permission.AllActions)
}

// GetPolicies 获取所有策略
// GET /authorization/metadata/policies
func (h *AdminAuthorizationHandler) GetPolicies(c *gin.Context) {
	roles, err := h.permissionService.GetRoles(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	result := make([]dto.PermissionResp, 0)
	for _, role := range roles {
		grants, err := h.rbac.GetRolePermissions(c.Request.Context(), role.Code)
		if err != nil {
			response.HandleError(c, err)
			return
		}
		for _, grant := range grants {
			for _, action := range grant.Actions {
				result = append(result, dto.PermissionResp{Role: role.Code, Resource: grant.ResourcePath, Action: action})
			}
		}
	}
	response.Success(c, result)
}

func (h *AdminAuthorizationHandler) GetRoleHierarchy(c *gin.Context) {
	roles, err := h.rbac.GetRoleJuniors(c.Request.Context(), c.Param("role"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, roles)
}

func (h *AdminAuthorizationHandler) SetRoleHierarchy(c *gin.Context) {
	var request struct {
		JuniorRoleCodes []string `json:"juniorRoleCodes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	roleCode := c.Param("role")
	if err := h.rbac.SetRoleJuniors(ucontext.WithAuditContext(c), roleCode, request.JuniorRoleCodes); err != nil {
		response.HandleError(c, err)
		return
	}
	h.logAudit(c, audit_log.AuditLogTypeUpdateRole, roleCode, fmt.Sprintf("Set junior roles: %v", request.JuniorRoleCodes), true)
	response.SuccessI18n(c, gin.H{"updated": true}, response.InfRoleUpdated)
}

func (h *AdminAuthorizationHandler) GetConstraints(c *gin.Context) {
	constraints, err := h.rbac.GetConstraints(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, constraints)
}

func (h *AdminAuthorizationHandler) CreateConstraint(c *gin.Context) {
	var constraint permission.SeparationConstraint
	if err := c.ShouldBindJSON(&constraint); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	if err := h.rbac.CreateConstraint(ucontext.WithAuditContext(c), &constraint); err != nil {
		response.HandleError(c, err)
		return
	}
	h.logAudit(c, audit_log.AuditLogTypeAddPermission, constraint.Code, "Created separation of duty constraint", true)
	response.Success(c, constraint)
}

func (h *AdminAuthorizationHandler) UpdateConstraint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	var constraint permission.SeparationConstraint
	if err := c.ShouldBindJSON(&constraint); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	if err := h.rbac.UpdateConstraint(ucontext.WithAuditContext(c), id, &constraint); err != nil {
		response.HandleError(c, err)
		return
	}
	h.logAudit(c, audit_log.AuditLogTypeAddPermission, constraint.Code, "Updated separation of duty constraint", true)
	response.Success(c, constraint)
}

func (h *AdminAuthorizationHandler) DeleteConstraint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	if err := h.rbac.DeleteConstraint(ucontext.WithAuditContext(c), id); err != nil {
		response.HandleError(c, err)
		return
	}
	h.logAudit(c, audit_log.AuditLogTypeDeletePermission, fmt.Sprintf("%d", id), "Deleted separation of duty constraint", true)
	response.Success(c, gin.H{"deleted": true})
}

// ============================================
// 辅助方法
// ============================================

// getClientIP 获取客户端 IP
func getClientIP(c *gin.Context) string {
	ip := c.Request.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}

// logAudit 记录审计日志（异步）
func (h *AdminAuthorizationHandler) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	h.auditLogService.LogAsync(&audit_log.AuditLog{
		LogType:    logType,
		Operator:   ucontext.GetUsername(c),
		OperatorID: ucontext.GetUserID(c),
		Target:     target,
		Details:    details,
		IpAddr:     getClientIP(c),
		Success:    success,
	})
}

// parseUintParam 解析 uint 类型的路径参数
func parseUintParam(c *gin.Context, name string) (uint, error) {
	param := c.Param(name)
	if param == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	var id uint
	if _, err := fmt.Sscanf(param, "%d", &id); err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}

	return id, nil
}
