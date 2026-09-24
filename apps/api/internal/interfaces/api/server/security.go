package server

import (
	"strings"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds browser protections that are safe for the API and the
// browser-facing deployment. HSTS is only emitted for HTTPS requests so a
// direct local HTTP deployment is not pinned accidentally.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")

		if !config.C.General.Development && isHTTPSRequest(c) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/v1/account/") {
			c.Header("Cache-Control", "no-store")
		}

		c.Next()
	}
}

func isHTTPSRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}
