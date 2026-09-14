package dto

import "github.com/castorworks/castor/internal/domain/permission"

// ==========================================
// Authorization Request DTOs
// ==========================================

// RoleCreateReq 创建角色请求
type RoleCreateReq struct {
	Code        string `json:"code" binding:"required,min=2,max=50"`    // 角色编码
	Name        string `json:"name" binding:"required,min=2,max=100"`   // 角色名称
	Description string `json:"description" binding:"omitempty,max=500"` // 角色描述
}

// RoleUpdateReq 更新角色请求
type RoleUpdateReq struct {
	Name        string  `json:"name" binding:"omitempty,min=2,max=100"`  // 角色名称
	Description *string `json:"description" binding:"omitempty,max=500"` // 角色描述
	IsEnabled   *bool   `json:"isEnabled"`                               // 是否启用
}

// RoleForUserReq 用户角色操作请求
type RoleForUserReq struct {
	Role string `json:"role" binding:"required"` // 角色编码
}

// ==========================================
// Resource Request DTOs
// ==========================================

// ResourceCreateReq 创建资源请求
type ResourceCreateReq struct {
	Code        string                      `json:"code" binding:"required,min=2,max=100"`   // 资源编码
	Name        string                      `json:"name" binding:"required,min=2,max=100"`   // 资源名称
	Description string                      `json:"description" binding:"omitempty,max=500"` // 资源描述
	Path        string                      `json:"path" binding:"required,min=1,max=200"`   // 资源路径
	Actions     []string                    `json:"actions" binding:"required,min=1"`        // 支持的操作
	Category    permission.ResourceCategory `json:"category" binding:"required"`             // 资源分类
	Module      string                      `json:"module" binding:"omitempty,max=100"`      // 所属模块
	SortOrder   int                         `json:"sortOrder"`                               // 排序顺序
}

// ResourceUpdateReq 更新资源请求
type ResourceUpdateReq struct {
	Name        string                      `json:"name" binding:"required,min=2,max=100"`   // 资源名称
	Description string                      `json:"description" binding:"omitempty,max=500"` // 资源描述
	Path        string                      `json:"path" binding:"required,min=1,max=200"`   // 资源路径
	Actions     []string                    `json:"actions" binding:"required,min=1"`        // 支持的操作
	Category    permission.ResourceCategory `json:"category" binding:"required"`             // 资源分类
	Module      string                      `json:"module" binding:"omitempty,max=100"`      // 所属模块
	SortOrder   int                         `json:"sortOrder"`                               // 排序顺序
	IsEnabled   *bool                       `json:"isEnabled"`                               // 是否启用
}

// ResourceQueryReq 资源查询请求
type ResourceQueryReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=500"`
	Category  string `form:"category" binding:"omitempty"`
	Module    string `form:"module" binding:"omitempty"`
	IsEnabled *bool  `form:"isEnabled"`                  // 可选：过滤启用/禁用状态
	Search    string `form:"search" binding:"omitempty"` // 关键词搜索（name/code/path）
	Sort      string `form:"sort" binding:"omitempty"`   // JSON 排序规则，如 [{"id":"name","desc":false}]
}

// ==========================================
// Role Permission Request DTOs
// ==========================================

// RolePermissionSetReq 设置角色权限请求（完全替换）
type RolePermissionSetReq struct {
	// An empty list is a valid full replacement that revokes every non-system grant.
	Grants []permission.PermissionGrant `json:"grants"`
}

// ==========================================
// Authorization Response DTOs
// ==========================================

// RoleResp 角色响应
type RoleResp struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsSystem    bool   `json:"isSystem"`
	IsEnabled   bool   `json:"isEnabled"`
	UserCount   int    `json:"userCount,omitempty"`
}

// RoleDetailResp 角色详情响应
type RoleDetailResp struct {
	RoleResp
	Users       []string             `json:"users,omitempty"`
	Permissions []RolePermissionResp `json:"permissions,omitempty"`
}

// PermissionResp 权限响应
type PermissionResp struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// ==========================================
// Resource Response DTOs
// ==========================================

// ResourceResp 资源响应
type ResourceResp struct {
	ID          uint     `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path"`
	Actions     []string `json:"actions"`
	Category    string   `json:"category"`
	Module      string   `json:"module,omitempty"`
	SortOrder   int      `json:"sortOrder"`
	IsSystem    bool     `json:"isSystem"`
	IsEnabled   bool     `json:"isEnabled"`
}

// ResourceListResp 资源列表响应
type ResourceListResp struct {
	Items    []ResourceResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// ModuleResp 模块响应
type ModuleResp struct {
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Resources []ResourceResp `json:"resources"`
}

// ==========================================
// Role Permission Response DTOs
// ==========================================

// RolePermissionResp 角色权限响应
type RolePermissionResp struct {
	ResourceID   uint     `json:"resourceId"`
	ResourceCode string   `json:"resourceCode"`
	ResourceName string   `json:"resourceName"`
	ResourcePath string   `json:"resourcePath"`
	Actions      []string `json:"actions"`
	IsSystem     bool     `json:"isSystem"`
}

// ==========================================
// 转换函数
// ==========================================

// ToResourceResp 转换资源实体到响应
func ToResourceResp(r *permission.Resource) ResourceResp {
	return ResourceResp{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		Path:        r.Path,
		Actions:     r.Actions,
		Category:    string(r.Category),
		Module:      r.Module,
		SortOrder:   r.SortOrder,
		IsSystem:    r.IsSystem,
		IsEnabled:   r.IsEnabled,
	}
}

// ToResourceRespList 转换资源实体列表到响应列表
func ToResourceRespList(resources []permission.Resource) []ResourceResp {
	result := make([]ResourceResp, len(resources))
	for i, r := range resources {
		result[i] = ToResourceResp(&r)
	}
	return result
}

// ToRoleResp 转换角色实体到响应
func ToRoleResp(r *permission.Role) RoleResp {
	return RoleResp{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		IsEnabled:   r.IsEnabled,
	}
}

// ToRoleRespList 转换角色实体列表到响应列表
func ToRoleRespList(roles []permission.Role) []RoleResp {
	result := make([]RoleResp, len(roles))
	for i, r := range roles {
		result[i] = ToRoleResp(&r)
	}
	return result
}
