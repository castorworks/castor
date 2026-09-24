package middleware

import (
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// ActorContext 放在 JWT 中间件之后：把已认证的调用者（用户 ID、用户名、来源 IP）写进
// 请求 context。handler 常把 c.Request.Context() 交给服务层，服务层据此记审计
// （如资产被业务引用而转为公开），不会因为拿不到 gin.Context 而丢掉操作者。
func ActorContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := ucontext.Actor{UserID: ucontext.GetUserID(c), Username: ucontext.GetUsername(c), IP: c.ClientIP()}
		c.Request = c.Request.WithContext(ucontext.WithActor(c.Request.Context(), actor))
		c.Next()
	}
}
