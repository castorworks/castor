package ucontext

import (
	"context"

	"github.com/gin-gonic/gin"
)

// AuditContext 审计上下文，用于在 Model 层获取操作用户信息
// 这样可以避免 Model 层直接依赖 gin.Context
type AuditContext struct {
	context.Context
	UserID uint
}

// NewAuditContext 创建审计上下文
func NewAuditContext(ctx context.Context, userID uint) *AuditContext {
	return &AuditContext{
		Context: ctx,
		UserID:  userID,
	}
}

// GetAuditUserID 从 context 中获取审计用户 ID
func GetAuditUserID(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}
	if ac, ok := ctx.(*AuditContext); ok {
		return ac.UserID
	}
	return 0
}

// contextKey 用于 context.WithValue 的键类型
type contextKey string

const (
	auditUserIDKey contextKey = "audit_user_id"
	actorKey       contextKey = "actor"
)

// Actor 发起当前请求的已登录用户。认证中间件把它放进请求 context，服务层记审计时
// 不必依赖 gin.Context 就能拿到操作者与来源 IP。
type Actor struct {
	UserID   uint
	Username string
	IP       string
}

// WithActor 将操作者存入 context
func WithActor(ctx context.Context, actor Actor) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, actorKey, actor)
}

// ActorFromContext 取出当前请求的操作者；传入 gin.Context 时直接从 JWT claims 读取。
// 没有登录身份（定时任务、公开接口）时返回零值。
func ActorFromContext(ctx context.Context) Actor {
	if ctx == nil {
		return Actor{}
	}
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return Actor{UserID: GetUserID(ginCtx), Username: GetUsername(ginCtx), IP: ginCtx.ClientIP()}
	}
	if actor, ok := ctx.Value(actorKey).(Actor); ok {
		return actor
	}
	return Actor{UserID: AuditUserIDFromContext(ctx)}
}

// WithAuditUserID 将用户 ID 存入 context
func WithAuditUserID(ctx context.Context, userID uint) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditUserIDKey, userID)
}

// AuditUserIDFromContext 从 context 中获取审计用户 ID（通用方法）
// 检查顺序：AuditContext → context.Value → gin.Context（兼容回退）
func AuditUserIDFromContext(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}
	// 首先检查 AuditContext
	if ac, ok := ctx.(*AuditContext); ok {
		return ac.UserID
	}
	// 然后检查 context.Value
	if userID, ok := ctx.Value(auditUserIDKey).(uint); ok {
		return userID
	}
	if actor, ok := ctx.Value(actorKey).(Actor); ok {
		return actor.UserID
	}
	// 兼容回退：当 handler 直接传递 gin.Context 时，从 JWT claims 中提取
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return GetUserID(ginCtx)
	}
	return 0
}
