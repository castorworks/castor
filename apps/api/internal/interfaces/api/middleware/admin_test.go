package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/gin-gonic/gin"
)

type recordingRBAC struct {
	service.RBACService
	object string
	action string
}

func (r *recordingRBAC) AuthorizeSession(_ context.Context, _ string, _ uint, object, action string) (bool, error) {
	r.object = object
	r.action = action
	return true, nil
}

func TestAdminRoleMiddlewareUsesGinRouteTemplate(t *testing.T) {
	recorder := &recordingRBAC{}
	middleware := NewAdminRoleMiddleware(recorder)
	router := gin.New()
	router.GET("/api/v1/admin/users/:id", func(c *gin.Context) {
		c.Set(constant.JWT_IDENTITY_KEY, &user.User{ID: 7, AuthorizationSessionID: "session-1"})
		c.Next()
	}, middleware.MiddlewareFunc, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/42", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if recorder.object != "/api/v1/admin/users/:id" || recorder.action != http.MethodGet {
		t.Fatalf("authorization received %q %q", recorder.object, recorder.action)
	}
}
