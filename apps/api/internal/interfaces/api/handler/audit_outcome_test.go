package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
)

// mockGrantRBAC 只实现角色授权与职责分离约束相关的方法
type mockGrantRBAC struct {
	service.RBACService
	ensureErr error
	setCalled bool
	deleteErr error
}

func (m *mockGrantRBAC) EnsureCanSetRolePermissions(context.Context, string, string, []permission.PermissionGrant) error {
	return m.ensureErr
}

func (m *mockGrantRBAC) SetRolePermissions(context.Context, string, []permission.PermissionGrant) error {
	m.setCalled = true
	return nil
}

func (m *mockGrantRBAC) DeleteConstraint(context.Context, uint) error { return m.deleteErr }

// mockMenuDeleteService 只实现菜单删除
type mockMenuDeleteService struct {
	service.MenuService
	err error
}

func (m *mockMenuDeleteService) Delete(context.Context, uint) error { return m.err }

// 越权被拒同样要留痕：只记成功分支时，越权尝试在审计日志里不可见。
func TestSetRolePermissions_DeniedIsAudited(t *testing.T) {
	audit := &mockAuditLogService{}
	rbac := &mockGrantRBAC{ensureErr: apperror.ErrGrantExceedsCaller}
	h := NewAdminAuthorizationHandler(nil, nil, rbac, audit, nil)

	router := setupTestRouter()
	router.PUT("/roles/:role/permissions", h.SetRolePermissions)
	req := httptest.NewRequest(http.MethodPut, "/roles/ops/permissions", strings.NewReader(`{"grants":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK || rbac.setCalled {
		t.Fatalf("a denied grant must not be applied (status %d)", w.Code)
	}
	entry := audit.requireEntry(t, audit_log.AuditLogTypeRolePermissionsSet)
	if entry.Success || entry.Target != "ops" {
		t.Fatalf("expected a failed entry for role ops, got %+v", entry)
	}
	if !strings.Contains(entry.Details, apperror.ErrGrantExceedsCaller.Error()) {
		t.Errorf("a known business error is recorded as the failure reason, got %q", entry.Details)
	}
}

// 未登记的错误可能带 SQL、Redis 等内部细节，只记为 internal error。
func TestAuditFailureReasonHidesInternalErrors(t *testing.T) {
	audit := &mockAuditLogService{}
	rbac := &mockGrantRBAC{deleteErr: errors.New(`pq: relation "sod_constraints" violates foreign key; password=hunter2`)}
	h := NewAdminAuthorizationHandler(nil, nil, rbac, audit, nil)

	router := setupTestRouter()
	router.DELETE("/constraints/:id", h.DeleteConstraint)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/constraints/4", nil))

	entry := audit.requireEntry(t, audit_log.AuditLogTypeSoDDelete)
	if entry.Success {
		t.Fatal("expected a failed entry")
	}
	if strings.Contains(entry.Details, "pq:") || strings.Contains(entry.Details, "hunter2") || !strings.Contains(entry.Details, "internal error") {
		t.Fatalf("raw error text must not reach the audit log, got %q", entry.Details)
	}
}

// 菜单与职责分离约束有自己的审计类型，不再记成"新增/删除权限"。
func TestMenuAndConstraintAuditTypes(t *testing.T) {
	audit := &mockAuditLogService{}
	menus := NewMenuHandler(&mockMenuDeleteService{}, nil, audit)
	auth := NewAdminAuthorizationHandler(nil, nil, &mockGrantRBAC{}, audit, nil)

	router := setupTestRouter()
	router.DELETE("/menus/:id", menus.Delete)
	router.DELETE("/constraints/:id", auth.DeleteConstraint)
	for _, path := range []string{"/menus/7", "/constraints/4"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, w.Code)
		}
	}
	if entry := audit.requireEntry(t, audit_log.AuditLogTypeMenuDelete); !entry.Success || entry.Target != "7" {
		t.Errorf("menu delete entry = %+v", entry)
	}
	if entry := audit.requireEntry(t, audit_log.AuditLogTypeSoDDelete); !entry.Success || entry.Target != "4" {
		t.Errorf("constraint delete entry = %+v", entry)
	}
}
