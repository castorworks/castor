package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AdminAuthorizationHandler 管理员授权处理器
type AdminAuthorizationHandler struct {
	permissionService service.PermissionService
	menus             service.MenuService
	rbac              service.RBACService
	auditLogService   service.AuditLogService
	users             service.UserService
}

// NewAdminAuthorizationHandler 创建管理员授权处理器
func NewAdminAuthorizationHandler(
	permissionService service.PermissionService,
	menus service.MenuService,
	rbac service.RBACService,
	auditLogService service.AuditLogService,
	users service.UserService,
) *AdminAuthorizationHandler {
	return &AdminAuthorizationHandler{
		permissionService: permissionService,
		menus:             menus,
		rbac:              rbac,
		auditLogService:   auditLogService,
		users:             users,
	}
}

// ============================================
// 资源管理 API
// ============================================

// GetResources 获取资源列表
// GET /resources
func (h *AdminAuthorizationHandler) GetResources(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(permission.Resource{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		resources, total, err := h.permissionService.GetResources(ctx.Request.Context(), page, size, order, opts...)
		if err != nil {
			log.Err(ctx, err).Msg("Failed to get resources")
			return nil, 0, err
		}
		return dto.ToResourceRespList(resources), total, nil
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

	err := h.permissionService.CreateResource(c.Request.Context(), resource)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeResourceCreate, req.Code,
		fmt.Sprintf("Create resource %s (%s %v)", req.Name, req.Path, req.Actions), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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

	// 资源的路径/动作决定 RBAC 判定面，改动（包括失败的尝试）必须留痕。
	target := fmt.Sprintf("%d", id)
	resource, err := h.permissionService.GetResource(c.Request.Context(), id)
	if err == nil {
		target = resource.Code
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
		err = h.permissionService.UpdateResource(c.Request.Context(), resource)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeResourceUpdate, target,
		fmt.Sprintf("Update resource %s (%s %v)", req.Name, req.Path, req.Actions), err)
	if err != nil {
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

	target := fmt.Sprintf("%d", id)
	if resource, getErr := h.permissionService.GetResource(c.Request.Context(), id); getErr == nil {
		target = resource.Code
	}
	err = h.permissionService.DeleteResource(c.Request.Context(), id)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeResourceDelete, target, fmt.Sprintf("Delete resource %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	roles, err := h.permissionService.GetRoles(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 获取每个角色的用户数
	result := make([]dto.RoleResp, len(roles))
	for i, role := range roles {
		result[i] = dto.ToRoleResp(&role)
		// 成员数只计调用者数据范围内的用户
		users, _ := h.rbac.GetRoleUsernames(c.Request.Context(), scope, role.Code)
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

	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	role, err := h.permissionService.GetRoleByCode(c.Request.Context(), roleCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	users, err := h.rbac.GetRoleUsernames(c.Request.Context(), scope, roleCode)
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

	err := h.permissionService.CreateRole(c.Request.Context(), role)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeAddRole, req.Code, fmt.Sprintf("Create role %s", req.Name), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	if err == nil {
		if req.IsEnabled != nil {
			role.IsEnabled = *req.IsEnabled
		}
		if req.Name != "" {
			role.Name = req.Name
		}
		if req.Description != nil {
			role.Description = *req.Description
		}
		err = h.permissionService.UpdateRole(c.Request.Context(), role)
	}
	details := "Update role"
	if role != nil {
		details = fmt.Sprintf("Update role %s (enabled=%t)", role.Name, role.IsEnabled)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeUpdateRole, roleCode, details, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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

	role, err := h.permissionService.GetRoleByCode(c.Request.Context(), roleCode)
	if err == nil {
		err = h.permissionService.DeleteRole(c.Request.Context(), role.ID)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeDeleteRole, roleCode, "Delete role "+roleCode, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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

	err := h.rbac.EnsureCanSetRolePermissions(c.Request.Context(), ucontext.GetUsername(c), roleCode, req.Grants)
	if err == nil {
		err = h.rbac.SetRolePermissions(c.Request.Context(), roleCode, req.Grants)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeRolePermissionsSet, roleCode, fmt.Sprintf("Set permissions: %v", req.Grants), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{"updated": true}, response.InfPermissionUpdated)
}

// GetRoleUsers 获取角色下的用户
// GET /roles/:role/users
func (h *AdminAuthorizationHandler) GetRoleUsers(c *gin.Context) {
	roleCode := c.Param("role")
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	users, err := h.rbac.GetRoleUsernames(c.Request.Context(), scope, roleCode)
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
	if !h.ensureUserVisible(c, username) {
		return
	}
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
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	err := h.rbac.EnsureUserVisible(ctx, scope, username)
	if err == nil {
		err = h.rbac.EnsureCanAssignRole(ctx, ucontext.GetUsername(c), username, req.Role)
	}
	if err == nil {
		err = h.rbac.AddUserRole(ctx, username, req.Role, false)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeAddRoleForUser, username, "Add role "+req.Role, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	err := h.rbac.EnsureUserVisible(ctx, scope, username)
	// 撤掉更高权限账号的角色同样是在管理对方账号，与改密码、禁用一致地做越权检查。
	if err == nil {
		target, lookupErr := h.users.GetRawByUsername(ctx, username)
		if err = lookupErr; err == nil {
			err = h.rbac.EnsureCanManageUser(ctx, ucontext.GetUserID(c), target.ID)
		}
	}
	// 删除用户角色分配并撤销授权会话
	if err == nil {
		err = h.rbac.RemoveUserRole(ctx, username, roleCode)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeDeleteRoleForUser, username, "Remove role "+roleCode, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{"deleted": true}, response.InfRoleUpdated)
}

// GetUserPermissions 获取用户的所有权限（包括通过角色继承的）
// GET /authorization/users/:username/permissions
func (h *AdminAuthorizationHandler) GetUserPermissions(c *gin.Context) {
	username := c.Param("username")
	if !h.ensureUserVisible(c, username) {
		return
	}
	access, err := h.rbac.GetUserAccess(c.Request.Context(), username)
	if err != nil {
		log.Err(c, err).Str("username", username).Msg("Failed to get user permissions")
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}

// ensureUserVisible 按用户名操作的授权接口只对数据范围内的用户开放；范围外按不存在处理。
func (h *AdminAuthorizationHandler) ensureUserVisible(c *gin.Context, username string) bool {
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return false
	}
	if err := h.rbac.EnsureUserVisible(c.Request.Context(), scope, username); err != nil {
		response.HandleError(c, err)
		return false
	}
	return true
}

// SetRoleDataScope 设置角色的数据范围
// PUT /roles/:role/data-scope
func (h *AdminAuthorizationHandler) SetRoleDataScope(c *gin.Context) {
	roleCode := c.Param("role")
	var req dto.RoleDataScopeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	ctx := ucontext.WithAuditContext(c)
	err := h.rbac.EnsureCanSetRoleDataScope(ctx, ucontext.GetUsername(c), roleCode, req.DataScope, req.DepartmentIDs)
	var role *permission.Role
	if err == nil {
		role, err = h.permissionService.SetRoleDataScope(ctx, roleCode, req.DataScope, req.DepartmentIDs)
	}
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeUpdateRole, roleCode,
		fmt.Sprintf("Set data scope %s (%d departments)", req.DataScope, len(req.DepartmentIDs)), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, dto.ToRoleResp(role), response.InfDataScopeUpdated)
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
	err := h.rbac.SetRoleJuniors(ucontext.WithAuditContext(c), roleCode, request.JuniorRoleCodes)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeUpdateRole, roleCode, fmt.Sprintf("Set junior roles: %v", request.JuniorRoleCodes), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	err := h.rbac.CreateConstraint(ucontext.WithAuditContext(c), &constraint)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeSoDCreate, constraint.Code, constraintAuditDetails("Create", &constraint), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
	err = h.rbac.UpdateConstraint(ucontext.WithAuditContext(c), id, &constraint)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeSoDUpdate, constraint.Code, constraintAuditDetails("Update", &constraint), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, constraint)
}

// constraintAuditDetails 记下约束的类型、角色集与基数：它们决定哪些角色组合被禁止。
func constraintAuditDetails(verb string, constraint *permission.SeparationConstraint) string {
	return fmt.Sprintf("%s %s constraint %q on role IDs %v (cardinality %d)", verb, constraint.Type, constraint.Name, constraint.RoleIDs, constraint.Cardinality)
}

func (h *AdminAuthorizationHandler) DeleteConstraint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	err = h.rbac.DeleteConstraint(ucontext.WithAuditContext(c), id)
	logAudit(c, h.auditLogService, audit_log.AuditLogTypeSoDDelete, fmt.Sprintf("%d", id), fmt.Sprintf("Delete constraint %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ============================================
// 辅助方法
// ============================================

// getClientIP 返回客户端 IP。使用 gin 的 ClientIP()，它按 TrustedProxies
// 解析 X-Forwarded-For，避免客户端伪造源 IP 污染审计日志。
func getClientIP(c *gin.Context) string {
	return c.ClientIP()
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
