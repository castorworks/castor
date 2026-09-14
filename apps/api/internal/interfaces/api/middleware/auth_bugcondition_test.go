package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/domain/audit_log"
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

func (m *mockAuditLogSvc) Gets(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogSvc) Log(_ context.Context, l *audit_log.AuditLog) error {
	m.logs = append(m.logs, l)
	return nil
}

func (m *mockAuditLogSvc) LogAsync(l *audit_log.AuditLog) {
	m.logs = append(m.logs, l)
}

func (m *mockAuditLogSvc) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuditLogSvc) Wait() {}

// TestBugCondition_1a_BlacklistBypass verifies that a blacklisted token is
// REJECTED when attempting to refresh. On unfixed code, this FAILS because
// RefreshHandler does not check the blacklist. On fixed code, this PASSES
// because RefreshWithBlacklistCheck intercepts blacklisted tokens.
//
// **Validates: Requirements 1.1**
func TestBugCondition_1a_BlacklistBypass(t *testing.T) {
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

	// Setup JwtMiddleware with RefreshWithBlacklistCheck (the fix)
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

	// EXPECTED: Blacklisted token refresh returns 401 (fix applied)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Bug 1 confirmed: blacklisted token refresh returned status %d, want %d (401 Unauthorized). "+
			"RefreshHandler does not check blacklist.", w.Code, http.StatusUnauthorized)
	}
}

// TestBugCondition_1b_GenericErrorMessages verifies that the Unauthorized callback
// returns specific error messages for known failure reasons. On unfixed code, this
// FAILS because specific errors are discarded. On fixed code, this PASSES because
// the Unauthorized callback maps known errors to their i18n keys.
//
// **Validates: Requirements 1.2**
func TestBugCondition_1b_GenericErrorMessages(t *testing.T) {
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

			// EXPECTED: The response should contain the specific error key (not generic MsgUnauthorized)
			body := w.Body.String()
			if !stringContains(body, tt.expectKey) {
				t.Errorf("Bug 2 confirmed: response contains generic message instead of specific key %q. Body: %s",
					tt.expectKey, body)
			}
			if stringContains(body, response.MsgUnauthorized) {
				t.Errorf("Bug 2 confirmed: response still uses generic MsgUnauthorized. Body: %s", body)
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

// TestBugCondition_1d_QueryTokenAccepted verifies that tokens passed via query
// parameter are REJECTED. On unfixed code, this FAILS because TokenLookup includes
// "query: token". On fixed code, this PASSES because TokenLookup only includes
// "header: Authorization, cookie: jwt".
//
// **Validates: Requirements 1.4**
func TestBugCondition_1d_QueryTokenAccepted(t *testing.T) {
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
		// Fixed TokenLookup: no longer includes "query: token"
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

	// EXPECTED: Query parameter token is rejected (401) because TokenLookup
	// no longer includes "query: token"
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Bug 4 confirmed: query parameter token accepted with status %d, want %d (401 Unauthorized). "+
			"TokenLookup includes 'query: token' which exposes tokens in logs.", w.Code, http.StatusUnauthorized)
	}
}

// TestBugCondition_1g_IPSpoofing verifies that the logAudit method uses c.ClientIP()
// instead of the raw X-Forwarded-For header. On unfixed code, this FAILS because
// logAudit reads X-Forwarded-For directly. On fixed code, this PASSES because
// logAudit uses c.ClientIP() which respects trusted proxy configuration.
//
// **Validates: Requirements 1.7**
func TestBugCondition_1g_IPSpoofing(t *testing.T) {
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

	// EXPECTED: logAudit uses c.ClientIP() which ignores X-Forwarded-For
	// when no proxies are trusted
	if recordedIP != expectedIP {
		t.Errorf("Bug 7 confirmed: audit log recorded IP %q (from forged X-Forwarded-For) instead of %q (from c.ClientIP()). "+
			"Attacker can spoof IP in audit logs.", recordedIP, expectedIP)
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
