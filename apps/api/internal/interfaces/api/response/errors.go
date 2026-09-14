package response

// BusinessErrorCode 业务错误码
// 错误码分为 HTTP 状态码 + 业务错误码两层：
//   - HTTP 状态码用于网关/基础设施路由（200/400/401/403/404/409/429/500）
//   - 业务错误码用于前端精准识别错误类型，支持多语言环境下保持稳定的错误处理逻辑
//
// 编码规则：
//   - 1xxx: 通用业务错误
//   - 2xxx: 认证/授权错误
//   - 3xxx: 用户相关错误
//   - 4xxx: 资产/文件相关错误
//   - 5xxx: 系统配置错误
//   - 9xxx: 系统内部错误
const (
	// === 成功 ===
	ErrCodeSuccess = 0 // 操作成功

	// === 1xxx: 通用业务错误 ===
	ErrCodeBadRequest      = 1000 // 参数错误
	ErrCodeInvalidInput    = 1001 // 输入校验失败
	ErrCodeNotFound        = 1002 // 资源不存在
	ErrCodeConflict        = 1003 // 资源冲突
	ErrCodeAlreadyExists   = 1004 // 资源已存在
	ErrCodeTooManyRequests = 1009 // 请求过于频繁

	// === 2xxx: 认证/授权错误 ===
	ErrCodeUnauthorized       = 2000 // 未认证或 Token 无效
	ErrCodeForbidden          = 2001 // 无权限访问
	ErrCodeInvalidCaptcha     = 2002 // 验证码错误
	ErrCodeInvalidCredentials = 2003 // 用户名或密码错误
	ErrCodeInvalidConfirmCode = 2004 // 验证码/确认码错误
	ErrCodeInvalidLoginMethod = 2005 // 不支持的登录方式
	ErrCodeAccountDisabled    = 2006 // 账号已禁用
	ErrCodeAccountLocked      = 2007 // 账号已锁定
	ErrCodeAccountExpired     = 2008 // 账号已过期
	ErrCodeCredentialExpired  = 2009 // 凭证已过期
	ErrCodeTooManyAttempts    = 2010 // 登录尝试次数过多
	ErrCodeRegisterDisabled   = 2011 // 注册功能未开放

	// === 3xxx: 用户相关错误 ===
	ErrCodeUserNotFound     = 3001 // 用户不存在
	ErrCodeUserCreateFailed = 3002 // 用户创建失败
	ErrCodeUserUpdateFailed = 3003 // 用户更新失败

	// === 4xxx: 资产/文件相关错误 ===
	ErrCodeAssetUploadFailed = 4001 // 文件上传失败
	ErrCodeAssetNotFound     = 4002 // 资产不存在
	ErrCodeAssetTooLarge     = 4003 // 文件超出大小限制
	ErrCodeAssetInvalidType  = 4004 // 不支持的文件类型

	// === 5xxx: 系统配置错误 ===
	ErrCodeSettingNotFound   = 5001 // 配置项不存在
	ErrCodeSettingSaveFailed = 5002 // 配置保存失败

	// === 9xxx: 系统内部错误 ===
	ErrCodeInternalError = 9000 // 服务器内部错误
	ErrCodeDatabaseError = 9001 // 数据库操作失败
	ErrCodeCacheError    = 9002 // 缓存操作失败
	ErrCodeStorageError  = 9003 // 存储服务异常
)

// GetHTTPStatus 根据业务错误码获取对应的 HTTP 状态码
// 委托 GetHTTPStatusForErrCode 使用统一的 errCodeRegistry 映射表
func GetHTTPStatus(errCode int) int {
	return GetHTTPStatusForErrCode(errCode)
}

// ErrorResponse 带业务错误码的响应结构
// 前端可通过 errorCode 精准判断错误类型，不依赖 message 文本
type ErrorResponse struct {
	Code      int    `json:"code"`      // HTTP 状态码（保持向后兼容）
	ErrorCode int    `json:"errorCode"` // 业务错误码（新增，用于前端精准匹配）
	Data      any    `json:"data"`      // 响应数据（错误时通常为 null）
	Message   string `json:"message"`   // 用户可见的错误消息（i18n）
	Detail    string `json:"detail"`    // 详细错误信息（仅开发模式）
}
