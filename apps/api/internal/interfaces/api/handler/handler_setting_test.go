package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
)

// ============================================
// Mock setting service for handler tests
// ============================================

type mockSettingService struct {
	getsResp       []dto.SettingResp
	getResp        *dto.SettingResp
	getByKeyResp   *dto.SettingResp
	getErr         error
	postResp       *dto.SettingResp
	postErr        error
	deleteErr      error
	publicSettings map[string]interface{}
}

func (m *mockSettingService) Gets(_ context.Context, _ string) ([]dto.SettingResp, error) {
	return m.getsResp, m.getErr
}

func (m *mockSettingService) Get(_ context.Context, _ uint) (*dto.SettingResp, error) {
	return m.getResp, m.getErr
}

func (m *mockSettingService) GetByKey(_ context.Context, _ string) (*dto.SettingResp, error) {
	return m.getByKeyResp, m.getErr
}

func (m *mockSettingService) Post(_ context.Context, req *dto.SettingPostReq) (*dto.SettingResp, error) {
	if m.postErr != nil {
		return nil, m.postErr
	}
	if m.postResp != nil {
		return m.postResp, nil
	}
	return &dto.SettingResp{Key: req.Key, Value: req.Value}, nil
}

func (m *mockSettingService) Put(_ context.Context, _ uint, req *dto.SettingPutReq) (*dto.SettingResp, error) {
	resp := &dto.SettingResp{}
	if req.Value != nil {
		resp.Value = *req.Value
	}
	return resp, m.getErr
}

func (m *mockSettingService) Delete(_ context.Context, _ uint) error {
	return m.deleteErr
}

func (m *mockSettingService) BatchUpdate(_ context.Context, _ *dto.SettingBatchUpdateReq) error {
	return m.getErr
}

func (m *mockSettingService) GetPublicSettings(_ context.Context) (map[string]interface{}, error) {
	return m.publicSettings, m.getErr
}

func (m *mockSettingService) GetValue(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockSettingService) GetBool(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockSettingService) GetInt(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// ============================================
// Tests - AdminSettingHandler
// ============================================

func TestAdminSettingHandler_Gets(t *testing.T) {
	t.Run("returns settings list", func(t *testing.T) {
		svc := &mockSettingService{
			getsResp: []dto.SettingResp{
				{Key: "site.name", Value: "MySite"},
				{Key: "site.logo", Value: "/logo.png"},
			},
		}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings", h.Gets)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatal("expected data array in response")
		}
		if len(data) != 2 {
			t.Errorf("expected 2 settings, got %d", len(data))
		}
	})

	t.Run("passes category query param", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings", h.Gets)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings?category=GENERAL", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestAdminSettingHandler_Get(t *testing.T) {
	t.Run("returns setting by ID", func(t *testing.T) {
		svc := &mockSettingService{
			getResp: &dto.SettingResp{Key: "test.key", Value: "test-value"},
		}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings/:id", h.Get)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("invalid ID returns bad request", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings/:id", h.Get)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestAdminSettingHandler_GetByKey(t *testing.T) {
	t.Run("returns setting by key", func(t *testing.T) {
		svc := &mockSettingService{
			getByKeyResp: &dto.SettingResp{Key: "site.name", Value: "MySite"},
		}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings/key/:key", h.GetByKey)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/key/site.name", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("empty key returns bad request", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.GET("/admin/settings/key/:key", h.GetByKey)

		req := httptest.NewRequest(http.MethodGet, "/admin/settings/key/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestAdminSettingHandler_Post(t *testing.T) {
	t.Run("creates setting", func(t *testing.T) {
		svc := &mockSettingService{
			postResp: &dto.SettingResp{Key: "new.key", Value: "new-value"},
		}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.POST("/admin/settings", h.Post)

		body := `{"key":"new.key","value":"new-value","name":"New Setting"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("empty key returns bad request", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.POST("/admin/settings", h.Post)

		body := `{"key":"","value":"v"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestAdminSettingHandler_Delete(t *testing.T) {
	t.Run("deletes setting", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.DELETE("/admin/settings/:id", h.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/admin/settings/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestAdminSettingHandler_BatchUpdate(t *testing.T) {
	t.Run("batch updates settings", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.PUT("/admin/settings/batch", h.BatchUpdate)

		body := `{"settings":[{"key":"k1","value":"v1"},{"key":"k2","value":"v2"}]}`
		req := httptest.NewRequest(http.MethodPut, "/admin/settings/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("invalid JSON returns bad request", func(t *testing.T) {
		svc := &mockSettingService{}
		h := NewAdminSettingHandler(svc, &mockAuditLogService{})

		router := setupTestRouter()
		router.PUT("/admin/settings/batch", h.BatchUpdate)

		req := httptest.NewRequest(http.MethodPut, "/admin/settings/batch", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// ============================================
// Tests - SettingHandler (public)
// ============================================

func TestSettingHandler_GetPublicSettings(t *testing.T) {
	t.Run("returns public settings", func(t *testing.T) {
		svc := &mockSettingService{
			publicSettings: map[string]interface{}{
				"site.name":  "MySite",
				"feature.on": true,
			},
		}
		h := NewSettingHandler(svc)

		router := setupTestRouter()
		router.GET("/settings/public", h.GetPublicSettings)

		req := httptest.NewRequest(http.MethodGet, "/settings/public", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data object in response")
		}
		if data["site.name"] != "MySite" {
			t.Errorf("expected site.name='MySite', got %v", data["site.name"])
		}
	})
}
