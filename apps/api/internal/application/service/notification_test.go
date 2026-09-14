package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
)

// ============================================
// Mock repositories for notification service
// ============================================

type mockNotificationRepo struct {
	notifs       map[uint]*notification.Notification
	globalNotifs []uint
	nextID       uint
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		notifs: make(map[uint]*notification.Notification),
		nextID: 1,
	}
}

func (m *mockNotificationRepo) Gets(_ context.Context, page, size int, order string, opts ...query.Option) ([]notification.Notification, int64, error) {
	var result []notification.Notification
	for _, n := range m.notifs {
		result = append(result, *n)
	}
	total := int64(len(result))
	return result, total, nil
}

func (m *mockNotificationRepo) Get(_ context.Context, id uint) (*notification.Notification, error) {
	if n, ok := m.notifs[id]; ok {
		return n, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockNotificationRepo) Create(_ context.Context, n *notification.Notification) error {
	n.ID = m.nextID
	m.nextID++
	m.notifs[n.ID] = n
	if n.IsGlobal {
		m.globalNotifs = append(m.globalNotifs, n.ID)
	}
	return nil
}

func (m *mockNotificationRepo) Update(_ context.Context, n *notification.Notification) error {
	m.notifs[n.ID] = n
	return nil
}

func (m *mockNotificationRepo) Delete(_ context.Context, id uint) error {
	delete(m.notifs, id)
	return nil
}

func (m *mockNotificationRepo) BatchDelete(_ context.Context, ids []uint) error {
	for _, id := range ids {
		delete(m.notifs, id)
	}
	return nil
}

func (m *mockNotificationRepo) GetGlobalNotifications(_ context.Context) ([]notification.Notification, error) {
	var result []notification.Notification
	for _, id := range m.globalNotifs {
		if n, ok := m.notifs[id]; ok {
			result = append(result, *n)
		}
	}
	return result, nil
}

type mockUserNotificationRepo struct {
	records map[uint]map[uint]*notification.UserNotification // userID -> notificationID -> record
	nextID  uint
}

func newMockUserNotificationRepo() *mockUserNotificationRepo {
	return &mockUserNotificationRepo{
		records: make(map[uint]map[uint]*notification.UserNotification),
		nextID:  1,
	}
}

func (m *mockUserNotificationRepo) GetsByUserID(_ context.Context, userID uint, page, size int, order string, unreadOnly bool) ([]notification.NotificationWithReadStatus, int64, error) {
	var result []notification.NotificationWithReadStatus
	if userRecords, ok := m.records[userID]; ok {
		for _, rec := range userRecords {
			if rec.IsDismissed {
				continue
			}
			if unreadOnly && rec.IsRead {
				continue
			}
			result = append(result, notification.NotificationWithReadStatus{
				Notification: notification.Notification{
					ID:    rec.NotificationID,
					Title: "Test",
				},
				IsRead:      rec.IsRead,
				ReadAt:      rec.ReadAt,
				IsDismissed: rec.IsDismissed,
				DismissedAt: rec.DismissedAt,
			})
		}
	}
	total := int64(len(result))
	return result, total, nil
}

func (m *mockUserNotificationRepo) GetDelivered(_ context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	if userRecords, ok := m.records[userID]; ok {
		if rec, ok := userRecords[notificationID]; ok && !rec.IsDismissed {
			return rec, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserNotificationRepo) GetUnreadCount(_ context.Context, userID uint) (int64, error) {
	var count int64
	if userRecords, ok := m.records[userID]; ok {
		for _, rec := range userRecords {
			if !rec.IsDismissed && !rec.IsRead {
				count++
			}
		}
	}
	return count, nil
}

func (m *mockUserNotificationRepo) Create(_ context.Context, rec *notification.UserNotification) error {
	if m.records[rec.UserID] == nil {
		m.records[rec.UserID] = make(map[uint]*notification.UserNotification)
	}
	rec.ID = m.nextID
	m.nextID++
	m.records[rec.UserID][rec.NotificationID] = rec
	return nil
}

func (m *mockUserNotificationRepo) BatchCreate(_ context.Context, recs []notification.UserNotification) error {
	for i := range recs {
		if m.records[recs[i].UserID] == nil {
			m.records[recs[i].UserID] = make(map[uint]*notification.UserNotification)
		}
		recs[i].ID = m.nextID
		recs[i].CreatedAt = time.Now()
		m.nextID++
		m.records[recs[i].UserID][recs[i].NotificationID] = &recs[i]
	}
	return nil
}

func (m *mockUserNotificationRepo) ReplaceRecipients(_ context.Context, notificationID uint, userIDs []uint) error {
	for userID, userRecords := range m.records {
		delete(userRecords, notificationID)
		if len(userRecords) == 0 {
			delete(m.records, userID)
		}
	}
	for _, userID := range userIDs {
		if m.records[userID] == nil {
			m.records[userID] = make(map[uint]*notification.UserNotification)
		}
		rec := &notification.UserNotification{
			ID:             m.nextID,
			CreatedAt:      time.Now(),
			UserID:         userID,
			NotificationID: notificationID,
		}
		m.nextID++
		m.records[userID][notificationID] = rec
	}
	return nil
}

func (m *mockUserNotificationRepo) MarkAsRead(_ context.Context, userID, notificationID uint) error {
	if userRecords, ok := m.records[userID]; ok {
		if rec, ok := userRecords[notificationID]; ok {
			rec.IsRead = true
		}
	}
	return nil
}

func (m *mockUserNotificationRepo) BatchMarkAsRead(_ context.Context, userID uint, notificationIDs []uint) error {
	if userRecords, ok := m.records[userID]; ok {
		for _, id := range notificationIDs {
			if rec, ok := userRecords[id]; ok {
				rec.IsRead = true
			}
		}
	}
	return nil
}

func (m *mockUserNotificationRepo) MarkAllAsRead(_ context.Context, userID uint) error {
	if userRecords, ok := m.records[userID]; ok {
		for _, rec := range userRecords {
			if !rec.IsDismissed {
				rec.IsRead = true
			}
		}
	}
	return nil
}

func (m *mockUserNotificationRepo) Dismiss(_ context.Context, userID, notificationID uint) error {
	if userRecords, ok := m.records[userID]; ok {
		if rec, ok := userRecords[notificationID]; ok {
			rec.IsDismissed = true
		}
	}
	return nil
}

func (m *mockUserNotificationRepo) StatsByNotificationID(_ context.Context, notificationID uint) (*notification.NotificationDeliveryStats, error) {
	stats := &notification.NotificationDeliveryStats{}
	for _, userRecords := range m.records {
		rec, ok := userRecords[notificationID]
		if !ok {
			continue
		}
		stats.RecipientCount++
		if rec.IsDismissed {
			stats.DismissedCount++
			continue
		}
		if rec.IsRead {
			stats.ReadCount++
		} else {
			stats.UnreadCount++
		}
	}
	return stats, nil
}

func (m *mockUserNotificationRepo) GetRecipients(_ context.Context, notificationID uint, page, size int) ([]notification.UserNotification, int64, error) {
	var result []notification.UserNotification
	for _, userRecords := range m.records {
		if rec, ok := userRecords[notificationID]; ok {
			if rec.CreatedAt.IsZero() {
				rec.CreatedAt = time.Now()
			}
			result = append(result, *rec)
		}
	}
	return result, int64(len(result)), nil
}

// ============================================
// Tests
// ============================================

func TestNotificationService_Post(t *testing.T) {
	t.Run("creates notification with default type and level", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		req := &dto.NotificationPostReq{
			Title:    "Test Notification",
			Content:  "Test content",
			IsGlobal: true,
		}

		resp, err := svc.Post(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Title != "Test Notification" {
			t.Errorf("expected title 'Test Notification', got %q", resp.Title)
		}
		if resp.Type != string(notification.TypeSystem) {
			t.Errorf("expected type %q, got %q", notification.TypeSystem, resp.Type)
		}
		if resp.Level != string(notification.LevelInfo) {
			t.Errorf("expected level %q, got %q", notification.LevelInfo, resp.Level)
		}
	})

	t.Run("creates with specified type and level", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		req := &dto.NotificationPostReq{
			Title:    "Test",
			Type:     string(notification.TypeAlert),
			Level:    string(notification.LevelWarning),
			IsGlobal: true,
		}

		resp, err := svc.Post(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Type != string(notification.TypeAlert) {
			t.Errorf("expected type %q, got %q", notification.TypeAlert, resp.Type)
		}
	})

	t.Run("creates user notification records for non-global notification", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		req := &dto.NotificationPostReq{
			Title:    "User-specific",
			IsGlobal: false,
			UserIDs:  []uint{1, 2, 3},
		}

		_, err := svc.Post(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The notification was created, check user records exist
		if len(userNotifRepo.records) != 3 {
			t.Errorf("expected 3 user notification records, got %d", len(userNotifRepo.records))
		}
	})
}

func TestNotificationService_Post_DeliveryValidation(t *testing.T) {
	t.Run("rejects targeted notification without recipients", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		_, err := svc.Post(context.Background(), &dto.NotificationPostReq{
			Title:    "Targeted",
			IsGlobal: false,
			UserIDs:  nil,
		})

		if !errors.Is(err, apperror.ErrNotificationRecipientsRequired) {
			t.Fatalf("expected ErrNotificationRecipientsRequired, got %v", err)
		}
	})

	t.Run("stores explicit delivery rows for targeted notification", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		resp, err := svc.Post(context.Background(), &dto.NotificationPostReq{
			Title:    "Targeted",
			IsGlobal: false,
			UserIDs:  []uint{3, 3, 5},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.RecipientCount != 2 {
			t.Fatalf("expected 2 unique recipients, got %d", resp.RecipientCount)
		}
		if _, ok := userNotifRepo.records[3][resp.ID]; !ok {
			t.Fatalf("expected delivery for user 3")
		}
		if _, ok := userNotifRepo.records[5][resp.ID]; !ok {
			t.Fatalf("expected delivery for user 5")
		}
	})
}

func TestNotificationService_UserDeleteGlobalDelivery(t *testing.T) {
	t.Run("dismisses global notification after delivery row exists", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		resp, err := svc.Post(context.Background(), &dto.NotificationPostReq{
			Title:    "Global",
			IsGlobal: true,
			UserIDs:  []uint{9},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := svc.DeleteUserNotification(context.Background(), 9, resp.ID); err != nil {
			t.Fatalf("unexpected delete error: %v", err)
		}

		items, total, err := svc.GetUserNotifications(context.Background(), 9, 1, 10, false)
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if total != 0 || len(items) != 0 {
			t.Fatalf("expected dismissed notification to be hidden, total=%d len=%d", total, len(items))
		}
	})
}

func TestNotificationService_MarkAsReadAuthorization(t *testing.T) {
	t.Run("does not create read row for notification not delivered to user", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		notifRepo.notifs[77] = &notification.Notification{ID: 77, Title: "Private", IsGlobal: false}
		svc := NewNotificationService(notifRepo, userNotifRepo)

		err := svc.MarkAsRead(context.Background(), 42, 77)

		if !errors.Is(err, apperror.ErrNotificationNotDelivered) {
			t.Fatalf("expected ErrNotificationNotDelivered, got %v", err)
		}
	})
}

func TestNotificationService_Put(t *testing.T) {
	t.Run("updates notification", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		notifRepo.notifs[1] = &notification.Notification{
			ID: 1, Title: "Old Title", Content: "Old Content",
		}
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		req := &dto.NotificationPutReq{Title: "New Title", Content: "New Content"}
		resp, err := svc.Put(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Title != "New Title" {
			t.Errorf("expected title 'New Title', got %q", resp.Title)
		}
		if resp.Content != "New Content" {
			t.Errorf("expected content 'New Content', got %q", resp.Content)
		}
	})

	t.Run("empty title does not overwrite", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		notifRepo.notifs[1] = &notification.Notification{
			ID: 1, Title: "Keep Title",
		}
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		req := &dto.NotificationPutReq{Content: "New Content"}
		resp, err := svc.Put(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Title != "Keep Title" {
			t.Errorf("expected title unchanged, got %q", resp.Title)
		}
	})
}

func TestNotificationService_GetUserNotifications(t *testing.T) {
	t.Run("returns user notifications", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()

		// Create user notification record
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			100: {ID: 1, UserID: 1, NotificationID: 100},
		}

		svc := NewNotificationService(notifRepo, userNotifRepo)
		resp, total, err := svc.GetUserNotifications(context.Background(), 1, 10, 10, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(resp) != 1 {
			t.Errorf("expected 1 response, got %d", len(resp))
		}
	})

	t.Run("unreadOnly filters correctly", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 1, NotificationID: 1, IsRead: true},
			2: {ID: 2, UserID: 1, NotificationID: 2, IsRead: false},
		}

		svc := NewNotificationService(notifRepo, userNotifRepo)
		resp, _, err := svc.GetUserNotifications(context.Background(), 1, 10, 10, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 1 {
			t.Errorf("expected 1 unread notification, got %d", len(resp))
		}
	})
}

func TestNotificationService_GetUnreadCount(t *testing.T) {
	t.Run("returns count of unread notifications", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 1, NotificationID: 1, IsRead: false},
			2: {ID: 2, UserID: 1, NotificationID: 2, IsRead: true},
			3: {ID: 3, UserID: 1, NotificationID: 3, IsRead: false},
		}

		svc := NewNotificationService(notifRepo, userNotifRepo)
		count, err := svc.GetUnreadCount(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 unread, got %d", count)
		}
	})
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	t.Run("marks delivered notification as read", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			99: {ID: 1, UserID: 1, NotificationID: 99},
		}
		svc := NewNotificationService(notifRepo, userNotifRepo)

		err := svc.MarkAsRead(context.Background(), 1, 99)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !userNotifRepo.records[1][99].IsRead {
			t.Error("expected record to be marked as read")
		}
	})
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	t.Run("marks delivered notifications as read", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo()
		userNotifRepo.records[5] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 5, NotificationID: 1},
		}

		svc := NewNotificationService(notifRepo, userNotifRepo)

		err := svc.MarkAllAsRead(context.Background(), 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !userNotifRepo.records[5][1].IsRead {
			t.Error("expected delivered notification to be marked as read")
		}
	})
}

func TestNotificationService_Delete(t *testing.T) {
	t.Run("deletes notification", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		notifRepo.notifs[1] = &notification.Notification{ID: 1, Title: "To Delete"}
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		err := svc.Delete(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = notifRepo.Get(context.Background(), 1)
		if err == nil {
			t.Error("expected notification to be deleted")
		}
	})
}

func TestNotificationService_BatchDelete(t *testing.T) {
	t.Run("deletes multiple notifications", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		notifRepo.notifs[1] = &notification.Notification{ID: 1}
		notifRepo.notifs[2] = &notification.Notification{ID: 2}
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		err := svc.BatchDelete(context.Background(), []uint{1, 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifRepo.notifs) != 0 {
			t.Error("expected all notifications to be deleted")
		}
	})
}

func TestNotificationService_Gets(t *testing.T) {
	t.Run("returns notification list", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		notifRepo.notifs[1] = &notification.Notification{ID: 1, Title: "One"}
		notifRepo.notifs[2] = &notification.Notification{ID: 2, Title: "Two"}
		userNotifRepo := newMockUserNotificationRepo()
		svc := NewNotificationService(notifRepo, userNotifRepo)

		resp, total, err := svc.Gets(context.Background(), 1, 10, "created_at DESC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 responses, got %d", len(resp))
		}
	})
}

func TestNotificationService_GetIncludesDeliveryStats(t *testing.T) {
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo()
	notifRepo.notifs[1] = &notification.Notification{ID: 1, Title: "Stats"}
	userNotifRepo.records[10] = map[uint]*notification.UserNotification{1: {UserID: 10, NotificationID: 1, IsRead: true}}
	userNotifRepo.records[11] = map[uint]*notification.UserNotification{1: {UserID: 11, NotificationID: 1, IsRead: false}}

	svc := NewNotificationService(notifRepo, userNotifRepo)
	resp, err := svc.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.RecipientCount != 2 || resp.ReadCount != 1 || resp.UnreadCount != 1 {
		t.Fatalf("unexpected stats: %#v", resp)
	}
}

func TestNotificationService_GetRecipients(t *testing.T) {
	notifRepo := newMockNotificationRepo()
	userNotifRepo := newMockUserNotificationRepo()
	userNotifRepo.records[10] = map[uint]*notification.UserNotification{7: {ID: 1, UserID: 10, NotificationID: 7, IsRead: true}}
	userNotifRepo.records[11] = map[uint]*notification.UserNotification{7: {ID: 2, UserID: 11, NotificationID: 7, IsRead: false, IsDismissed: true}}
	svc := NewNotificationService(notifRepo, userNotifRepo)

	items, total, err := svc.GetRecipients(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 recipients, total=%d len=%d", total, len(items))
	}
	if items[0].UserID == 0 || items[0].DeliveredAt.IsZero() {
		t.Fatalf("expected recipient user id and delivery time, got %#v", items[0])
	}
}
