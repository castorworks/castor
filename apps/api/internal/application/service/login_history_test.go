package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
)

// --- mockLoginHistoryRepo implements login_history.Repository ---

type mockLoginHistoryRepo struct {
	items     []login_history.LoginHistory
	nextID    uint
	createErr error
	deleteErr error
	batchErr  error
	getErr    error
}

func newMockLoginHistoryRepo() *mockLoginHistoryRepo {
	return &mockLoginHistoryRepo{nextID: 1}
}

func (m *mockLoginHistoryRepo) CountSuccessfulToday(_ context.Context, _ ...query.Option) (int64, error) {
	return 0, nil
}

func (m *mockLoginHistoryRepo) CountSuccessfulByMonth(_ context.Context, _ int, _ ...query.Option) ([]shared.MonthlyCount, error) {
	return []shared.MonthlyCount{}, nil
}

func (m *mockLoginHistoryRepo) Gets(_ context.Context, page, size int, _ string, _ ...query.Option) ([]login_history.LoginHistory, int64, error) {
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	total := int64(len(m.items))
	start := (page - 1) * size
	if start >= len(m.items) {
		return nil, total, nil
	}
	end := start + size
	if end > len(m.items) {
		end = len(m.items)
	}
	return m.items[start:end], total, nil
}

func (m *mockLoginHistoryRepo) Get(_ context.Context, id uint) (*login_history.LoginHistory, error) {
	for i := range m.items {
		if m.items[i].ID == id {
			return &m.items[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockLoginHistoryRepo) Create(_ context.Context, item *login_history.LoginHistory) error {
	if m.createErr != nil {
		return m.createErr
	}
	item.ID = m.nextID
	m.nextID++
	m.items = append(m.items, *item)
	return nil
}

func (m *mockLoginHistoryRepo) Delete(_ context.Context, id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i := range m.items {
		if m.items[i].ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockLoginHistoryRepo) BatchDelete(_ context.Context, ids []uint) error {
	if m.batchErr != nil {
		return m.batchErr
	}
	idSet := make(map[uint]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	var remaining []login_history.LoginHistory
	for _, item := range m.items {
		if !idSet[item.ID] {
			remaining = append(remaining, item)
		}
	}
	m.items = remaining
	return nil
}

func TestLoginHistoryService_Record(t *testing.T) {
	t.Parallel()

	t.Run("successful record", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		err := svc.Record(context.Background(), &dto.LoginHistoryPostReq{
			UserID:      1,
			Username:    "testuser",
			IpAddr:      "192.168.1.1",
			UserAgent:   "Mozilla/5.0",
			LoginMethod: "password",
			Success:     true,
		})

		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(repo.items) != 1 {
			t.Fatalf("expected 1 item, got: %d", len(repo.items))
		}
		if repo.items[0].Username != "testuser" {
			t.Errorf("expected username 'testuser', got: %s", repo.items[0].Username)
		}
		if repo.items[0].UserID != 1 {
			t.Errorf("expected userID 1, got: %d", repo.items[0].UserID)
		}
		if repo.items[0].UserAgent != "Mozilla/5.0" {
			t.Errorf("expected userAgent 'Mozilla/5.0', got: %s", repo.items[0].UserAgent)
		}
	})

	t.Run("record with create error", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		repo.createErr = errors.New("db error")
		svc := NewLoginHistoryService(repo)

		err := svc.Record(context.Background(), &dto.LoginHistoryPostReq{
			Username:    "testuser",
			IpAddr:      "192.168.1.1",
			LoginMethod: "password",
			Success:     false,
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLoginHistoryService_Gets(t *testing.T) {
	t.Parallel()

	t.Run("returns paginated results", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		// Create 3 records
		for i := 0; i < 3; i++ {
			_ = svc.Record(context.Background(), &dto.LoginHistoryPostReq{
				Username:    "user",
				IpAddr:      "1.2.3.4",
				LoginMethod: "password",
				Success:     true,
			})
		}

		items, total, err := svc.Gets(context.Background(), fullScope, 1, 2, "id desc")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got: %d", total)
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items, got: %d", len(items))
		}
	})

	t.Run("returns empty for no data", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		items, total, err := svc.Gets(context.Background(), fullScope, 1, 10, "id desc")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got: %d", total)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got: %d", len(items))
		}
	})

	t.Run("filters by username via option", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		_ = svc.Record(context.Background(), &dto.LoginHistoryPostReq{
			Username: "alice", IpAddr: "1.1.1.1", LoginMethod: "password", Success: true,
		})
		_ = svc.Record(context.Background(), &dto.LoginHistoryPostReq{
			Username: "bob", IpAddr: "2.2.2.2", LoginMethod: "email", Success: true,
		})

		// Gets returns all since mock doesn't filter by opts
		items, total, err := svc.Gets(context.Background(), fullScope, 1, 10, "id desc", *query.NewOption("username = ?", "alice"))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Mock doesn't actually filter, just verify no error
		_ = items
		_ = total
	})
}

func TestLoginHistoryService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		_ = svc.Record(context.Background(), &dto.LoginHistoryPostReq{
			Username: "user", IpAddr: "1.1.1.1", LoginMethod: "password", Success: true,
		})

		err := svc.Delete(context.Background(), fullScope, 1)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(repo.items) != 0 {
			t.Errorf("expected 0 items after delete, got: %d", len(repo.items))
		}
	})

	t.Run("delete non-existent returns error", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		err := svc.Delete(context.Background(), fullScope, 999)
		if err == nil {
			t.Fatal("expected error for non-existent ID")
		}
	})
}

func TestLoginHistoryService_BatchDelete(t *testing.T) {
	t.Parallel()

	t.Run("successful batch delete", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		svc := NewLoginHistoryService(repo)

		for i := 0; i < 5; i++ {
			_ = svc.Record(context.Background(), &dto.LoginHistoryPostReq{
				Username: "user", IpAddr: "1.1.1.1", LoginMethod: "password", Success: true,
			})
		}

		err := svc.BatchDelete(context.Background(), fullScope, []uint{1, 3, 5})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(repo.items) != 2 {
			t.Errorf("expected 2 items remaining, got: %d", len(repo.items))
		}
	})

	t.Run("batch delete with error", func(t *testing.T) {
		t.Parallel()
		repo := newMockLoginHistoryRepo()
		repo.batchErr = errors.New("db error")
		svc := NewLoginHistoryService(repo)

		err := svc.BatchDelete(context.Background(), fullScope, []uint{1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
