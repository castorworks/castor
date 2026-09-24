package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hyperits/gosuite/logger"
	"github.com/rs/zerolog"
)

// LogRequests 记录 API 访问日志
// 使用结构化字段输出，携带 requestId 用于链路追踪，按状态码区分日志级别
func LogRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := GetRequestID(c)

		if query != "" {
			path = path + "?" + query
		}

		l := logger.Logger()
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = l.Error()
		case status >= 400:
			event = l.Warn()
		default:
			event = l.Info()
		}

		event.
			Str("requestId", requestID).
			Int("status", status).
			Dur("latency", latency).
			Str("clientIP", clientIP).
			Str("method", method).
			Str("path", path).
			Int("bodySize", c.Writer.Size()).
			Msg("API request")
	}
}
