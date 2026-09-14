package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
)

type orderRecordingAssetService struct {
	service.AssetService
	calls int
	order string
}

func (s *orderRecordingAssetService) GetPublicAssets(_ context.Context, _, _ int, order string) ([]dto.AssetResp, int64, error) {
	s.calls++
	s.order = order
	return nil, 0, nil
}

var sqlInjectionOrders = []string{
	"created_at DESC; DROP TABLE users--",
	"(CASE WHEN (SELECT substr(password,1,1) FROM users LIMIT 1)='a' THEN size ELSE id END)",
	"id desc, (SELECT pg_sleep(5))",
	"created_at/**/DESC",
	"created_at DESC NULLS FIRST",
	"unknown_column desc",
}

func TestAssetHandler_GetPublicRejectsInjectedOrder(t *testing.T) {
	svc := &orderRecordingAssetService{}
	h := NewAssetHandler(svc)
	r := setupTestRouter()
	r.GET("/assets/public", h.GetPublic)

	for _, payload := range sqlInjectionOrders {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/public?order="+url.QueryEscape(payload), nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("order %q: status = %d, want 400", payload, w.Code)
		}
	}
	if svc.calls != 0 {
		t.Fatalf("service must not be called with an invalid order, got %d calls", svc.calls)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/public?order="+url.QueryEscape("size desc"), nil))
	if w.Code != http.StatusOK || svc.order != "size DESC" {
		t.Fatalf("valid order: status = %d, order = %q", w.Code, svc.order)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/public", nil))
	if w.Code != http.StatusOK || svc.order != "created_at DESC" {
		t.Fatalf("default order: status = %d, order = %q", w.Code, svc.order)
	}
}

func TestGenericGetsExcludesSensitiveFields(t *testing.T) {
	allowed := getAllowedFields(reflect.TypeOf(user.User{}))
	for _, column := range []string{"password", "authorization_session_id"} {
		if allowed[column] {
			t.Errorf("%s must not be filterable or sortable", column)
		}
	}
	if !allowed["username"] || !allowed["id"] {
		t.Fatalf("regular columns must stay allowed: %v", allowed)
	}

	var gotOpts []query.Option
	var called bool
	r := setupTestRouter()
	r.GET("/users", func(c *gin.Context) {
		GenericGets(c, reflect.TypeOf(user.User{}), func(_ *gin.Context, _, _ int, _ string, opts ...query.Option) (interface{}, int64, error) {
			called = true
			gotOpts = opts
			return []any{}, 0, nil
		})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?password-like=%242a", nil))
	if w.Code != http.StatusOK || len(gotOpts) != 0 {
		t.Fatalf("password filter must be ignored: status=%d opts=%v", w.Code, gotOpts)
	}

	called = false
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?searchText=x&searchFields=password", nil))
	if w.Code != http.StatusBadRequest || called {
		t.Fatalf("password search field must be rejected: status=%d called=%v", w.Code, called)
	}

	for _, payload := range append([]string{"password desc"}, sqlInjectionOrders...) {
		called = false
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?order="+url.QueryEscape(payload), nil))
		if w.Code != http.StatusBadRequest || called {
			t.Errorf("order %q: status=%d called=%v, want 400 without service call", payload, w.Code, called)
		}
	}
}
