package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/pkg/query"
)

// ============================================
// Mock audit log repository
// ============================================

type mockAuditLogRepo struct {
	logs   []audit_log.AuditLog
	nextID uint
	mu     sync.Mutex
}

func newMockAuditLogRepo() *mockAuditLogRepo {
	return &mockAuditLogRepo{}
}

func (m *mockAuditLogRepo) Gets(_ context.Context, page, size int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := int64(len(m.logs))
	start := (page - 1) * size
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > len(m.logs) {
		end = len(m.logs)
	}
	if start >= len(m.logs) {
		return []audit_log.AuditLog{}, total, nil
	}
	return m.logs[start:end], total, nil
}

func (m *mockAuditLogRepo) Create(_ context.Context, log *audit_log.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	log.ID = m.nextID
	log.CreatedAt = time.Now()
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockAuditLogRepo) Get(_ context.Context, id uint) (*audit_log.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, l := range m.logs {
		if l.ID == id {
			return &l, nil
		}
	}
	return nil, nil
}

func (m *mockAuditLogRepo) DeleteBefore(_ context.Context, before time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var remaining []audit_log.AuditLog
	var deleted int64
	for _, l := range m.logs {
		if l.CreatedAt.Before(before) {
			deleted++
		} else {
			remaining = append(remaining, l)
		}
	}
	m.logs = remaining
	return deleted, nil
}

// ============================================
// Tests
// ============================================

func TestAuditLogService_Log(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	log := &audit_log.AuditLog{
		LogType:    audit_log.AuditLogTypeUserCreate,
		Operator:   "admin",
		OperatorID: 1,
		Target:     "user",
		Details:    "created user",
	}

	err := svc.Log(context.Background(), log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(repo.logs))
	}
	if repo.logs[0].LogType != audit_log.AuditLogTypeUserCreate {
		t.Errorf("expected log type %q, got %q", audit_log.AuditLogTypeUserCreate, repo.logs[0].LogType)
	}
}

func TestAuditLogService_Gets(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	// Add some logs
	for i := 0; i < 5; i++ {
		_ = svc.Log(context.Background(), &audit_log.AuditLog{
			LogType:  audit_log.AuditLogTypeLogin,
			Operator: "test",
		})
	}

	logs, total, err := svc.Gets(context.Background(), fullScope, 1, 3, "created_at DESC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs on first page, got %d", len(logs))
	}

	logs, total, err = svc.Gets(context.Background(), fullScope, 2, 3, "created_at DESC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs on second page, got %d", len(logs))
	}
}

func TestAuditLogService_LogAsync(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	// Log asynchronously
	svc.LogAsync(&audit_log.AuditLog{
		LogType:  audit_log.AuditLogTypeLogin,
		Operator: "async_test",
	})

	// Wait for async operation to complete
	svc.Wait()

	if len(repo.logs) != 1 {
		t.Errorf("expected 1 async log, got %d", len(repo.logs))
	}
	if repo.logs[0].Operator != "async_test" {
		t.Errorf("expected operator 'async_test', got %q", repo.logs[0].Operator)
	}
}

func TestAuditLogService_Wait(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	// Log multiple async operations
	for i := 0; i < 10; i++ {
		svc.LogAsync(&audit_log.AuditLog{
			LogType:  audit_log.AuditLogTypeLogin,
			Operator: "batch_async",
		})
	}

	// Wait for all to complete
	svc.Wait()

	if len(repo.logs) != 10 {
		t.Errorf("expected 10 async logs after wait, got %d", len(repo.logs))
	}
}

func TestAuditLogService_Wait_NoPending(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	// Wait with no pending operations should not block
	done := make(chan bool, 1)
	go func() {
		svc.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Good - completed quickly
	case <-time.After(1 * time.Second):
		t.Error("Wait() blocked unexpectedly")
	}
}

func TestAuditLogService_DeleteBefore(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	// Create logs with different timestamps
	for i := 0; i < 5; i++ {
		_ = svc.Log(context.Background(), &audit_log.AuditLog{
			LogType:  audit_log.AuditLogTypeLogin,
			Operator: "old_user",
		})
	}

	// All logs were created "now", so deleting before "now + 1s" should remove all
	cutoff := time.Now().Add(1 * time.Second)
	deleted, err := svc.DeleteBefore(context.Background(), fullScope, cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 5 {
		t.Errorf("expected 5 deleted, got %d", deleted)
	}
	if len(repo.logs) != 0 {
		t.Errorf("expected 0 remaining logs, got %d", len(repo.logs))
	}
}
