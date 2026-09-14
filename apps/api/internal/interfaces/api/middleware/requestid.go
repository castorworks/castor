package middleware

import (
	"github.com/castorworks/castor/internal/pkg/generator"
	"github.com/gin-gonic/gin"
)

const (
	// HeaderXRequestID 请求 ID 的 HTTP 头名称
	HeaderXRequestID = "X-Request-ID"
	// ContextKeyRequestID 请求 ID 在 Gin Context 中的 key
	ContextKeyRequestID = "requestId"
)

// RequestID 请求 ID 中间件
// 为每个请求生成唯一 ID，用于日志追踪和问题排查
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用客户端传入的 Request ID
		requestID := c.GetHeader(HeaderXRequestID)
		if requestID == "" {
			requestID = generator.GenerateShortUUIDString()
		}

		// 设置到 context 和响应头
		c.Set(ContextKeyRequestID, requestID)
		c.Header(HeaderXRequestID, requestID)

		c.Next()
	}
}

// GetRequestID 从 Gin Context 中获取请求 ID
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(ContextKeyRequestID); exists {
		return id.(string)
	}
	return ""
}
