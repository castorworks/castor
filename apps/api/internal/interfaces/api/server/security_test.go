package server

import (
	"net/http"
	"net/http/httptest"
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
