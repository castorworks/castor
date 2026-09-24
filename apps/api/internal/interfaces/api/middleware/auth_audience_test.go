package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

// 两套部署误用了同一个 JwtKey 时，一套签发的 token 在另一套上既不能访问接口，也不能刷新：
// 否则持有 A 的 token 就能在 B 上冒充同 ID 的用户（可能是 B 的管理员）。
func TestTokenFromAnotherInstanceIsRejected(t *testing.T) {
	// Not parallel: NewJwtMiddleware reads package-level config.C.
	orig := config.C.General
	t.Cleanup(func() { config.C.General = orig })
	config.C.General.JwtKey = "shared-key-by-mistake-0123456789abcdef"
	config.C.General.JwtTimeoutHours = 2
	config.C.General.JwtMaxRefreshHours = 168
	config.C.General.Development = true

	newMiddleware := func(instance string) *JwtMiddleware {
		t.Helper()
		config.C.General.InstanceID = instance
		mw, err := NewJwtMiddleware(nil, &sessionRBACStub{validSession: "session-1"},
			&mockTokenBlacklistSvc{blacklisted: map[string]bool{}}, &mockAuditLogSvc{}, nil)
		if err != nil {
			t.Fatalf("NewJwtMiddleware(%s) error = %v", instance, err)
		}
		return mw
	}
	tenantA := newMiddleware("tenant-a")
	tenantB := newMiddleware("tenant-b")

	identity := &user.User{ID: 1, Name: "Admin", Username: "system", AccountSource: "internal", AuthorizationSessionID: "session-1"}
	tokenA, _, err := tenantA.AuthMiddleware.TokenGenerator(identity)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := tenantA.AuthMiddleware.ParseTokenString(tokenA)
	if err != nil {
		t.Fatal(err)
	}
	if aud := jwt.ExtractClaimsFromToken(parsed)["aud"]; aud != "tenant-a" {
		t.Fatalf("token aud = %v, want tenant-a", aud)
	}

	serve := func(mw *JwtMiddleware, method, path string) int {
		router := newI18nTestRouter()
		router.GET("/me", mw.AuthMiddleware.MiddlewareFunc(), func(c *gin.Context) { c.Status(http.StatusOK) })
		router.POST("/refresh", mw.RefreshWithBlacklistCheck)
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	if code := serve(tenantA, http.MethodGet, "/me"); code != http.StatusOK {
		t.Fatalf("issuing instance must accept its own token, status = %d", code)
	}
	if code := serve(tenantB, http.MethodGet, "/me"); code == http.StatusOK {
		t.Fatal("another instance must not accept the token")
	}
	if code := serve(tenantB, http.MethodPost, "/refresh"); code != http.StatusUnauthorized {
		t.Fatalf("another instance must not refresh the token, status = %d", code)
	}
}
