package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
)

// ============================================
// Mocks（沿用仓库的手写 mock 约定：嵌入接口，只覆盖被测方法）
// ============================================

// mockAssetStatusService 只实现审计测试需要的资产状态更新
type mockAssetStatusService struct {
	service.AssetService
	err error
}

func (m *mockAssetStatusService) UpdateStatus(_ context.Context, _ uint, _ asset.AssetStatus) error {
	return m.err
}

// mockDictItemService 只实现审计测试需要的字典项删除
type mockDictItemService struct {
	service.DictionaryService
	err error
}

func (m *mockDictItemService) DeleteItem(_ context.Context, _ uint) error {
	return m.err
}

// mockLoginHistoryDeleteService 只实现审计测试需要的登录历史删除
type mockLoginHistoryDeleteService struct {
	service.LoginHistoryService
	err error
}

func (m *mockLoginHistoryDeleteService) Delete(_ context.Context, _ permission.AccessScope, _ uint) error {
	return m.err
}

// mockActiveRolesService 只实现审计测试需要的激活角色切换
type mockActiveRolesService struct {
	service.RBACService
	err error
}

func (m *mockActiveRolesService) SetActiveRoles(_ context.Context, _ string, _ uint, roleCodes []string) (*service.AccessSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &service.AccessSnapshot{Username: "tester"}, nil
}

// ============================================
// Tests - 变更类接口必须留下审计
// ============================================

func TestAdminSettingHandler_PutRecordsAudit(t *testing.T) {
	// 配置值可能是凭据，审计必须记录配置键而不是取值。
	const secretValue = "super-secret-smtp-password"

	t.Run("success is audited with the key only", func(t *testing.T) {
		svc := &mockSettingService{getResp: &dto.SettingResp{Key: "mail.smtp.password"}}
		audit := &mockAuditLogService{}
		h := NewAdminSettingHandler(svc, audit)

		router := setupTestRouter()
		router.PUT("/admin/settings/:id", h.Put)

		body := `{"value":"` + secretValue + `"}`
		req := httptest.NewRequest(http.MethodPut, "/admin/settings/7", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		entry := audit.requireEntry(t, audit_log.AuditLogTypeSettingUpdate)
		if !entry.Success {
			t.Error("expected the audit entry to be marked successful")
		}
		if entry.Target != "mail.smtp.password" {
			t.Errorf("expected target to be the setting key, got %q", entry.Target)
		}
		if strings.Contains(entry.Details, secretValue) {
			t.Fatalf("audit details must never contain the setting value, got %q", entry.Details)
		}
	})

	t.Run("failure is audited as unsuccessful", func(t *testing.T) {
		svc := &mockSettingService{getErr: errors.New("boom")}
		audit := &mockAuditLogService{}
		h := NewAdminSettingHandler(svc, audit)

		router := setupTestRouter()
		router.PUT("/admin/settings/:id", h.Put)

		req := httptest.NewRequest(http.MethodPut, "/admin/settings/7", strings.NewReader(`{"value":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		entry := audit.requireEntry(t, audit_log.AuditLogTypeSettingUpdate)
		if entry.Success {
			t.Error("expected the audit entry to be marked failed")
		}
	})
}

func TestAdminSettingHandler_BatchUpdateRecordsAudit(t *testing.T) {
	// 登录方式、验证码开关等都通过批量接口改动，是影响最大的一条路径。
	svc := &mockSettingService{}
	audit := &mockAuditLogService{}
	h := NewAdminSettingHandler(svc, audit)

	router := setupTestRouter()
	router.PUT("/admin/settings/batch", h.BatchUpdate)

	body := `{"settings":[{"key":"auth.login.methods","value":"password"},{"key":"auth.captcha.enabled","value":"true"}]}`
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeSettingUpdate)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	for _, key := range []string{"auth.login.methods", "auth.captcha.enabled"} {
		if !strings.Contains(entry.Details, key) {
			t.Errorf("expected details to name changed key %q, got %q", key, entry.Details)
		}
	}
}

func TestAdminDictionaryHandler_DeleteItemRecordsAudit(t *testing.T) {
	audit := &mockAuditLogService{}
	h := NewAdminDictionaryHandler(&mockDictItemService{}, audit)

	router := setupTestRouter()
	router.DELETE("/admin/dict-items/:id", h.DeleteItem)

	req := httptest.NewRequest(http.MethodDelete, "/admin/dict-items/12", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeDictItemDelete)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	if entry.Target != "12" {
		t.Errorf("expected target %q, got %q", "12", entry.Target)
	}
}

func TestAdminAssetHandler_UpdateStatusRecordsAudit(t *testing.T) {
	audit := &mockAuditLogService{}
	h := NewAdminAssetHandler(&mockAssetStatusService{}, nil, audit)

	router := setupTestRouter()
	router.PUT("/admin/assets/:id/status", h.UpdateStatus)

	req := httptest.NewRequest(http.MethodPut, "/admin/assets/3/status", strings.NewReader(`{"status":"ARCHIVED"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeAssetUpdate)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	if entry.Target != "3" {
		t.Errorf("expected target %q, got %q", "3", entry.Target)
	}
	if !strings.Contains(entry.Details, "ARCHIVED") {
		t.Errorf("expected details to name the new status, got %q", entry.Details)
	}
}

func TestAdminAssetHandler_UpdateStatusFailureIsAudited(t *testing.T) {
	audit := &mockAuditLogService{}
	h := NewAdminAssetHandler(&mockAssetStatusService{err: errors.New("boom")}, nil, audit)

	router := setupTestRouter()
	router.PUT("/admin/assets/:id/status", h.UpdateStatus)

	req := httptest.NewRequest(http.MethodPut, "/admin/assets/3/status", strings.NewReader(`{"status":"ARCHIVED"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	entry := audit.requireEntry(t, audit_log.AuditLogTypeAssetUpdate)
	if entry.Success {
		t.Error("expected the audit entry to be marked failed")
	}
}

func TestAccountPermissionHandler_PutActiveRolesRecordsAudit(t *testing.T) {
	audit := &mockAuditLogService{}
	h := NewAccountPermissionHandler(&mockActiveRolesService{}, audit)

	router := setupTestRouter()
	router.PUT("/account/session/roles", h.PutActiveRoles)

	req := httptest.NewRequest(http.MethodPut, "/account/session/roles", strings.NewReader(`{"roleCodes":["admin","auditor"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeActiveRolesChange)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	if !strings.Contains(entry.Details, "admin") || !strings.Contains(entry.Details, "auditor") {
		t.Errorf("expected details to name the activated roles, got %q", entry.Details)
	}
}

func TestAdminAuditLogHandler_DeleteBeforeRecordsAudit(t *testing.T) {
	// 审计清理会抹掉证据链，清理动作自身必须留痕。
	audit := &mockAuditLogService{deletedCount: 42}
	h := NewAdminAuditLogHandler(audit, allScopeRBAC{}, nil)

	router := setupTestRouter()
	router.POST("/admin/audit-logs/cleanup", h.DeleteBefore)

	req := httptest.NewRequest(http.MethodPost, "/admin/audit-logs/cleanup", strings.NewReader(`{"retentionDays":30}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeAuditLogCleanup)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	if !strings.Contains(entry.Details, "30") || !strings.Contains(entry.Details, "42") {
		t.Errorf("expected details to carry retention days and deleted count, got %q", entry.Details)
	}
}

func TestAdminLoginHistoryHandler_DeleteRecordsAudit(t *testing.T) {
	audit := &mockAuditLogService{}
	h := NewAdminLoginHistoryHandler(&mockLoginHistoryDeleteService{}, audit, allScopeRBAC{}, nil)

	router := setupTestRouter()
	router.DELETE("/admin/login-histories/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/login-histories/9", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	entry := audit.requireEntry(t, audit_log.AuditLogTypeLoginHistoryDelete)
	if !entry.Success {
		t.Error("expected the audit entry to be marked successful")
	}
	if entry.Target != "9" {
		t.Errorf("expected target %q, got %q", "9", entry.Target)
	}
}
