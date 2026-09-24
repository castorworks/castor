package permission

import (
	"time"
)

// ============================================
// 权限系统常量定义
// ============================================

// ResourceCategory 资源分类
type ResourceCategory string

const (
	CategoryAdmin ResourceCategory = "admin" // 管理端资源
	CategoryUser  ResourceCategory = "user"  // 用户端资源
)

// HTTP 方法常量
const (
	ActionGET    = "GET"
	ActionPOST   = "POST"
	ActionPUT    = "PUT"
	ActionPATCH  = "PATCH"
	ActionDELETE = "DELETE"
)

// AllActions 所有支持的 HTTP 方法
var AllActions = []string{ActionGET, ActionPOST, ActionPUT, ActionPATCH, ActionDELETE}

// 角色常量
const (
	RoleAdmin   = "admin"
	RoleAuditor = "auditor"
	RoleUser    = "user"
)

// 默认账户常量
const (
	AccountSystem = "system"
	AccountAudit  = "auditor"
)

// Resource 资源定义实体（纯 Domain 模型，无 ORM 依赖）
type Resource struct {
	ID          uint             `json:"id"`
	CreatedAt   time.Time        `json:"createdAt"`
	CreatedBy   uint             `json:"createdBy"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	UpdatedBy   uint             `json:"updatedBy"`
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Path        string           `json:"path"`
	Actions     StringSlice      `json:"actions"`
	Category    ResourceCategory `json:"category"`
	Module      string           `json:"module"`
	SortOrder   int              `json:"sortOrder"`
	IsSystem    bool             `json:"isSystem"`
	IsEnabled   bool             `json:"isEnabled"`
}

// Role 角色实体（纯 Domain 模型，无 ORM 依赖）
type Role struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   uint      `json:"createdBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   uint      `json:"updatedBy"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"isSystem"`
	IsEnabled   bool      `json:"isEnabled"`
	// DataScope 决定成员能看到哪些用户的数据；CUSTOM 时范围是 DepartmentIDs。
	DataScope     DataScope `json:"dataScope"`
	DepartmentIDs []uint    `json:"departmentIds"`
}

// ==========================================
// 默认资源配置（保持与 domain 层一致）
// ==========================================

var DefaultResources = []Resource{
	{Code: "admin:menus:list", Name: "seedResources.admin.menus.list", Path: "/api/v1/admin/menus", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "menus", SortOrder: 650, IsSystem: true, IsEnabled: true},
	{Code: "admin:menus:create", Name: "seedResources.admin.menus.create", Path: "/api/v1/admin/menus", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "menus", SortOrder: 651, IsSystem: true, IsEnabled: true},
	{Code: "admin:menus:update", Name: "seedResources.admin.menus.update", Path: "/api/v1/admin/menus/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "menus", SortOrder: 652, IsSystem: true, IsEnabled: true},
	{Code: "admin:menus:delete", Name: "seedResources.admin.menus.delete", Path: "/api/v1/admin/menus/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "menus", SortOrder: 653, IsSystem: true, IsEnabled: true},
	// ===== Dashboard =====
	{Code: "admin:dashboard:stats", Name: "seedResources.admin.dashboard.stats", Path: "/api/v1/admin/dashboard/stats", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dashboard", SortOrder: 200, IsSystem: true, IsEnabled: true},

	// ===== Assets =====
	{Code: "admin:assets:download", Name: "seedResources.admin.assets.download", Path: "/api/v1/admin/assets/download/:objectKey", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "assets", SortOrder: 209, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:list", Name: "seedResources.admin.assets.list", Path: "/api/v1/admin/assets", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "assets", SortOrder: 210, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:stats", Name: "seedResources.admin.assets.stats", Path: "/api/v1/admin/assets/stats", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "assets", SortOrder: 211, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:detail", Name: "seedResources.admin.assets.detail", Path: "/api/v1/admin/assets/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "assets", SortOrder: 215, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:create", Name: "seedResources.admin.assets.create", Path: "/api/v1/admin/assets", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "assets", SortOrder: 216, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:update", Name: "seedResources.admin.assets.update", Path: "/api/v1/admin/assets/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "assets", SortOrder: 217, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:update-status", Name: "seedResources.admin.assets.updateStatus", Path: "/api/v1/admin/assets/:id/status", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "assets", SortOrder: 218, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:move", Name: "seedResources.admin.assets.move", Path: "/api/v1/admin/assets/:id/move", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "assets", SortOrder: 219, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:delete", Name: "seedResources.admin.assets.delete", Path: "/api/v1/admin/assets/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "assets", SortOrder: 220, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:batch-delete", Name: "seedResources.admin.assets.batchDelete", Path: "/api/v1/admin/assets/batch/delete", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "assets", SortOrder: 221, IsSystem: true, IsEnabled: true},
	{Code: "admin:assets:batch-update-status", Name: "seedResources.admin.assets.batchUpdateStatus", Path: "/api/v1/admin/assets/batch/status", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "assets", SortOrder: 222, IsSystem: true, IsEnabled: true},

	// ===== Login Histories =====
	{Code: "admin:login-histories:list", Name: "seedResources.admin.loginHistories.list", Path: "/api/v1/admin/login-histories", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "login-histories", SortOrder: 230, IsSystem: true, IsEnabled: true},
	{Code: "admin:login-histories:delete", Name: "seedResources.admin.loginHistories.delete", Path: "/api/v1/admin/login-histories/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "login-histories", SortOrder: 231, IsSystem: true, IsEnabled: true},
	{Code: "admin:login-histories:batch-delete", Name: "seedResources.admin.loginHistories.batchDelete", Path: "/api/v1/admin/login-histories/batch/delete", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "login-histories", SortOrder: 232, IsSystem: true, IsEnabled: true},
	{Code: "admin:login-histories:export", Name: "seedResources.admin.loginHistories.export", Path: "/api/v1/admin/login-histories/export", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "login-histories", SortOrder: 233, IsSystem: true, IsEnabled: true},
	{Code: "admin:sessions:list", Name: "seedResources.admin.sessions.list", Path: "/api/v1/admin/sessions", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "sessions", SortOrder: 235, IsSystem: true, IsEnabled: true},
	{Code: "admin:sessions:revoke", Name: "seedResources.admin.sessions.revoke", Path: "/api/v1/admin/sessions/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "sessions", SortOrder: 236, IsSystem: true, IsEnabled: true},
	{Code: "admin:jobs:list", Name: "seedResources.admin.jobs.list", Path: "/api/v1/admin/jobs", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "jobs", SortOrder: 240, IsSystem: true, IsEnabled: true},
	{Code: "admin:jobs:update", Name: "seedResources.admin.jobs.update", Path: "/api/v1/admin/jobs/:key", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "jobs", SortOrder: 241, IsSystem: true, IsEnabled: true},
	{Code: "admin:jobs:run", Name: "seedResources.admin.jobs.run", Path: "/api/v1/admin/jobs/:key/run", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "jobs", SortOrder: 242, IsSystem: true, IsEnabled: true},
	{Code: "admin:oidc-providers:list", Name: "seedResources.admin.oidcProviders.list", Path: "/api/v1/admin/oidc-providers", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "oidc-providers", SortOrder: 260, IsSystem: true, IsEnabled: true},
	{Code: "admin:oidc-providers:create", Name: "seedResources.admin.oidcProviders.create", Path: "/api/v1/admin/oidc-providers", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "oidc-providers", SortOrder: 261, IsSystem: true, IsEnabled: true},
	{Code: "admin:oidc-providers:update", Name: "seedResources.admin.oidcProviders.update", Path: "/api/v1/admin/oidc-providers/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "oidc-providers", SortOrder: 262, IsSystem: true, IsEnabled: true},
	{Code: "admin:oidc-providers:delete", Name: "seedResources.admin.oidcProviders.delete", Path: "/api/v1/admin/oidc-providers/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "oidc-providers", SortOrder: 263, IsSystem: true, IsEnabled: true},
	{Code: "admin:openapi:get", Name: "seedResources.admin.openapi.get", Path: "/api/v1/admin/openapi.json", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "openapi", SortOrder: 250, IsSystem: true, IsEnabled: true},
	{Code: "admin:job-runs:list", Name: "seedResources.admin.jobRuns.list", Path: "/api/v1/admin/job-runs", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "jobs", SortOrder: 243, IsSystem: true, IsEnabled: true},

	// ===== Audit Logs =====
	{Code: "admin:audit-logs:list", Name: "seedResources.admin.auditLogs.list", Path: "/api/v1/admin/audit-logs", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "audit-logs", SortOrder: 240, IsSystem: true, IsEnabled: true},
	{Code: "admin:audit-logs:cleanup", Name: "seedResources.admin.auditLogs.cleanup", Path: "/api/v1/admin/audit-logs/cleanup", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "audit-logs", SortOrder: 241, IsSystem: true, IsEnabled: true},
	{Code: "admin:audit-logs:export", Name: "seedResources.admin.auditLogs.export", Path: "/api/v1/admin/audit-logs/export", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "audit-logs", SortOrder: 242, IsSystem: true, IsEnabled: true},

	// ===== Settings =====
	{Code: "admin:settings:list", Name: "seedResources.admin.settings.list", Path: "/api/v1/admin/settings", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "settings", SortOrder: 250, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:detail", Name: "seedResources.admin.settings.detail", Path: "/api/v1/admin/settings/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "settings", SortOrder: 251, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:get-by-key", Name: "seedResources.admin.settings.getByKey", Path: "/api/v1/admin/settings/key/:key", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "settings", SortOrder: 252, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:create", Name: "seedResources.admin.settings.create", Path: "/api/v1/admin/settings", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "settings", SortOrder: 253, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:update", Name: "seedResources.admin.settings.update", Path: "/api/v1/admin/settings/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "settings", SortOrder: 254, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:batch-update", Name: "seedResources.admin.settings.batchUpdate", Path: "/api/v1/admin/settings/batch", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "settings", SortOrder: 255, IsSystem: true, IsEnabled: true},
	{Code: "admin:settings:delete", Name: "seedResources.admin.settings.delete", Path: "/api/v1/admin/settings/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "settings", SortOrder: 256, IsSystem: true, IsEnabled: true},

	// ===== Dict Types =====
	{Code: "admin:dict-types:list", Name: "seedResources.admin.dictTypes.list", Path: "/api/v1/admin/dict-types", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dict-types", SortOrder: 260, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-types:detail", Name: "seedResources.admin.dictTypes.detail", Path: "/api/v1/admin/dict-types/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dict-types", SortOrder: 261, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-types:create", Name: "seedResources.admin.dictTypes.create", Path: "/api/v1/admin/dict-types", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "dict-types", SortOrder: 262, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-types:update", Name: "seedResources.admin.dictTypes.update", Path: "/api/v1/admin/dict-types/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "dict-types", SortOrder: 263, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-types:delete", Name: "seedResources.admin.dictTypes.delete", Path: "/api/v1/admin/dict-types/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "dict-types", SortOrder: 264, IsSystem: true, IsEnabled: true},

	// ===== Dict Items =====
	{Code: "admin:dict-items:list", Name: "seedResources.admin.dictItems.list", Path: "/api/v1/admin/dict-items", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 270, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-items:list-by-type", Name: "seedResources.admin.dictItems.listByType", Path: "/api/v1/admin/dict-items/type/:typeCode", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 271, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-items:detail", Name: "seedResources.admin.dictItems.detail", Path: "/api/v1/admin/dict-items/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 272, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-items:create", Name: "seedResources.admin.dictItems.create", Path: "/api/v1/admin/dict-items", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 273, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-items:update", Name: "seedResources.admin.dictItems.update", Path: "/api/v1/admin/dict-items/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 274, IsSystem: true, IsEnabled: true},
	{Code: "admin:dict-items:delete", Name: "seedResources.admin.dictItems.delete", Path: "/api/v1/admin/dict-items/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "dict-items", SortOrder: 275, IsSystem: true, IsEnabled: true},

	// ===== Notifications =====
	{Code: "admin:notifications:list", Name: "seedResources.admin.notifications.list", Path: "/api/v1/admin/notifications", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "notifications", SortOrder: 280, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:channels", Name: "seedResources.admin.notifications.channels", Path: "/api/v1/admin/notifications/channels", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "notifications", SortOrder: 305, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:detail", Name: "seedResources.admin.notifications.detail", Path: "/api/v1/admin/notifications/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "notifications", SortOrder: 281, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:create", Name: "seedResources.admin.notifications.create", Path: "/api/v1/admin/notifications", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "notifications", SortOrder: 282, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:update", Name: "seedResources.admin.notifications.update", Path: "/api/v1/admin/notifications/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "notifications", SortOrder: 283, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:delete", Name: "seedResources.admin.notifications.delete", Path: "/api/v1/admin/notifications/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "notifications", SortOrder: 284, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:batch-delete", Name: "seedResources.admin.notifications.batchDelete", Path: "/api/v1/admin/notifications/batch/delete", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "notifications", SortOrder: 285, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:recipients", Name: "seedResources.admin.notifications.recipients", Path: "/api/v1/admin/notifications/:id/recipients", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "notifications", SortOrder: 286, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:upload-attachment", Name: "seedResources.admin.notifications.uploadAttachment", Path: "/api/v1/admin/notifications/attachments", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "notifications", SortOrder: 287, IsSystem: true, IsEnabled: true},
	{Code: "admin:notifications:download-attachment", Name: "seedResources.admin.notifications.downloadAttachment", Path: "/api/v1/admin/notifications/:id/attachments/:objectKey", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "notifications", SortOrder: 288, IsSystem: true, IsEnabled: true},

	// ===== Examples =====

	// ===== Users =====
	{Code: "admin:users:list", Name: "seedResources.admin.users.list", Path: "/api/v1/admin/users", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "users", SortOrder: 100, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:detail", Name: "seedResources.admin.users.detail", Path: "/api/v1/admin/users/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "users", SortOrder: 101, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:create", Name: "seedResources.admin.users.create", Path: "/api/v1/admin/users", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "users", SortOrder: 102, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:update", Name: "seedResources.admin.users.update", Path: "/api/v1/admin/users/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "users", SortOrder: 103, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:delete", Name: "seedResources.admin.users.delete", Path: "/api/v1/admin/users/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "users", SortOrder: 104, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:reset-totp", Name: "seedResources.admin.users.resetTotp", Path: "/api/v1/admin/users/:id/totp", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "users", SortOrder: 108, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:export", Name: "seedResources.admin.users.export", Path: "/api/v1/admin/users/export", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "users", SortOrder: 105, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:import-template", Name: "seedResources.admin.users.importTemplate", Path: "/api/v1/admin/users/import-template", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "users", SortOrder: 106, IsSystem: true, IsEnabled: true},
	{Code: "admin:users:import", Name: "seedResources.admin.users.import", Path: "/api/v1/admin/users/import", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "users", SortOrder: 107, IsSystem: true, IsEnabled: true},

	// ===== Permission Resources =====
	{Code: "admin:resources:list", Name: "seedResources.admin.resources.list", Path: "/api/v1/admin/resources", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "resources", SortOrder: 500, IsSystem: true, IsEnabled: true},
	{Code: "admin:resources:modules", Name: "seedResources.admin.resources.modules", Path: "/api/v1/admin/resources/modules", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "resources", SortOrder: 501, IsSystem: true, IsEnabled: true},
	{Code: "admin:resources:detail", Name: "seedResources.admin.resources.detail", Path: "/api/v1/admin/resources/:id", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "resources", SortOrder: 502, IsSystem: true, IsEnabled: true},
	{Code: "admin:resources:create", Name: "seedResources.admin.resources.create", Path: "/api/v1/admin/resources", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "resources", SortOrder: 503, IsSystem: true, IsEnabled: true},
	{Code: "admin:resources:update", Name: "seedResources.admin.resources.update", Path: "/api/v1/admin/resources/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "resources", SortOrder: 504, IsSystem: true, IsEnabled: true},
	{Code: "admin:resources:delete", Name: "seedResources.admin.resources.delete", Path: "/api/v1/admin/resources/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "resources", SortOrder: 505, IsSystem: true, IsEnabled: true},

	// ===== Roles =====
	{Code: "admin:roles:list", Name: "seedResources.admin.roles.list", Path: "/api/v1/admin/roles", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "roles", SortOrder: 600, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:detail", Name: "seedResources.admin.roles.detail", Path: "/api/v1/admin/roles/:role", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "roles", SortOrder: 601, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:create", Name: "seedResources.admin.roles.create", Path: "/api/v1/admin/roles", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "roles", SortOrder: 602, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:update", Name: "seedResources.admin.roles.update", Path: "/api/v1/admin/roles/:role", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "roles", SortOrder: 603, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:delete", Name: "seedResources.admin.roles.delete", Path: "/api/v1/admin/roles/:role", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "roles", SortOrder: 604, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:permissions", Name: "seedResources.admin.roles.permissions", Path: "/api/v1/admin/roles/:role/permissions", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "roles", SortOrder: 605, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:set-permissions", Name: "seedResources.admin.roles.setPermissions", Path: "/api/v1/admin/roles/:role/permissions", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "roles", SortOrder: 607, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:users", Name: "seedResources.admin.roles.users", Path: "/api/v1/admin/roles/:role/users", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "roles", SortOrder: 609, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:hierarchy", Name: "seedResources.admin.roles.hierarchy", Path: "/api/v1/admin/roles/:role/hierarchy", Actions: StringSlice{ActionGET, ActionPUT}, Category: CategoryAdmin, Module: "roles", SortOrder: 610, IsSystem: true, IsEnabled: true},
	{Code: "admin:roles:data-scope", Name: "seedResources.admin.roles.dataScope", Path: "/api/v1/admin/roles/:role/data-scope", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "roles", SortOrder: 611, IsSystem: true, IsEnabled: true},
	{Code: "admin:departments:list", Name: "seedResources.admin.departments.list", Path: "/api/v1/admin/departments", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "departments", SortOrder: 620, IsSystem: true, IsEnabled: true},
	{Code: "admin:departments:create", Name: "seedResources.admin.departments.create", Path: "/api/v1/admin/departments", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "departments", SortOrder: 621, IsSystem: true, IsEnabled: true},
	{Code: "admin:departments:update", Name: "seedResources.admin.departments.update", Path: "/api/v1/admin/departments/:id", Actions: StringSlice{ActionPUT}, Category: CategoryAdmin, Module: "departments", SortOrder: 622, IsSystem: true, IsEnabled: true},
	{Code: "admin:departments:delete", Name: "seedResources.admin.departments.delete", Path: "/api/v1/admin/departments/:id", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "departments", SortOrder: 623, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:constraints", Name: "seedResources.admin.authorization.constraints", Path: "/api/v1/admin/authorization/constraints", Actions: StringSlice{ActionGET, ActionPOST}, Category: CategoryAdmin, Module: "authorization", SortOrder: 620, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:constraint", Name: "seedResources.admin.authorization.constraint", Path: "/api/v1/admin/authorization/constraints/:id", Actions: StringSlice{ActionPUT, ActionDELETE}, Category: CategoryAdmin, Module: "authorization", SortOrder: 621, IsSystem: true, IsEnabled: true},

	// ===== Authorization / User Roles =====
	{Code: "admin:authorization:user-roles", Name: "seedResources.admin.authorization.userRoles", Path: "/api/v1/admin/authorization/users/:username/roles", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 630, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:add-user-role", Name: "seedResources.admin.authorization.addUserRole", Path: "/api/v1/admin/authorization/users/:username/roles", Actions: StringSlice{ActionPOST}, Category: CategoryAdmin, Module: "authorization", SortOrder: 631, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:delete-user-role", Name: "seedResources.admin.authorization.deleteUserRole", Path: "/api/v1/admin/authorization/users/:username/roles/:role", Actions: StringSlice{ActionDELETE}, Category: CategoryAdmin, Module: "authorization", SortOrder: 632, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:user-permissions", Name: "seedResources.admin.authorization.userPermissions", Path: "/api/v1/admin/authorization/users/:username/permissions", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 633, IsSystem: true, IsEnabled: true},

	// ===== Authorization / Metadata =====
	{Code: "admin:authorization:subjects", Name: "seedResources.admin.authorization.subjects", Path: "/api/v1/admin/authorization/metadata/subjects", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 640, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:objects", Name: "seedResources.admin.authorization.objects", Path: "/api/v1/admin/authorization/metadata/objects", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 641, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:actions", Name: "seedResources.admin.authorization.actions", Path: "/api/v1/admin/authorization/metadata/actions", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 642, IsSystem: true, IsEnabled: true},
	{Code: "admin:authorization:policies", Name: "seedResources.admin.authorization.policies", Path: "/api/v1/admin/authorization/metadata/policies", Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "authorization", SortOrder: 643, IsSystem: true, IsEnabled: true},
}

var DefaultRoles = []Role{
	{Code: RoleAdmin, Name: "seedRoles.admin", Description: "seedRoles.adminDesc", IsSystem: true, IsEnabled: true, DataScope: DataScopeAll},
	{Code: RoleAuditor, Name: "seedRoles.auditor", Description: "seedRoles.auditorDesc", IsSystem: true, IsEnabled: true, DataScope: DataScopeAll},
	{Code: RoleUser, Name: "seedRoles.user", Description: "seedRoles.userDesc", IsSystem: true, IsEnabled: true, DataScope: DataScopeSelf},
}
