package log

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
	"github.com/hyperits/gosuite/logger"
	"github.com/rs/zerolog"
)

const (
	// contextKeyRequestID 请求 ID 在 Gin Context 中的 key
	// 与 middleware.ContextKeyRequestID 保持一致，避免循环引用
	contextKeyRequestID = "requestId"
)

// getRequestID 从 Gin Context 中获取请求 ID
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get(contextKeyRequestID); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// baseEvent 为日志事件附加请求上下文字段
func baseEvent(c *gin.Context, event *zerolog.Event) *zerolog.Event {
	event = event.
		Str("requestId", getRequestID(c)).
		Str("path", c.Request.URL.Path).
		Str("method", c.Request.Method)

	if uid := ucontext.GetUserID(c); uid > 0 {
		event = event.Uint("userId", uid).Str("username", ucontext.GetUsername(c))
	}

	return event
}

// getLogger 获取 zerolog.Logger 指针
func getLogger() *zerolog.Logger {
	l := logger.Logger()
	return &l
}

// Err 从 gin.Context 构建带有请求上下文字段的 error 级别日志事件
func Err(c *gin.Context, err error) *zerolog.Event {
	return baseEvent(c, getLogger().Error()).Err(err)
}

// Warn 从 gin.Context 构建带有请求上下文字段的 warn 级别日志事件
func Warn(c *gin.Context) *zerolog.Event {
	return baseEvent(c, getLogger().Warn())
}

// Info 从 gin.Context 构建带有请求上下文字段的 info 级别日志事件
func Info(c *gin.Context) *zerolog.Event {
	return baseEvent(c, getLogger().Info())
}

// ErrCtx 从任意 context 构建 error 级别日志事件；传入 gin.Context 时附带请求 ID
func ErrCtx(ctx context.Context, err error) *zerolog.Event {
	return ctxEvent(ctx, getLogger().Error()).Err(err)
}

// WarnCtx 从任意 context 构建 warn 级别日志事件；传入 gin.Context 时附带请求 ID
func WarnCtx(ctx context.Context) *zerolog.Event {
	return ctxEvent(ctx, getLogger().Warn())
}

func ctxEvent(ctx context.Context, event *zerolog.Event) *zerolog.Event {
	if ctx == nil {
		return event
	}
	if id, ok := ctx.Value(contextKeyRequestID).(string); ok && id != "" {
		event = event.Str("requestId", id)
	}
	return event
}
