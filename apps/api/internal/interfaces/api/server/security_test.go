package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	original := config.C
	config.C = new(config.Config)
	config.C.General.Development = false
	t.Cleanup(func() { config.C = original })

	engine := gin.New()
	engine.Use(SecurityHeaders())
	engine.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)

	for header, want := range map[string]string{
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                   "DENY",
		"Referrer-Policy":                   "strict-origin-when-cross-origin",
		"Permissions-Policy":                "camera=(), geolocation=(), microphone=()",
		"X-Permitted-Cross-Domain-Policies": "none",
		"Strict-Transport-Security":         "max-age=31536000; includeSubDomains",
		"Cache-Control":                     "no-store",
	} {
		if got := res.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestSecurityHeadersDoNotPinDevelopmentHTTP(t *testing.T) {
	original := config.C
	config.C = new(config.Config)
	config.C.General.Development = true
	t.Cleanup(func() { config.C = original })

	engine := gin.New()
	engine.Use(SecurityHeaders())
	engine.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))

	if got := res.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("development HTTP response unexpectedly set HSTS: %q", got)
	}
}

// 生产模式不能跑在 gin 的调试模式下，也不应有 gin 自带的纯文本访问日志；
// panic 回统一的 JSON 500，而不是断开连接或输出纯文本。
func TestNewGinEngineProductionMode(t *testing.T) {
	original, originalMode := config.C, gin.Mode()
	config.C = new(config.Config)
	config.C.General.Development = false
	config.C.GinLogger.FilePath = filepath.Join(t.TempDir(), "gin.log")
	config.C.I18n.RootPath = filepath.Join("..", "..", "..", "..", "configs", "i18n")
	t.Cleanup(func() { config.C = original; gin.SetMode(originalMode) })
	gin.SetMode(gin.DebugMode)

	engine, err := NewGinEngine()
	if err != nil {
		t.Fatal(err)
	}
	if gin.Mode() != gin.ReleaseMode {
		t.Fatalf("gin mode = %s, want release outside development", gin.Mode())
	}
	engine.GET("/boom", func(*gin.Context) { panic("kaboom") })
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if res.Code != http.StatusInternalServerError || !strings.Contains(res.Body.String(), `"code":500`) {
		t.Fatalf("panic response = %d %s", res.Code, res.Body.String())
	}
}
