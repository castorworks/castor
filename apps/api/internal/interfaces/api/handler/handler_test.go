package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// ============================================
// Mock implementations
// ============================================

// mockUserService 模拟用户服务
type mockUserService struct {
	getResp *dto.UserResp
	getErr  error
	postErr error
	delErr  error
}

func (m *mockUserService) Gets(_ context.Context, _ permission.AccessScope, _, _ int, _ string, _ ...query.Option) ([]dto.UserResp, int64, error) {
	if m.getResp != nil {
		return []dto.UserResp{*m.getResp}, 1, nil
	}
	return []dto.UserResp{}, 0, nil
}

func (m *mockUserService) Get(_ context.Context, _ permission.AccessScope, _ uint) (*dto.UserResp, error) {
	return m.getResp, m.getErr
}

func (m *mockUserService) GetByUsername(_ context.Context, _ string) (*dto.UserResp, error) {
	return m.getResp, m.getErr
}

func (m *mockUserService) GetRawByUsername(_ context.Context, _ string) (*user.User, error) {
	if m.getResp != nil {
		return &user.User{Username: m.getResp.Username}, nil
	}
	return nil, m.getErr
}

func (m *mockUserService) Post(_ context.Context, _ permission.AccessScope, _ *dto.UserPostReq) (*dto.UserResp, error) {
	if m.postErr != nil {
		return nil, m.postErr
	}
	return m.getResp, nil
}

func (m *mockUserService) Put(_ context.Context, _ permission.AccessScope, _ uint, _ *dto.UserPutReq) (*dto.UserResp, error) {
	return m.getResp, m.getErr
}

func (m *mockUserService) CreateFederated(_ context.Context, _ *user.User) error { return nil }

func (m *mockUserService) Import(_ context.Context, _ permission.AccessScope, _ *dto.UserImportReq) (*dto.UserImportResult, error) {
	return nil, nil
}

func (m *mockUserService) Delete(_ context.Context, _ permission.AccessScope, _ uint) error {
	return m.delErr
}

// mockAuditLogService 模拟审计日志服务，并记录写入的条目供断言使用
type mockAuditLogService struct {
	mu           sync.Mutex
	entries      []audit_log.AuditLog
	deletedCount int64
	deleteErr    error
}

func (m *mockAuditLogService) Gets(_ context.Context, _ permission.AccessScope, _, _ int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogService) Log(_ context.Context, item *audit_log.AuditLog) error {
	m.record(item)
	return nil
}

func (m *mockAuditLogService) LogAsync(item *audit_log.AuditLog) {
	m.record(item)
}

func (m *mockAuditLogService) DeleteBefore(_ context.Context, _ permission.AccessScope, _ time.Time) (int64, error) {
	return m.deletedCount, m.deleteErr
}

func (m *mockAuditLogService) Wait() {}

func (m *mockAuditLogService) record(item *audit_log.AuditLog) {
	if item == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, *item)
}

// records 返回已记录的审计条目副本
func (m *mockAuditLogService) records() []audit_log.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]audit_log.AuditLog(nil), m.entries...)
}

// requireEntry 断言恰好记录了一条指定类型的审计条目，并返回它
func (m *mockAuditLogService) requireEntry(t *testing.T, logType audit_log.AuditLogType) audit_log.AuditLog {
	t.Helper()
	matched := make([]audit_log.AuditLog, 0, 1)
	for _, entry := range m.records() {
		if entry.LogType == logType {
			matched = append(matched, entry)
		}
	}
	if len(matched) != 1 {
		t.Fatalf("expected exactly 1 audit entry of type %s, got %d (all: %+v)", logType, len(matched), m.records())
	}
	return matched[0]
}

// ============================================
// Test helpers
// ============================================

// setupTestRouter 创建测试用的 Gin 引擎（包含 i18n 中间件）
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 使用内嵌的 i18n 配置，避免依赖文件系统
	r.Use(ginI18n.Localize(
		ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         "../../../../configs/i18n",
			AcceptLanguage:   []language.Tag{language.Chinese, language.English},
			DefaultLanguage:  language.Chinese,
			UnmarshalFunc:    toml.Unmarshal,
			FormatBundleFile: "toml",
		}),
	))
	return r
}

// parseResponse 解析 JSON 响应
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v\nBody: %s", err, w.Body.String())
	}
	return resp
}

// ============================================
// Tests - AdminUserHandler
// ============================================

func TestAdminUserHandler_Get_Success(t *testing.T) {
	userSvc := &mockUserService{
		getResp: &dto.UserResp{
			BaseModel: shared.BaseModel{ID: 1},
			Username:  "testuser",
			Name:      "Test User",
		},
	}

	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     userSvc,
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.GET("/admin/users/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data field in response")
	}
	if data["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got '%v'", data["username"])
	}
}

func TestAdminUserHandler_Get_NotFound(t *testing.T) {
	userSvc := &mockUserService{
		getErr: shared.ErrNotFound,
	}

	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     userSvc,
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.GET("/admin/users/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAdminUserHandler_Get_InvalidID(t *testing.T) {
	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     &mockUserService{},
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.GET("/admin/users/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminUserHandler_Delete_Success(t *testing.T) {
	userSvc := &mockUserService{
		delErr: nil,
	}

	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     userSvc,
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.DELETE("/admin/users/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminUserHandler_Delete_NotFound(t *testing.T) {
	userSvc := &mockUserService{
		delErr: shared.ErrNotFound,
	}

	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     userSvc,
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.DELETE("/admin/users/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAdminUserHandler_Post_InvalidJSON(t *testing.T) {
	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     &mockUserService{},
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.POST("/admin/users", h.Post)

	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminUserHandler_Post_EmptyUsername(t *testing.T) {
	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     &mockUserService{},
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.POST("/admin/users", h.Post)

	body := `{"username":"","password":"password123","name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminUserHandler_Post_ServiceError(t *testing.T) {
	userSvc := &mockUserService{
		postErr: errors.New("password length must be at least 8 characters"),
	}

	h := &AdminUserHandler{
		RBAC:            allScopeRBAC{},
		UserService:     userSvc,
		AuditLogService: &mockAuditLogService{},
	}

	router := setupTestRouter()
	router.POST("/admin/users", h.Post)

	body := `{"username":"newuser","password":"short1","name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}
}

// allScopeRBAC 让处理器测试以"全部数据"范围运行：数据范围本身由服务层测试覆盖。
type allScopeRBAC struct{ service.RBACService }

func (allScopeRBAC) SessionScope(context.Context, string, uint) (permission.AccessScope, error) {
	return permission.AccessScope{All: true}, nil
}
