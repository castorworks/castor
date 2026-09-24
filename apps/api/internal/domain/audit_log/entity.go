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
	AuditLogTypeRolePermissionsSet AuditLogType = "ROLE_PERMISSIONS_SET"
	AuditLogTypeResourceCreate     AuditLogType = "RESOURCE_CREATE"
	AuditLogTypeResourceUpdate     AuditLogType = "RESOURCE_UPDATE"
	AuditLogTypeResourceDelete     AuditLogType = "RESOURCE_DELETE"
	AuditLogTypeMenuCreate         AuditLogType = "MENU_CREATE"
	AuditLogTypeMenuUpdate         AuditLogType = "MENU_UPDATE"
	AuditLogTypeMenuDelete         AuditLogType = "MENU_DELETE"
	AuditLogTypeSoDCreate          AuditLogType = "SOD_CONSTRAINT_CREATE"
	AuditLogTypeSoDUpdate          AuditLogType = "SOD_CONSTRAINT_UPDATE"
	AuditLogTypeSoDDelete          AuditLogType = "SOD_CONSTRAINT_DELETE"
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
	AuditLogTypeAssetCreate        AuditLogType = "ASSET_CREATE"
	AuditLogTypeAssetUpdate        AuditLogType = "ASSET_UPDATE"
	AuditLogTypeAssetDelete        AuditLogType = "ASSET_DELETE"
	AuditLogTypeSettingCreate      AuditLogType = "SETTING_CREATE"
	AuditLogTypeSettingUpdate      AuditLogType = "SETTING_UPDATE"
	AuditLogTypeSettingDelete      AuditLogType = "SETTING_DELETE"
	AuditLogTypeDictTypeCreate     AuditLogType = "DICT_TYPE_CREATE"
	AuditLogTypeDictTypeUpdate     AuditLogType = "DICT_TYPE_UPDATE"
	AuditLogTypeDictTypeDelete     AuditLogType = "DICT_TYPE_DELETE"
	AuditLogTypeDictItemCreate     AuditLogType = "DICT_ITEM_CREATE"
	AuditLogTypeDictItemUpdate     AuditLogType = "DICT_ITEM_UPDATE"
	AuditLogTypeDictItemDelete     AuditLogType = "DICT_ITEM_DELETE"
	AuditLogTypeProfileUpdate      AuditLogType = "PROFILE_UPDATE"
	AuditLogTypeAvatarUpdate       AuditLogType = "AVATAR_UPDATE"
	AuditLogTypeActiveRolesChange  AuditLogType = "ACTIVE_ROLES_CHANGE"
	AuditLogTypeAuditLogCleanup    AuditLogType = "AUDIT_LOG_CLEANUP"
	AuditLogTypeLoginHistoryDelete AuditLogType = "LOGIN_HISTORY_DELETE"
	AuditLogTypeSessionRevoke      AuditLogType = "SESSION_REVOKE"
	AuditLogTypeDepartmentCreate   AuditLogType = "DEPT_CREATE"
	AuditLogTypeDepartmentUpdate   AuditLogType = "DEPT_UPDATE"
	AuditLogTypeDepartmentDelete   AuditLogType = "DEPT_DELETE"
	AuditLogTypeJobUpdate          AuditLogType = "JOB_UPDATE"
	AuditLogTypeJobRun             AuditLogType = "JOB_RUN"
	AuditLogTypeUserExport         AuditLogType = "USER_EXPORT"
	AuditLogTypeUserImport         AuditLogType = "USER_IMPORT"
	AuditLogTypeAuditLogExport     AuditLogType = "AUDIT_LOG_EXPORT"
	AuditLogTypeLoginHistoryExport AuditLogType = "LOGIN_HISTORY_EXPORT"
	AuditLogTypeMFASetup           AuditLogType = "MFA_SETUP"
	AuditLogTypeMFAEnable          AuditLogType = "MFA_ENABLE"
	AuditLogTypeMFADisable         AuditLogType = "MFA_DISABLE"
	AuditLogTypeMFARecoveryCodes   AuditLogType = "MFA_RECOVERY_CODES"
	AuditLogTypeMFAReset           AuditLogType = "MFA_RESET"
	AuditLogTypeIdentityLink       AuditLogType = "IDENTITY_LINK"
	AuditLogTypeIdentityUnlink     AuditLogType = "IDENTITY_UNLINK"
	AuditLogTypeOIDCProviderCreate AuditLogType = "OIDC_PROVIDER_CREATE"
	AuditLogTypeOIDCProviderUpdate AuditLogType = "OIDC_PROVIDER_UPDATE"
	AuditLogTypeOIDCProviderDelete AuditLogType = "OIDC_PROVIDER_DELETE"
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
