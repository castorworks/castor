package service

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/audit_log"
)

func TestNewRsaService_KeySelection(t *testing.T) {
	tests := []struct {
		name    string
		keys    RsaKeyConfig
		wantKey bool
	}{
		{name: "dedicated secret", keys: RsaKeyConfig{PrivateKeySecret: "rsa-secret", FallbackSecret: "jwt-key"}, wantKey: true},
		{name: "falls back to jwt key", keys: RsaKeyConfig{FallbackSecret: "jwt-key"}, wantKey: true},
		{name: "no key material", keys: RsaKeyConfig{}, wantKey: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewRsaService(nil, "castor", tt.keys).(*rsaService)
			if got := len(svc.aesKey) > 0; got != tt.wantKey {
				t.Fatalf("aes key present = %v, want %v", got, tt.wantKey)
			}
		})
	}

	dedicated := NewRsaService(nil, "castor", RsaKeyConfig{PrivateKeySecret: "rsa-secret", FallbackSecret: "jwt-key"}).(*rsaService)
	fallback := NewRsaService(nil, "castor", RsaKeyConfig{FallbackSecret: "jwt-key"}).(*rsaService)
	if bytes.Equal(dedicated.aesKey, fallback.aesKey) {
		t.Fatal("dedicated secret must take precedence over fallback secret")
	}
}

type recordingLoginHistoryService struct {
	mockLoginHistoryService
	mu  sync.Mutex
	req *dto.LoginHistoryPostReq
}

func (m *recordingLoginHistoryService) Record(_ context.Context, req *dto.LoginHistoryPostReq) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.req = req
	return nil
}

type recordingAuditLogService struct {
	mockAuditLogService
	mu  sync.Mutex
	log *audit_log.AuditLog
}

func (m *recordingAuditLogService) LogAsync(log *audit_log.AuditLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.log = log
}

func TestLoginService_RecordLoginUsesClientInfo(t *testing.T) {
	history := &recordingLoginHistoryService{}
	audit := &recordingAuditLogService{}
	svc := &LoginService{loginHistoryService: history, auditLogService: audit}

	svc.recordLogin("alice", LoginClient{IP: "203.0.113.7", UserAgent: "test-agent"}, "password", false, 42)

	if history.req == nil || history.req.IpAddr != "203.0.113.7" || history.req.UserAgent != "test-agent" || history.req.UserID != 42 || history.req.Success {
		t.Fatalf("unexpected login history: %+v", history.req)
	}
	if audit.log == nil || audit.log.IpAddr != "203.0.113.7" || audit.log.Success || audit.log.Details != "Login failed via password" {
		t.Fatalf("unexpected audit log: %+v", audit.log)
	}
}
