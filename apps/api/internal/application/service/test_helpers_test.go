package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
)

// ============================================
// Shared mock implementations for all test files
// ============================================

// --- mockUserRepo implements user.Repository ---

type mockUserRepo struct {
	users   map[string]*user.User
	created []*user.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*user.User)}
}

func (m *mockUserRepo) Gets(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]user.User, int64, error) {
	var result []user.User
	for _, u := range m.users {
		result = append(result, *u)
	}
	return result, int64(len(result)), nil
}

func (m *mockUserRepo) Get(_ context.Context, id uint) (*user.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*user.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserRepo) Create(_ context.Context, u *user.User) error {
	if _, ok := m.users[u.Username]; ok {
		return errors.New("duplicate username")
	}
	m.created = append(m.created, u)
	u.ID = uint(len(m.users) + 1)
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepo) Update(_ context.Context, u *user.User) error {
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id uint) error {
	for k, u := range m.users {
		if u.ID == id {
			delete(m.users, k)
			return nil
		}
	}
	return shared.ErrNotFound
}

// --- mockSettingRepo implements setting.Repository ---

type mockSettingRepo struct {
	settings map[string]*setting.Setting
}

func newMockSettingRepo() *mockSettingRepo {
	return &mockSettingRepo{settings: map[string]*setting.Setting{
		setting.KeyFeatureRegisterEnabled: {
			Key:   setting.KeyFeatureRegisterEnabled,
			Value: "true",
		},
		setting.KeySecurityPasswordMinLength: {
			Key:   setting.KeySecurityPasswordMinLength,
			Value: "6",
		},
	}}
}

func (m *mockSettingRepo) Gets(_ context.Context, _ string) ([]setting.Setting, error) {
	return nil, nil
}

func (m *mockSettingRepo) Get(_ context.Context, _ uint) (*setting.Setting, error) {
	return nil, shared.ErrNotFound
}

func (m *mockSettingRepo) GetByKey(_ context.Context, key string) (*setting.Setting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockSettingRepo) Create(_ context.Context, _ *setting.Setting) error { return nil }
func (m *mockSettingRepo) Update(_ context.Context, _ *setting.Setting) error { return nil }
func (m *mockSettingRepo) Delete(_ context.Context, _ uint) error             { return nil }
func (m *mockSettingRepo) GetByKeys(_ context.Context, _ []string) ([]setting.Setting, error) {
	return nil, nil
}
func (m *mockSettingRepo) BatchUpdate(_ context.Context, _ []setting.Setting) error { return nil }
func (m *mockSettingRepo) GetPublicSettings(_ context.Context) ([]setting.Setting, error) {
	return nil, nil
}
func (m *mockSettingRepo) GetByCategory(_ context.Context, _ setting.SettingCategory) ([]setting.Setting, error) {
	return nil, nil
}
func (m *mockSettingRepo) ExistsByKey(_ context.Context, _ string) (bool, error) { return false, nil }

// --- mockCaptchaClient implements captchaProvider ---

type mockCaptchaClient struct {
	valid bool
}

func (m *mockCaptchaClient) VerifyCaptcha(_, _ string) bool {
	return m.valid
}

func (m *mockCaptchaClient) GetCaptcha() (string, string, error) {
	return "captcha-id", "captcha-image", nil
}

// --- mockRateLimiter implements requestRateLimiter ---

type mockRateLimiter struct {
	blocked bool
	err     error
}

func (m *mockRateLimiter) IsRequestReachLimit(_ context.Context, _ string, _ time.Duration, _ int64) (bool, error) {
	return m.blocked, m.err
}

// --- mockVerifyCodeClient implements VerificationCodeStore ---

type mockVerifyCodeClient struct {
	valid bool
}

func (m *mockVerifyCodeClient) Generate(_ context.Context, _, _ string) (string, error) {
	return "123456", nil
}

func (m *mockVerifyCodeClient) Verify(_ context.Context, _, _, _ string) (bool, error) {
	return m.valid, nil
}

// --- mockAuthService implements AuthService ---

type mockAuthService struct {
	roleAssigned bool
	err          error
}

func (m *mockAuthService) GetCaptcha(_ context.Context) (*dto.CaptchaResp, error) {
	return nil, nil
}

func (m *mockAuthService) PostCode(_ context.Context, _ *dto.ConfirmCodeReq) error {
	return nil
}

func (m *mockAuthService) Register(_ context.Context, _ *dto.UserRegisterReq) (*dto.UserResp, error) {
	return nil, nil
}

func (m *mockAuthService) AssignDefaultRole(_ context.Context, _ string) error {
	m.roleAssigned = true
	return m.err
}

// --- mockRsaService implements RsaService ---

type mockRsaService struct{}

func (m *mockRsaService) GenerateKeyPair(_ context.Context) (string, error) { return "", nil }
func (m *mockRsaService) GetPublicKey(_ context.Context) (string, error)    { return "", nil }
func (m *mockRsaService) Decrypt(_ context.Context, data string) (string, error) {
	return data, nil
}
func (m *mockRsaService) RotateKeyPair(_ context.Context) (string, error) { return "", nil }

// --- mockUserService implements UserService ---

type mockUserService struct {
	users map[string]*user.User
}

func (m *mockUserService) Gets(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]dto.UserResp, int64, error) {
	return nil, 0, nil
}

func (m *mockUserService) Get(_ context.Context, _ uint) (*dto.UserResp, error) {
	return nil, nil
}

func (m *mockUserService) GetByUsername(_ context.Context, username string) (*dto.UserResp, error) {
	if u, ok := m.users[username]; ok {
		resp := &dto.UserResp{}
		resp.Username = u.Username
		return resp, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserService) GetRawByUsername(_ context.Context, username string) (*user.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserService) Post(_ context.Context, _ *dto.UserPostReq) (*dto.UserResp, error) {
	return nil, nil
}

func (m *mockUserService) Put(_ context.Context, _ uint, _ *dto.UserPutReq) (*dto.UserResp, error) {
	return nil, nil
}

func (m *mockUserService) Delete(_ context.Context, _ uint) error {
	return nil
}

// --- mockLoginHistoryService implements LoginHistoryService ---

type mockLoginHistoryService struct{}

func (m *mockLoginHistoryService) Gets(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]dto.LoginHistoryResp, int64, error) {
	return nil, 0, nil
}

func (m *mockLoginHistoryService) Record(_ context.Context, _ *dto.LoginHistoryPostReq) error {
	return nil
}

func (m *mockLoginHistoryService) Delete(_ context.Context, _ uint) error {
	return nil
}

func (m *mockLoginHistoryService) BatchDelete(_ context.Context, _ []uint) error {
	return nil
}

// --- mockAuditLogService implements AuditLogService ---

type mockAuditLogService struct{}

func (m *mockAuditLogService) Gets(_ context.Context, _, _ int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogService) Log(_ context.Context, _ *audit_log.AuditLog) error {
	return nil
}

func (m *mockAuditLogService) LogAsync(_ *audit_log.AuditLog) {}

func (m *mockAuditLogService) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuditLogService) Wait() {}

// --- mockSettingService implements SettingService ---

type mockSettingService struct {
	settings map[string]string
}

func (m *mockSettingService) Gets(_ context.Context, _ string) ([]dto.SettingResp, error) {
	return nil, nil
}

func (m *mockSettingService) Get(_ context.Context, _ uint) (*dto.SettingResp, error) {
	return nil, nil
}

func (m *mockSettingService) GetByKey(_ context.Context, key string) (*dto.SettingResp, error) {
	if v, ok := m.settings[key]; ok {
		return &dto.SettingResp{Key: key, Value: v}, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockSettingService) Post(_ context.Context, _ *dto.SettingPostReq) (*dto.SettingResp, error) {
	return nil, nil
}

func (m *mockSettingService) Put(_ context.Context, _ uint, _ *dto.SettingPutReq) (*dto.SettingResp, error) {
	return nil, nil
}

func (m *mockSettingService) Delete(_ context.Context, _ uint) error { return nil }

func (m *mockSettingService) BatchUpdate(_ context.Context, _ *dto.SettingBatchUpdateReq) error {
	return nil
}

func (m *mockSettingService) GetPublicSettings(_ context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockSettingService) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := m.settings[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

func (m *mockSettingService) GetBool(_ context.Context, key string) (bool, error) {
	v, ok := m.settings[key]
	if !ok {
		return false, fmt.Errorf("key not found: %s", key)
	}
	return v == "true" || v == "1", nil
}

func (m *mockSettingService) GetInt(_ context.Context, key string) (int, error) {
	v, ok := m.settings[key]
	if !ok {
		return 0, fmt.Errorf("key not found: %s", key)
	}
	var i int
	_, err := fmt.Sscanf(v, "%d", &i)
	return i, err
}

// --- trackingRateLimiterMock tracks calls to IsRequestReachLimit ---

type trackingRateLimiterMock struct {
	onCall func(key string, duration time.Duration, maxAllowed int64)
}

func (m *trackingRateLimiterMock) IsRequestReachLimit(_ context.Context, key string, duration time.Duration, maxAllowed int64) (bool, error) {
	if m.onCall != nil {
		m.onCall(key, duration, maxAllowed)
	}
	return false, nil
}

// --- Helper functions ---

func newTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	return c
}
