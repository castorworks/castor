package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
)

// mockTokenBlacklistSvc implements service.TokenBlacklistService for testing
type mockTokenBlacklistSvc struct {
	blacklisted map[string]bool
}

func (m *mockTokenBlacklistSvc) AddToBlacklist(_ context.Context, token string, _ time.Duration) error {
	m.blacklisted[token] = true
	return nil
}

func (m *mockTokenBlacklistSvc) IsBlacklisted(_ context.Context, token string) bool {
	return m.blacklisted[token]
}

// mockAuditLogSvc implements service.AuditLogService for testing
type mockAuditLogSvc struct {
	logs []*audit_log.AuditLog
}

func (m *mockAuditLogSvc) Gets(_ context.Context, _ permission.AccessScope, _, _ int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogSvc) Log(_ context.Context, l *audit_log.AuditLog) error {
	m.logs = append(m.logs, l)
	return nil
}

func (m *mockAuditLogSvc) LogAsync(l *audit_log.AuditLog) {
	m.logs = append(m.logs, l)
}

func (m *mockAuditLogSvc) DeleteBefore(_ context.Context, _ permission.AccessScope, _ time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuditLogSvc) Wait() {}

// TestRefreshRejectsBlacklistedToken verifies that RefreshWithBlacklistCheck
// refuses to refresh a blacklisted (logged-out or rotated) token, so a revoked
// token cannot be exchanged for a fresh one.
func TestRefreshRejectsBlacklistedToken(t *testing.T) {
	// Not parallel: mutates package-level config.C.

	blacklistSvc := &mockTokenBlacklistSvc{
		blacklisted: make(map[string]bool),
	}

	testKey := "test-secret-key-for-jwt-blacklist"
	origKey := config.C.General.JwtKey
	origTimeout := config.C.General.JwtTimeoutHours
	origMaxRefresh := config.C.General.JwtMaxRefreshHours
	config.C.General.JwtKey = testKey
	config.C.General.JwtTimeoutHours = 1
	config.C.General.JwtMaxRefreshHours = 168
	defer func() {
		config.C.General.JwtKey = origKey
		config.C.General.JwtTimeoutHours = origTimeout
		config.C.General.JwtMaxRefreshHours = origMaxRefresh
	}()

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "test",
		Key:         []byte(testKey),
		Timeout:     1 * time.Hour,
		MaxRefresh:  168 * time.Hour,
		IdentityKey: "id",
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			return jwt.MapClaims{"id": float64(1)}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			return map[string]interface{}{"id": float64(1)}
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			return true
		},
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})
	if err != nil {
		t.Fatalf("Failed to create JWT middleware: %v", err)
	}

	// Generate a valid token
	token, _, err := authMiddleware.TokenGenerator(jwt.MapClaims{"id": float64(1)})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Add token to blacklist
	blacklistSvc.blacklisted[token] = true

	// Setup JwtMiddleware with RefreshWithBlacklistCheck
	jwtMw := &JwtMiddleware{
		AuthMiddleware:        authMiddleware,
		TokenBlacklistService: blacklistSvc,
	}

	router := gin.New()
	router.POST("/refresh", jwtMw.RefreshWithBlacklistCheck)

	// Attempt to refresh with blacklisted token
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("blacklisted token refresh returned status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestJWTUnauthorizedMapsKnownFailuresToSpecificMessages verifies that the
// Unauthorized callback maps known login failures (disabled, locked, too many
// attempts, disabled method) to their own i18n keys and statuses instead of a
// generic "unauthorized", so users learn why sign-in failed.
func TestJWTUnauthorizedMapsKnownFailuresToSpecificMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		errorMessage string
		expectKey    string
		expectStatus int
	}{
		{
			name:         "disabled account should return specific message",
			errorMessage: "user account is disabled",
			expectKey:    response.MsgUserDisabled,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "locked account should return specific message",
			errorMessage: "user account is locked",
			expectKey:    response.MsgUserLocked,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "too many attempts should return specific message",
			errorMessage: "too many login attempts, please try again later",
			expectKey:    response.MsgTooManyLoginAttempts,
			expectStatus: http.StatusTooManyRequests,
		},
		{
			name:         "disabled login method should return specific message",
			errorMessage: "login method is disabled",
			expectKey:    response.MsgLoginMethodDisabled,
			expectStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/login", nil)

			code := http.StatusUnauthorized
			message := tt.errorMessage
			msgKey := jwtUnauthorizedMessageKey(message)
			statusCode := jwtUnauthorizedStatus(code, msgKey)

			c.JSON(statusCode, gin.H{
				"code":    statusCode,
				"data":    nil,
				"message": msgKey,
			})

			body := w.Body.String()
			if !stringContains(body, tt.expectKey) {
				t.Errorf("response lacks specific message key %q. Body: %s",
					tt.expectKey, body)
			}
			if stringContains(body, response.MsgUnauthorized) {
				t.Errorf("response uses generic MsgUnauthorized for a known failure. Body: %s", body)
			}
			if w.Code != tt.expectStatus {
				t.Errorf("Unexpected status: got %d, want %d. Body: %s", w.Code, tt.expectStatus, body)
			}
		})
	}
}

func TestJWTUnauthorizedStatus_UnknownErrorUsesDefaultCode(t *testing.T) {
	t.Parallel()

	msgKey := jwtUnauthorizedMessageKey("auth failed")
	if msgKey != response.MsgUnauthorized {
		t.Fatalf("msgKey = %q, want %q", msgKey, response.MsgUnauthorized)
	}

	statusCode := jwtUnauthorizedStatus(http.StatusUnauthorized, msgKey)
	if statusCode != http.StatusUnauthorized {
		t.Fatalf("statusCode = %d, want %d", statusCode, http.StatusUnauthorized)
	}
}

// TestJWTRejectsTokenInQueryParameter verifies that with TokenLookup
// "header: Authorization, cookie: jwt" a token passed only as a query parameter
// is rejected: URLs end up in access logs, proxies and browser history.
func TestJWTRejectsTokenInQueryParameter(t *testing.T) {
	// Not parallel: mutates package-level config.C.

	testKey := "test-secret-key-for-jwt-query"
	origKey := config.C.General.JwtKey
	origTimeout := config.C.General.JwtTimeoutHours
	origMaxRefresh := config.C.General.JwtMaxRefreshHours
	config.C.General.JwtKey = testKey
	config.C.General.JwtTimeoutHours = 1
	config.C.General.JwtMaxRefreshHours = 168
	defer func() {
		config.C.General.JwtKey = origKey
		config.C.General.JwtTimeoutHours = origTimeout
		config.C.General.JwtMaxRefreshHours = origMaxRefresh
	}()

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "test",
		Key:         []byte(testKey),
		Timeout:     1 * time.Hour,
		MaxRefresh:  168 * time.Hour,
		IdentityKey: "id",
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			return jwt.MapClaims{"id": float64(1)}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			return map[string]interface{}{"id": float64(1)}
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			return true
		},
		// Header and cookie only; never "query: token"
		TokenLookup:   "header: Authorization, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})
	if err != nil {
		t.Fatalf("Failed to create JWT middleware: %v", err)
	}

	// Generate a valid token
	token, _, err := authMiddleware.TokenGenerator(jwt.MapClaims{"id": float64(1)})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Setup router with auth middleware
	router := gin.New()
	router.GET("/protected", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send request with token ONLY in query parameter
	req := httptest.NewRequest(http.MethodGet, "/protected?token="+token, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("token in query parameter returned status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestAuditLogUsesTrustedClientIP verifies that logAudit records c.ClientIP(),
// which honours the trusted proxy configuration, rather than the raw
// X-Forwarded-For header, so a client cannot forge the IP in audit logs.
func TestAuditLogUsesTrustedClientIP(t *testing.T) {
	t.Parallel()

	mockAudit := &mockAuditLogSvc{}
	jwtMw := &JwtMiddleware{
		AuditLogService: mockAudit,
	}

	// Create a Gin engine with no trusted proxies to simulate proper production config
	engine := gin.New()
	engine.SetTrustedProxies(nil) // No trusted proxies — X-Forwarded-For is ignored

	// Use a route handler to get a proper context bound to the engine
	// (Gin v1.9.1 doesn't have CreateTestContextOnly)
	w := httptest.NewRecorder()
	var capturedCtx *gin.Context
	engine.POST("/test-audit", func(c *gin.Context) {
		capturedCtx = c
	})

	req := httptest.NewRequest(http.MethodPost, "/test-audit", nil)
	// Set a forged X-Forwarded-For header
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	// The actual client IP (from RemoteAddr)
	req.RemoteAddr = "192.168.1.100:12345"
	engine.ServeHTTP(w, req)

	if capturedCtx == nil {
		t.Fatal("Failed to capture gin context from route handler")
	}

	// Call logAudit - this exercises the actual middleware code
	jwtMw.logAudit(capturedCtx, audit_log.AuditLogTypeLogout, "testuser", "test", true)

	// Check what IP was recorded
	if len(mockAudit.logs) == 0 {
		t.Fatal("No audit log was recorded")
	}

	recordedIP := mockAudit.logs[0].IpAddr
	// c.ClientIP() without trusted proxies returns RemoteAddr host part
	expectedIP := "192.168.1.100"

	// With no trusted proxies, c.ClientIP() ignores X-Forwarded-For
	if recordedIP != expectedIP {
		t.Errorf("audit log recorded IP %q, want %q (c.ClientIP(), ignoring the untrusted X-Forwarded-For)",
			recordedIP, expectedIP)
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
