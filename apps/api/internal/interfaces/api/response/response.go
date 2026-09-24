package response

import (
	"net/http"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/log"
	gi18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Response 通用 API 响应结构
type Response struct {
	Code    int    `json:"code"`    // 状态码
	Data    any    `json:"data"`    // 响应数据
	Message string `json:"message"` // 响应消息
}

// ListData 列表数据结构
type ListData struct {
	Total      int64 `json:"total"`      // 总数
	List       any   `json:"list"`       // 列表数据
	Page       int   `json:"page"`       // 当前页码
	PageSize   int   `json:"pageSize"`   // 每页数量
	TotalPages int   `json:"totalPages"` // 总页数
}

// wrapResult 包装响应结果
func wrapResult(code int, data any, message string) map[string]any {
	return gin.H{
		"code":    code,
		"data":    data,
		"message": message,
	}
}

// devOnlyErrData returns the underlying err.Error() string only when the server
// is running in Development mode; otherwise returns nil so internal details
// (binding errors, ORM messages, file paths, ...) never leak to clients.
func devOnlyErrData(err error) any {
	if err == nil {
		return nil
	}
	if config.C.General.Development {
		return err.Error()
	}
	return nil
}

// getI18nMessage 获取国际化消息
func getI18nMessage(c *gin.Context, key string) string {
	return gi18n.MustGetMessage(c, key)
}

// getI18nMessageWithParams 获取带参数的国际化消息
func getI18nMessageWithParams(c *gin.Context, key string, params map[string]any) string {
	return gi18n.MustGetMessage(c, &i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: params,
	})
}

// wrapResultI18n 包装国际化响应结果
func wrapResultI18n(c *gin.Context, data any, msgKey string) map[string]any {
	message := getI18nMessage(c, msgKey)
	code := GetCode(msgKey)
	return wrapResult(code, data, message)
}

// wrapResultI18nWithParams 包装带参数的国际化响应结果
func wrapResultI18nWithParams(c *gin.Context, data any, msgKey string, params map[string]any) map[string]any {
	message := getI18nMessageWithParams(c, msgKey, params)
	code := GetCode(msgKey)
	return wrapResult(code, data, message)
}

// ====================
// Raw 响应
// ====================

// Raw 直接返回数据
func Raw(c *gin.Context, statusCode int, obj any) {
	c.JSON(statusCode, obj)
}

// ====================
// 200 成功响应
// ====================

// Success 返回成功响应
func Success(c *gin.Context, obj any) {
	c.JSON(http.StatusOK, wrapResultI18n(c, obj, MsgSuccess))
}

// SuccessEmpty 返回空 JSON 的成功响应
func SuccessEmpty(c *gin.Context) {
	c.JSON(http.StatusOK, wrapResultI18n(c, gin.H{}, MsgSuccess))
}

// SuccessList 返回列表数据的成功响应
func SuccessList(c *gin.Context, total int64, list any) {
	c.JSON(http.StatusOK, wrapResultI18n(c, gin.H{"total": total, "list": list}, MsgSuccess))
}

// SuccessListPaged 返回带分页元数据的列表数据成功响应
func SuccessListPaged(c *gin.Context, total int64, list any, page, pageSize int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, wrapResultI18n(c, gin.H{
		"total":      total,
		"list":       list,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
	}, MsgSuccess))
}

// SuccessI18n 返回自定义 i18n 消息的成功响应
func SuccessI18n(c *gin.Context, data any, msgKey string) {
	c.JSON(http.StatusOK, wrapResultI18n(c, data, msgKey))
}

// SuccessI18nWithParams 返回带参数的 i18n 消息的成功响应
func SuccessI18nWithParams(c *gin.Context, data any, msgKey string, params map[string]any) {
	c.JSON(http.StatusOK, wrapResultI18nWithParams(c, data, msgKey, params))
}

// ====================
// 400 错误请求
// ====================

// BadRequest 返回 400 错误
func BadRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, wrapResultI18n(c, nil, MsgBadRequest))
}

// BadRequestErr 返回 400 错误（带错误信息）
// 仅在 Development 模式下回显 err.Error()，生产环境只记录日志
func BadRequestErr(c *gin.Context, err error) {
	log.Err(c, err).Msg("BadRequest")
	c.JSON(http.StatusBadRequest, wrapResultI18n(c, devOnlyErrData(err), MsgBadRequest))
}

// BadRequestI18n 返回 400 错误（带 i18n 消息）
func BadRequestI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusBadRequest, wrapResultI18n(c, nil, msgKey))
}

// RequestTooLargeAbort 返回 413 错误并中止请求
func RequestTooLargeAbort(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, wrapResultI18n(c, nil, ErrRequestTooLarge))
}

// BadRequestAbort 返回 400 错误并中止请求
func BadRequestAbort(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, wrapResultI18n(c, nil, MsgBadRequest))
}

// ====================
// 401 未认证
// ====================

// Unauthorized 返回 401 错误
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, wrapResultI18n(c, nil, MsgUnauthorized))
}

// UnauthorizedAbort 返回 401 错误并中止请求
func UnauthorizedAbort(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, wrapResultI18n(c, nil, MsgUnauthorized))
}

// UnauthorizedErr 返回 401 错误（带错误信息）
// 仅在 Development 模式下回显 err.Error()，生产环境只记录日志
func UnauthorizedErr(c *gin.Context, err error) {
	log.Err(c, err).Msg("Unauthorized")
	c.JSON(http.StatusUnauthorized, wrapResultI18n(c, devOnlyErrData(err), MsgUnauthorized))
}

// UnauthorizedI18n 返回 401 错误（带 i18n 消息）
func UnauthorizedI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusUnauthorized, wrapResultI18n(c, nil, msgKey))
}

// ====================
// 403 禁止访问
// ====================

// Forbidden 返回 403 错误
func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, wrapResultI18n(c, nil, MsgForbidden))
}

// ForbiddenErr 返回 403 错误（带错误信息）
// 仅在 Development 模式下回显 err.Error()，生产环境只记录日志
func ForbiddenErr(c *gin.Context, err error) {
	log.Warn(c).Err(err).Msg("Forbidden")
	c.JSON(http.StatusForbidden, wrapResultI18n(c, devOnlyErrData(err), MsgForbidden))
}

// ForbiddenI18n 返回 403 错误（带 i18n 消息）
func ForbiddenI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusForbidden, wrapResultI18n(c, nil, msgKey))
}

// ====================
// 404 未找到
// ====================

// NotFound 返回 404 错误
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, wrapResultI18n(c, nil, MsgNotFound))
}

// NotFoundErr 返回 404 错误（带错误信息）
// 仅在 Development 模式下回显 err.Error()，生产环境只记录日志
func NotFoundErr(c *gin.Context, err error) {
	log.Warn(c).Err(err).Msg("NotFound")
	c.JSON(http.StatusNotFound, wrapResultI18n(c, devOnlyErrData(err), MsgNotFound))
}

// NotFoundI18n 返回 404 错误（带 i18n 消息）
func NotFoundI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusNotFound, wrapResultI18n(c, nil, msgKey))
}

// ====================
// 409 冲突
// ====================

// Conflict 返回 409 错误
func Conflict(c *gin.Context) {
	c.JSON(http.StatusConflict, wrapResultI18n(c, nil, MsgConflict))
}

// ConflictErr 返回 409 错误（带错误信息）
// 仅在 Development 模式下回显 err.Error()，生产环境只记录日志
func ConflictErr(c *gin.Context, err error) {
	log.Warn(c).Err(err).Msg("Conflict")
	c.JSON(http.StatusConflict, wrapResultI18n(c, devOnlyErrData(err), MsgConflict))
}

// ConflictI18n 返回 409 错误（带 i18n 消息）
func ConflictI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusConflict, wrapResultI18n(c, nil, msgKey))
}

// ====================
// 429 请求过多
// ====================

// TooManyRequestsAbort 返回 429 错误并中止请求
func TooManyRequestsAbort(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, wrapResultI18n(c, nil, MsgTooManyRequests))
}

// ====================
// 500 内部错误
// ====================

// InternalServerError 返回 500 错误
func InternalServerError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, wrapResultI18n(c, nil, MsgInternalServerError))
}

// InternalServerErrorAbort 返回 500 错误并中止请求
func InternalServerErrorAbort(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, wrapResultI18n(c, nil, MsgInternalServerError))
}

// InternalServerErrorErr 返回 500 错误（带错误信息）
// 注意：生产环境不返回详细错误信息，仅记录日志
func InternalServerErrorErr(c *gin.Context, err error) {
	log.Err(c, err).Msg("InternalServerError")
	// 仅在开发模式下返回详细错误信息
	if config.C.General.Development {
		c.JSON(http.StatusInternalServerError, wrapResultI18n(c, err.Error(), MsgInternalServerError))
		return
	}
	c.JSON(http.StatusInternalServerError, wrapResultI18n(c, nil, MsgInternalServerError))
}

// InternalServerErrorI18n 返回 500 错误（带 i18n 消息）
func InternalServerErrorI18n(c *gin.Context, msgKey string) {
	c.JSON(http.StatusInternalServerError, wrapResultI18n(c, nil, msgKey))
}

// Message 返回 key 在请求语言下的文案，用于表头、单元格一类不走响应消息的文本。
func Message(c *gin.Context, key string) string {
	return getI18nMessage(c, key)
}

// Language 返回本次请求实际采用的语言（zh/en/ja/ko）。取自语言包里的 LanguageCode 条目，
// 与消息翻译的语言协商结果（含回退到默认语言）始终一致。
func Language(c *gin.Context) string {
	return getI18nMessage(c, "LanguageCode")
}
