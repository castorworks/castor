package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/infrastructure/config"
)

type sessionRBACStub struct {
	service.RBACService
	validSession string
}

func (s *sessionRBACStub) GetSessionAccess(_ context.Context, sessionID string, _ uint) (*service.AccessSnapshot, error) {
	if sessionID != s.validSession {
		return nil, apperror.ErrAuthorizationSessionRevoked
	}
	return &service.AccessSnapshot{SessionID: sessionID}, nil
}

func TestRefreshRetiresPreviousToken(t *testing.T) {
	// Not parallel: NewJwtMiddleware reads package-level config.C.
	orig := config.C.General
	config.C.General.JwtKey = "refresh-test-key-0123456789abcdef"
	config.C.General.JwtTimeoutHours = 2
	config.C.General.JwtMaxRefreshHours = 168
	config.C.General.Development = true
	t.Cleanup(func() { config.C.General = orig })

	blacklist := &mockTokenBlacklistSvc{blacklisted: map[string]bool{}}
	rbac := &sessionRBACStub{validSession: "session-1"}
	mw, err := NewJwtMiddleware(nil, rbac, blacklist, &mockAuditLogSvc{})
	if err != nil {
		t.Fatalf("NewJwtMiddleware() error = %v", err)
	}
	identity := &user.User{ID: 42, Name: "Alice", Username: "alice", AccountSource: "internal", AuthorizationSessionID: "session-1"}
	oldToken, _, err := mw.AuthMiddleware.TokenGenerator(identity)
	if err != nil {
		t.Fatal(err)
	}

	router := newI18nTestRouter()
	router.POST("/refresh", mw.RefreshWithBlacklistCheck)
	refresh := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	w := refresh(oldToken)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", w.Code, w.Body.String())
	}
	if !blacklist.blacklisted[oldToken] {
		t.Fatal("previous token must be blacklisted after refresh")
	}
	var newToken string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "jwt" {
			newToken = cookie.Value
		}
	}
	if newToken == "" || newToken == oldToken {
		t.Fatalf("refresh must issue a distinct token, got %q", newToken)
	}
	parsed, err := mw.AuthMiddleware.ParseTokenString(newToken)
	if err != nil {
		t.Fatalf("new token invalid: %v", err)
	}
	claims := jwt.ExtractClaimsFromToken(parsed)
	if claims["jti"] == "" || claims["username"] == nil {
		t.Fatalf("new token lost claims: %v", claims)
	}

	if w := refresh(oldToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("replayed token refresh status = %d, want 401", w.Code)
	}

	rbac.validSession = "other"
	if w := refresh(newToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session refresh status = %d, want 401", w.Code)
	}
	if blacklist.blacklisted[newToken] {
		t.Fatal("a rejected refresh must not blacklist the presented token")
	}
}

func TestRefreshableUntil(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	claims := jwt.MapClaims{"exp": float64(now.Add(2 * time.Hour).Unix()), "orig_iat": float64(now.Unix())}
	if got, want := refreshableUntil(claims, 168*time.Hour), now.Add(168*time.Hour); !got.Equal(want) {
		t.Fatalf("refreshableUntil = %v, want %v", got, want)
	}
	if got, want := refreshableUntil(claims, time.Hour), now.Add(2*time.Hour); !got.Equal(want) {
		t.Fatalf("refreshableUntil = %v, want %v", got, want)
	}
}
