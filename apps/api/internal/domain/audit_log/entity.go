package audit_log

import "time"

// AuditLogType 审计日志类型
type AuditLogType string

const (
	AuditLogTypeAddRole            AuditLogType = "ADD_ROLE"
	AuditLogTypeUpdateRole         AuditLogType = "UPDATE_ROLE"
	AuditLogTypeDeleteRole         AuditLogType = "DELETE_ROLE"
	AuditLogTypeAddRoleForUser     AuditLogType = "ADD_ROLE_FOR_USER"
	AuditLogTypeDeleteRoleForUser  AuditLogType = "DELETE_ROLE_FOR_USER"
	AuditLogTypeAddPermission      AuditLogType = "ADD_PERMISSION"
	AuditLogTypeDeletePermission   AuditLogType = "DELETE_PERMISSION"
	AuditLogTypeLogin              AuditLogType = "LOGIN"
	AuditLogTypeLogout             AuditLogType = "LOGOUT"
	AuditLogTypePasswordChange     AuditLogType = "PASSWORD_CHANGE"
	AuditLogTypePasswordReset      AuditLogType = "PASSWORD_RESET"
	AuditLogTypeUserCreate         AuditLogType = "USER_CREATE"
	AuditLogTypeUserUpdate         AuditLogType = "USER_UPDATE"
	AuditLogTypeUserDelete         AuditLogType = "USER_DELETE"
	AuditLogTypeNotificationCreate AuditLogType = "NOTIFICATION_CREATE"
	AuditLogTypeNotificationUpdate AuditLogType = "NOTIFICATION_UPDATE"
	AuditLogTypeNotificationDelete AuditLogType = "NOTIFICATION_DELETE"
)

// AuditLog 审计日志实体（纯 Domain 模型，无 ORM 依赖）
type AuditLog struct {
	ID         uint         `json:"id"`
	CreatedAt  time.Time    `json:"createdAt"`
	LogType    AuditLogType `json:"logType"`
	Operator   string       `json:"operator"`
	OperatorID uint         `json:"operatorId"`
	Target     string       `json:"target"`
	Details    string       `json:"details"`
	IpAddr     string       `json:"ipAddr"`
	Success    bool         `json:"success"`
}
