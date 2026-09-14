package dictionary

import "time"

// DictType 字典类型（纯 Domain 模型，无 ORM 依赖）
type DictType struct {
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
	SortOrder   int       `json:"sortOrder"`
}

// DictItem 字典项（纯 Domain 模型，无 ORM 依赖）
type DictItem struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   uint      `json:"createdBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   uint      `json:"updatedBy"`
	TypeCode    string    `json:"typeCode"`
	Label       string    `json:"label"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	Extra       string    `json:"extra"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	ParentID    *uint     `json:"parentId"`
	IsDefault   bool      `json:"isDefault"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
}

// DefaultDictTypes 默认字典类型
var DefaultDictTypes = []DictType{
	{Code: "gender", Name: "Gender", IsSystem: true, IsEnabled: true, SortOrder: 1},
	{Code: "status", Name: "Status", IsSystem: true, IsEnabled: true, SortOrder: 2},
	{Code: "priority", Name: "Priority", IsSystem: true, IsEnabled: true, SortOrder: 3},
	{Code: "boolean", Name: "Boolean", IsSystem: true, IsEnabled: true, SortOrder: 4},
	{Code: "asset_status", Name: "Asset Status", IsSystem: true, IsEnabled: true, SortOrder: 5},
	{Code: "asset_category", Name: "Asset Category", IsSystem: true, IsEnabled: true, SortOrder: 6},
	{Code: "account_source", Name: "Account Source", IsSystem: true, IsEnabled: true, SortOrder: 7},
	{Code: "notification_type", Name: "Notification Type", IsSystem: true, IsEnabled: true, SortOrder: 8},
	{Code: "notification_level", Name: "Notification Level", IsSystem: true, IsEnabled: true, SortOrder: 9},
	{Code: "login_method", Name: "Login Method", IsSystem: true, IsEnabled: true, SortOrder: 10},
	{Code: "audit_log_type", Name: "Audit Log Type", IsSystem: true, IsEnabled: true, SortOrder: 11},
}

// DefaultDictItems 默认字典项
var DefaultDictItems = []DictItem{
	{TypeCode: "gender", Label: "Male", Value: "MALE", SortOrder: 1, IsEnabled: true},
	{TypeCode: "gender", Label: "Female", Value: "FEMALE", SortOrder: 2, IsEnabled: true},
	{TypeCode: "gender", Label: "Unknown", Value: "UNKNOWN", SortOrder: 3, IsEnabled: true},
	{TypeCode: "status", Label: "Enabled", Value: "ENABLED", Color: "green", SortOrder: 1, IsEnabled: true, IsDefault: true},
	{TypeCode: "status", Label: "Disabled", Value: "DISABLED", Color: "red", SortOrder: 2, IsEnabled: true},
	{TypeCode: "status", Label: "Pending", Value: "PENDING", Color: "orange", SortOrder: 3, IsEnabled: true},
	{TypeCode: "priority", Label: "Low", Value: "LOW", Color: "gray", SortOrder: 1, IsEnabled: true},
	{TypeCode: "priority", Label: "Medium", Value: "MEDIUM", Color: "blue", SortOrder: 2, IsEnabled: true, IsDefault: true},
	{TypeCode: "priority", Label: "High", Value: "HIGH", Color: "orange", SortOrder: 3, IsEnabled: true},
	{TypeCode: "priority", Label: "Urgent", Value: "URGENT", Color: "red", SortOrder: 4, IsEnabled: true},
	{TypeCode: "boolean", Label: "Yes", Value: "true", SortOrder: 1, IsEnabled: true},
	{TypeCode: "boolean", Label: "No", Value: "false", SortOrder: 2, IsEnabled: true},
	{TypeCode: "asset_status", Label: "Pending", Value: "PENDING", Color: "orange", Icon: "clock", SortOrder: 1, IsEnabled: true},
	{TypeCode: "asset_status", Label: "Active", Value: "ACTIVE", Color: "green", Icon: "check", SortOrder: 2, IsEnabled: true, IsDefault: true},
	{TypeCode: "asset_status", Label: "Archived", Value: "ARCHIVED", Color: "gray", Icon: "archive", SortOrder: 3, IsEnabled: true},
	{TypeCode: "asset_status", Label: "Deleted", Value: "DELETED", Color: "red", Icon: "close", SortOrder: 4, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Image", Value: "IMAGE", Icon: "media", SortOrder: 1, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Video", Value: "VIDEO", Icon: "video", SortOrder: 2, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Audio", Value: "AUDIO", Icon: "music", SortOrder: 3, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Document", Value: "DOCUMENT", Icon: "post", SortOrder: 4, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Archive", Value: "ARCHIVE", Icon: "archive", SortOrder: 5, IsEnabled: true},
	{TypeCode: "asset_category", Label: "Other", Value: "OTHER", Icon: "page", SortOrder: 6, IsEnabled: true, IsDefault: true},
	{TypeCode: "account_source", Label: "Local", Value: "INTERNAL", Icon: "mail", SortOrder: 1, IsEnabled: true, IsDefault: true},
	{TypeCode: "account_source", Label: "Facebook", Value: "FACEBOOK", Icon: "globe", SortOrder: 2, IsEnabled: true},
	{TypeCode: "account_source", Label: "QQ", Value: "QQ", Icon: "globe", SortOrder: 3, IsEnabled: true},
	{TypeCode: "account_source", Label: "WeChat", Value: "WECHAT", Icon: "globe", SortOrder: 4, IsEnabled: true},
	{TypeCode: "account_source", Label: "Google", Value: "GOOGLE", Icon: "globe", SortOrder: 5, IsEnabled: true},
	{TypeCode: "account_source", Label: "GitHub", Value: "GITHUB", Icon: "globe", SortOrder: 6, IsEnabled: true},
	{TypeCode: "notification_type", Label: "System", Value: "SYSTEM", Color: "blue", Icon: "settings", SortOrder: 1, IsEnabled: true, IsDefault: true},
	{TypeCode: "notification_type", Label: "Announcement", Value: "ANNOUNCE", Color: "purple", Icon: "speaker", SortOrder: 2, IsEnabled: true},
	{TypeCode: "notification_type", Label: "Message", Value: "MESSAGE", Color: "gray", Icon: "mail", SortOrder: 3, IsEnabled: true},
	{TypeCode: "notification_type", Label: "Alert", Value: "ALERT", Color: "red", Icon: "warning", SortOrder: 4, IsEnabled: true},
	{TypeCode: "notification_type", Label: "Task", Value: "TASK", Color: "blue", Icon: "kanban", SortOrder: 5, IsEnabled: true},
	{TypeCode: "notification_level", Label: "Info", Value: "INFO", Color: "blue", SortOrder: 1, IsEnabled: true, IsDefault: true},
	{TypeCode: "notification_level", Label: "Success", Value: "SUCCESS", Color: "green", SortOrder: 2, IsEnabled: true},
	{TypeCode: "notification_level", Label: "Warning", Value: "WARNING", Color: "orange", SortOrder: 3, IsEnabled: true},
	{TypeCode: "notification_level", Label: "Error", Value: "ERROR", Color: "red", SortOrder: 4, IsEnabled: true},
	{TypeCode: "login_method", Label: "Password", Value: "password", SortOrder: 1, IsEnabled: true, IsDefault: true},
	{TypeCode: "login_method", Label: "Email OTP", Value: "email_otp", SortOrder: 2, IsEnabled: true},
	{TypeCode: "login_method", Label: "Mobile OTP", Value: "mobile_otp", SortOrder: 3, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Add Role", Value: "ADD_ROLE", SortOrder: 1, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Delete Role", Value: "DELETE_ROLE", SortOrder: 2, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Assign Role", Value: "ADD_ROLE_FOR_USER", SortOrder: 3, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Remove Role", Value: "DELETE_ROLE_FOR_USER", SortOrder: 4, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Add Permission", Value: "ADD_PERMISSION", SortOrder: 5, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Delete Permission", Value: "DELETE_PERMISSION", SortOrder: 6, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Login", Value: "LOGIN", SortOrder: 7, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Logout", Value: "LOGOUT", SortOrder: 8, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Password Change", Value: "PASSWORD_CHANGE", SortOrder: 9, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "Password Reset", Value: "PASSWORD_RESET", SortOrder: 10, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "User Create", Value: "USER_CREATE", SortOrder: 11, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "User Update", Value: "USER_UPDATE", SortOrder: 12, IsEnabled: true},
	{TypeCode: "audit_log_type", Label: "User Delete", Value: "USER_DELETE", SortOrder: 13, IsEnabled: true},
}
