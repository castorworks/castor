package response

// I18n 消息 Key 常量
// 直接使用字符串常量作为 i18n key，消除冗余映射
const (
	// [basic] 基础 HTTP 状态消息
	MsgSuccess             = "ErrSuccess"
	MsgBadRequest          = "ErrBadRequest"
	MsgUnauthorized        = "ErrUnauthorized"
	MsgForbidden           = "ErrForbidden"
	MsgNotFound            = "ErrNotFound"
	MsgConflict            = "ErrConflict"
	MsgTooManyRequests     = "ErrTooManyRequests"
	MsgInternalServerError = "ErrInternalServerError"

	// [auth] 认证相关消息
	MsgInvalidCaptchaCode        = "ErrInvalidCaptchaCode"
	MsgInvalidConfirmCode        = "ErrInvalidConfirmCode"
	MsgInvalidLoginMethod        = "ErrInvalidLoginMethod"
	MsgInvalidUsernameOrPassword = "ErrInvalidUsernameOrPassword"
	MsgRegisterNotEnabled        = "ErrRegisterNotEnabled"
	MsgUserDisabled              = "ErrUserDisabled"
	MsgUserLocked                = "ErrUserLocked"
	MsgAccountExpired            = "ErrAccountExpired"
	MsgCredentialExpired         = "ErrCredentialExpired"
	MsgTooManyLoginAttempts      = "ErrTooManyLoginAttempts"
	MsgLoginMethodDisabled       = "ErrLoginMethodDisabled"
	InfLoginSuccess              = "InfLoginSuccess"
	InfLogoutSuccess             = "InfLogoutSuccess"

	// [crud] 通用操作消息
	InfCreateSuccess      = "InfCreateSuccess"
	InfUpdateSuccess      = "InfUpdateSuccess"
	InfDeleteSuccess      = "InfDeleteSuccess"
	InfBatchDeleteSuccess = "InfBatchDeleteSuccess"
	InfUploadSuccess      = "InfUploadSuccess"
	InfMoveSuccess        = "InfMoveSuccess"
	InfStatusUpdated      = "InfStatusUpdated"
	InfBatchStatusUpdated = "InfBatchStatusUpdated"

	// [account/user] 账户和用户消息
	InfUserCreated              = "InfUserCreated"
	InfPasswordChanged          = "InfPasswordChanged"
	InfPasswordReset            = "InfPasswordReset"
	InfCodeSent                 = "InfCodeSent"
	ErrUserNotFound             = "ErrUserNotFound"
	ErrEmailAlreadyExists       = "ErrEmailAlreadyExists"
	ErrCurrentPasswordIncorrect = "ErrCurrentPasswordIncorrect"
	ErrPasswordEncryptFailed    = "ErrPasswordEncryptFailed"
	ErrPasswordDecryptFailed    = "ErrPasswordDecryptFailed"
	ErrUsernameAlreadyTaken     = "ErrUsernameAlreadyTaken"
	ErrInvalidCodeType          = "ErrInvalidCodeType"
	ErrCannotModifySystemUser   = "ErrCannotModifySystemUser"
	ErrCannotModifyOwnStatus    = "ErrCannotModifyOwnStatus"
	ErrCannotDeleteSystemUser   = "ErrCannotDeleteSystemUser"
	ErrCannotDeleteOwnAccount   = "ErrCannotDeleteOwnAccount"
	ErrNoFieldsToUpdate         = "ErrNoFieldsToUpdate"
	ErrNameTooLong              = "ErrNameTooLong"
	ErrPasswordTooShort         = "ErrPasswordTooShort"

	// [asset] 资产消息
	InfFileUploaded      = "InfFileUploaded"
	InfFileUpdated       = "InfFileUpdated"
	InfFileDeleted       = "InfFileDeleted"
	InfFileDuplicate     = "InfFileDuplicate"
	ErrFileUploadFailed  = "ErrFileUploadFailed"
	ErrFileTooLarge      = "ErrFileTooLarge"
	ErrRequestTooLarge   = "ErrRequestTooLarge"
	ErrInvalidFilename   = "ErrInvalidFilename"
	ErrInvalidFileType   = "ErrInvalidFileType"
	ErrAssetNotFound     = "ErrAssetNotFound"
	ErrInvalidID         = "ErrInvalidID"
	ErrObjectKeyRequired = "ErrObjectKeyRequired"

	// [authorization] 权限和角色消息
	InfRoleCreated                  = "InfRoleCreated"
	InfRoleUpdated                  = "InfRoleUpdated"
	InfRoleDeleted                  = "InfRoleDeleted"
	InfPermissionUpdated            = "InfPermissionUpdated"
	ErrRoleNotFound                 = "ErrRoleNotFound"
	ErrRoleCodeExists               = "ErrRoleCodeExists"
	ErrPermissionDenied             = "ErrPermissionDenied"
	ErrResourceNotFound             = "ErrResourceNotFound"
	ErrResourceCodeExists           = "ErrResourceCodeExists"
	ErrInvalidResource              = "ErrInvalidResource"
	ErrResourcePermissionConflict   = "ErrResourcePermissionConflict"
	ErrSystemResourceDelete         = "ErrSystemResourceDelete"
	ErrSystemRoleDelete             = "ErrSystemRoleDelete"
	ErrInvalidRole                  = "ErrInvalidRole"
	ErrRoleInUse                    = "ErrRoleInUse"
	ErrInvalidRolePermission        = "ErrInvalidRolePermission"
	ErrRoleHierarchyCycle           = "ErrRoleHierarchyCycle"
	ErrInvalidRoleHierarchy         = "ErrInvalidRoleHierarchy"
	ErrInvalidConstraint            = "ErrInvalidConstraint"
	ErrSSDViolation                 = "ErrSSDViolation"
	ErrDSDViolation                 = "ErrDSDViolation"
	ErrRoleNotAuthorized            = "ErrRoleNotAuthorized"
	ErrAuthorizationSessionNotFound = "ErrAuthorizationSessionNotFound"
	ErrAuthorizationSessionExpired  = "ErrAuthorizationSessionExpired"
	ErrAuthorizationSessionRevoked  = "ErrAuthorizationSessionRevoked"

	// [dictionary] 数据字典消息
	InfDictCreated         = "InfDictCreated"
	InfDictUpdated         = "InfDictUpdated"
	InfDictDeleted         = "InfDictDeleted"
	ErrDictKeyExists       = "ErrDictKeyExists"
	ErrDictItemValueExists = "ErrDictItemValueExists"
	ErrSystemDictDelete    = "ErrSystemDictDelete"

	// [setting] 系统设置消息
	InfSettingUpdated            = "InfSettingUpdated"
	ErrSettingKeyExists          = "ErrSettingKeyExists"
	ErrInvalidSettingValue       = "ErrInvalidSettingValue"
	ErrSystemSettingDelete       = "ErrSystemSettingDelete"
	ErrInvalidLoginMethodSetting = "ErrInvalidLoginMethodSetting"

	// [notification] 通知消息
	InfNotificationSent               = "InfNotificationSent"
	InfNotificationRead               = "InfNotificationRead"
	InfAllMarkedRead                  = "InfAllMarkedRead"
	ErrNotificationRecipientsRequired = "ErrNotificationRecipientsRequired"
	ErrNotificationNotDelivered       = "ErrNotificationNotDelivered"

	// [validation] 验证消息
	ErrFieldRequired      = "ErrFieldRequired"
	ErrFieldTooLong       = "ErrFieldTooLong"
	ErrInvalidEmail       = "ErrInvalidEmail"
	ErrInvalidPhoneFormat = "ErrInvalidPhoneFormat"
	ErrInvalidOrder       = "ErrInvalidOrder"
	ErrInvalidSearchField = "ErrInvalidSearchField"
)

// HTTP 状态码常量
const (
	CodeSuccess             = 200
	CodeBadRequest          = 400
	CodeUnauthorized        = 401
	CodeForbidden           = 403
	CodeNotFound            = 404
	CodeConflict            = 409
	CodePayloadTooLarge     = 413
	CodeTooManyRequests     = 429
	CodeInternalServerError = 500
)

// msgToCode i18n key 到 HTTP 状态码的映射
var msgToCode = map[string]int{
	ErrInvalidMenu: CodeBadRequest, ErrMenuNotFound: CodeNotFound, ErrMenuConflict: CodeConflict, ErrMenuHierarchy: CodeConflict, ErrMenuHasChildren: CodeConflict, ErrMenuPermission: CodeBadRequest,
	InfMenuSaved: CodeSuccess, InfMenuDeleted: CodeSuccess,
	// [basic]
	MsgSuccess:             CodeSuccess,
	MsgBadRequest:          CodeBadRequest,
	MsgUnauthorized:        CodeUnauthorized,
	MsgForbidden:           CodeForbidden,
	MsgNotFound:            CodeNotFound,
	MsgConflict:            CodeConflict,
	MsgTooManyRequests:     CodeTooManyRequests,
	MsgInternalServerError: CodeInternalServerError,

	// [auth] - 业务错误通常返回对应的 HTTP 状态码
	MsgInvalidCaptchaCode:        CodeBadRequest,
	MsgInvalidConfirmCode:        CodeBadRequest,
	MsgInvalidLoginMethod:        CodeBadRequest,
	MsgInvalidUsernameOrPassword: CodeUnauthorized,
	MsgRegisterNotEnabled:        CodeForbidden,
	MsgUserDisabled:              CodeForbidden,
	MsgUserLocked:                CodeForbidden,
	MsgAccountExpired:            CodeForbidden,
	MsgCredentialExpired:         CodeForbidden,
	MsgTooManyLoginAttempts:      CodeTooManyRequests,
	MsgLoginMethodDisabled:       CodeForbidden,

	// [info] - 成功消息
	InfLoginSuccess:  CodeSuccess,
	InfLogoutSuccess: CodeSuccess,

	// [crud]
	InfCreateSuccess:      CodeSuccess,
	InfUpdateSuccess:      CodeSuccess,
	InfDeleteSuccess:      CodeSuccess,
	InfBatchDeleteSuccess: CodeSuccess,
	InfUploadSuccess:      CodeSuccess,
	InfMoveSuccess:        CodeSuccess,
	InfStatusUpdated:      CodeSuccess,
	InfBatchStatusUpdated: CodeSuccess,

	// [account/user]
	InfUserCreated:              CodeSuccess,
	InfPasswordChanged:          CodeSuccess,
	InfPasswordReset:            CodeSuccess,
	InfCodeSent:                 CodeSuccess,
	ErrUserNotFound:             CodeNotFound,
	ErrEmailAlreadyExists:       CodeConflict,
	ErrCurrentPasswordIncorrect: CodeBadRequest,
	ErrPasswordEncryptFailed:    CodeInternalServerError,
	ErrPasswordDecryptFailed:    CodeInternalServerError,
	ErrUsernameAlreadyTaken:     CodeConflict,
	ErrInvalidCodeType:          CodeBadRequest,
	ErrCannotModifySystemUser:   CodeForbidden,
	ErrCannotModifyOwnStatus:    CodeForbidden,
	ErrCannotDeleteSystemUser:   CodeForbidden,
	ErrCannotDeleteOwnAccount:   CodeForbidden,
	ErrNoFieldsToUpdate:         CodeBadRequest,
	ErrNameTooLong:              CodeBadRequest,
	ErrPasswordTooShort:         CodeBadRequest,

	// [asset]
	InfFileUploaded:      CodeSuccess,
	InfFileUpdated:       CodeSuccess,
	InfFileDeleted:       CodeSuccess,
	InfFileDuplicate:     CodeSuccess,
	ErrFileUploadFailed:  CodeInternalServerError,
	ErrFileTooLarge:      CodeBadRequest,
	ErrRequestTooLarge:   CodePayloadTooLarge,
	ErrInvalidFilename:   CodeBadRequest,
	ErrInvalidFileType:   CodeBadRequest,
	ErrAssetNotFound:     CodeNotFound,
	ErrInvalidID:         CodeBadRequest,
	ErrObjectKeyRequired: CodeBadRequest,

	// [authorization]
	InfRoleCreated:                  CodeSuccess,
	InfRoleUpdated:                  CodeSuccess,
	InfRoleDeleted:                  CodeSuccess,
	InfPermissionUpdated:            CodeSuccess,
	ErrRoleNotFound:                 CodeNotFound,
	ErrRoleCodeExists:               CodeConflict,
	ErrPermissionDenied:             CodeForbidden,
	ErrResourceNotFound:             CodeNotFound,
	ErrResourceCodeExists:           CodeConflict,
	ErrInvalidResource:              CodeBadRequest,
	ErrResourcePermissionConflict:   CodeConflict,
	ErrSystemResourceDelete:         CodeForbidden,
	ErrSystemRoleDelete:             CodeForbidden,
	ErrInvalidRole:                  CodeBadRequest,
	ErrRoleInUse:                    CodeConflict,
	ErrRoleHierarchyCycle:           CodeConflict,
	ErrInvalidRoleHierarchy:         CodeBadRequest,
	ErrInvalidConstraint:            CodeBadRequest,
	ErrSSDViolation:                 CodeConflict,
	ErrDSDViolation:                 CodeConflict,
	ErrRoleNotAuthorized:            CodeForbidden,
	ErrAuthorizationSessionNotFound: CodeUnauthorized,
	ErrAuthorizationSessionExpired:  CodeUnauthorized,
	ErrAuthorizationSessionRevoked:  CodeUnauthorized,
	ErrInvalidRolePermission:        CodeBadRequest,

	// [dictionary]
	InfDictCreated:         CodeSuccess,
	InfDictUpdated:         CodeSuccess,
	InfDictDeleted:         CodeSuccess,
	ErrDictKeyExists:       CodeConflict,
	ErrDictItemValueExists: CodeConflict,
	ErrSystemDictDelete:    CodeForbidden,

	// [setting]
	InfSettingUpdated:            CodeSuccess,
	ErrSettingKeyExists:          CodeConflict,
	ErrInvalidSettingValue:       CodeBadRequest,
	ErrSystemSettingDelete:       CodeForbidden,
	ErrInvalidLoginMethodSetting: CodeBadRequest,

	// [notification]
	InfNotificationSent:               CodeSuccess,
	InfNotificationRead:               CodeSuccess,
	InfAllMarkedRead:                  CodeSuccess,
	ErrNotificationRecipientsRequired: CodeBadRequest,
	ErrNotificationNotDelivered:       CodeForbidden,

	// [validation]
	ErrFieldRequired:      CodeBadRequest,
	ErrFieldTooLong:       CodeBadRequest,
	ErrInvalidEmail:       CodeBadRequest,
	ErrInvalidPhoneFormat: CodeBadRequest,
	ErrInvalidOrder:       CodeBadRequest,
	ErrInvalidSearchField: CodeBadRequest,
}

// GetCode 获取 i18n key 对应的 HTTP 状态码
func GetCode(key string) int {
	if code, ok := msgToCode[key]; ok {
		return code
	}
	return CodeInternalServerError
}

const (
	ErrInvalidMenu     = "ErrInvalidMenu"
	ErrMenuNotFound    = "ErrMenuNotFound"
	ErrMenuConflict    = "ErrMenuConflict"
	ErrMenuHierarchy   = "ErrMenuHierarchy"
	ErrMenuHasChildren = "ErrMenuHasChildren"
	ErrMenuPermission  = "ErrMenuPermission"
)

const (
	InfMenuSaved   = "InfMenuSaved"
	InfMenuDeleted = "InfMenuDeleted"
)
