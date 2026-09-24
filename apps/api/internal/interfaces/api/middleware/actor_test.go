package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// 服务层只拿到 c.Request.Context() 时也要知道操作者（如资产转公开的审计）。
func TestActorContextPutsTheCallerOnTheRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var got ucontext.Actor
	router.Use(func(c *gin.Context) {
		// 模拟 JWT 中间件已验证的 claims
		c.Set("JWT_PAYLOAD", jwt.MapClaims{constant.JWT_IDENTITY_KEY: float64(7), constant.JWT_USERNAME: "alice"})
	})
	router.Use(ActorContext())
	router.GET("/", func(c *gin.Context) {
		got = ucontext.ActorFromContext(c.Request.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:4321"
	router.ServeHTTP(httptest.NewRecorder(), req)

	if got.UserID != 7 || got.Username != "alice" || got.IP != "203.0.113.9" {
		t.Fatalf("actor = %+v", got)
	}
}
