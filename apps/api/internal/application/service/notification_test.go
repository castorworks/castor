package service

import (
	"context"
	"errors"
	"sort"
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
	notifs map[uint]*notification.Notification
	nextID uint
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

type mockUserNotificationRepo struct {
	notifRepo *mockNotificationRepo
	records   map[uint]map[uint]*notification.UserNotification // userID -> notificationID -> record
	// enabledUsers 模拟启用用户总数，用于全局通知的投递统计
	enabledUsers int64
	nextID       uint
}

func newMockUserNotificationRepo(notifRepo *mockNotificationRepo) *mockUserNotificationRepo {
	return &mockUserNotificationRepo{
		notifRepo: notifRepo,
		records:   make(map[uint]map[uint]*notification.UserNotification),
		nextID:    1,
	}
}

// deliverableGlobal 判断通知是否为未过期的全局通知
func (m *mockUserNotificationRepo) deliverableGlobal(notificationID uint) bool {
	n, ok := m.notifRepo.notifs[notificationID]
	if !ok || !n.IsGlobal {
		return false
	}
	return n.ExpireAt == nil || n.ExpireAt.After(time.Now())
}

// visibleIDs 返回对用户可见的通知 ID：全局通知 + 定向投递行，排除已删除
func (m *mockUserNotificationRepo) visibleIDs(userID uint) []uint {
	seen := make(map[uint]struct{})
	var ids []uint
	add := func(id uint) {
		if rec, ok := m.records[userID][id]; ok && rec.IsDismissed {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for id := range m.notifRepo.notifs {
		if m.deliverableGlobal(id) {
			add(id)
		}
	}
	for id := range m.records[userID] {
		add(id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// state 返回用户对某通知的状态；全局通知未实体化时返回合成的初始状态
func (m *mockUserNotificationRepo) state(userID, notificationID uint) *notification.UserNotification {
	if rec, ok := m.records[userID][notificationID]; ok {
		return rec
	}
	return &notification.UserNotification{UserID: userID, NotificationID: notificationID}
}

// materialize 为全局通知实体化关联行（幂等）
func (m *mockUserNotificationRepo) materialize(userID, notificationID uint) *notification.UserNotification {
	if rec, ok := m.records[userID][notificationID]; ok {
		return rec
	}
	if !m.deliverableGlobal(notificationID) {
		return nil
	}
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
	return rec
}

func (m *mockUserNotificationRepo) GetsByUserID(_ context.Context, userID uint, page, size int, order string, unreadOnly bool) ([]notification.NotificationWithReadStatus, int64, error) {
	var result []notification.NotificationWithReadStatus
	for _, id := range m.visibleIDs(userID) {
		rec := m.state(userID, id)
		if unreadOnly && rec.IsRead {
			continue
		}
		n := notification.Notification{ID: id, Title: "Test"}
		if entity, ok := m.notifRepo.notifs[id]; ok {
			n = *entity
		}
		result = append(result, notification.NotificationWithReadStatus{
			Notification: n,
			IsRead:       rec.IsRead,
			ReadAt:       rec.ReadAt,
			IsDismissed:  rec.IsDismissed,
			DismissedAt:  rec.DismissedAt,
		})
	}
	return result, int64(len(result)), nil
}

func (m *mockUserNotificationRepo) GetVisible(_ context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	for _, id := range m.visibleIDs(userID) {
		if id == notificationID {
			return m.state(userID, id), nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockUserNotificationRepo) GetUnreadCount(_ context.Context, userID uint) (int64, error) {
	var count int64
	for _, id := range m.visibleIDs(userID) {
		if !m.state(userID, id).IsRead {
			count++
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
	if rec := m.materialize(userID, notificationID); rec != nil && !rec.IsDismissed {
		now := time.Now()
		rec.IsRead = true
		rec.ReadAt = &now
	}
	return nil
}

func (m *mockUserNotificationRepo) BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error {
	for _, id := range notificationIDs {
		if err := m.MarkAsRead(ctx, userID, id); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockUserNotificationRepo) MarkAllAsRead(ctx context.Context, userID uint) error {
	return m.BatchMarkAsRead(ctx, userID, m.visibleIDs(userID))
}

func (m *mockUserNotificationRepo) Dismiss(_ context.Context, userID, notificationID uint) error {
	if rec := m.materialize(userID, notificationID); rec != nil {
		now := time.Now()
		rec.IsDismissed = true
		rec.DismissedAt = &now
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
	if m.deliverableGlobal(notificationID) {
		stats.RecipientCount = m.enabledUsers
		stats.UnreadCount = max(m.enabledUsers-stats.ReadCount, 0)
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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		notifRepo.notifs[77] = &notification.Notification{ID: 77, Title: "Private", IsGlobal: false}
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)

		// Create user notification record
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			100: {ID: 1, UserID: 1, NotificationID: 100},
		}

		svc := newTestNotificationService(notifRepo, userNotifRepo)
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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 1, NotificationID: 1, IsRead: true},
			2: {ID: 2, UserID: 1, NotificationID: 2, IsRead: false},
		}

		svc := newTestNotificationService(notifRepo, userNotifRepo)
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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 1, NotificationID: 1, IsRead: false},
			2: {ID: 2, UserID: 1, NotificationID: 2, IsRead: true},
			3: {ID: 3, UserID: 1, NotificationID: 3, IsRead: false},
		}

		svc := newTestNotificationService(notifRepo, userNotifRepo)
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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		userNotifRepo.records[1] = map[uint]*notification.UserNotification{
			99: {ID: 1, UserID: 1, NotificationID: 99},
		}
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		userNotifRepo.records[5] = map[uint]*notification.UserNotification{
			1: {ID: 1, UserID: 5, NotificationID: 1},
		}

		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)

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
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	notifRepo.notifs[1] = &notification.Notification{ID: 1, Title: "Stats"}
	userNotifRepo.records[10] = map[uint]*notification.UserNotification{1: {UserID: 10, NotificationID: 1, IsRead: true}}
	userNotifRepo.records[11] = map[uint]*notification.UserNotification{1: {UserID: 11, NotificationID: 1, IsRead: false}}

	svc := newTestNotificationService(notifRepo, userNotifRepo)
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
	userNotifRepo := newMockUserNotificationRepo(notifRepo)
	userNotifRepo.records[10] = map[uint]*notification.UserNotification{7: {ID: 1, UserID: 10, NotificationID: 7, IsRead: true}}
	userNotifRepo.records[11] = map[uint]*notification.UserNotification{7: {ID: 2, UserID: 11, NotificationID: 7, IsRead: false, IsDismissed: true}}
	svc := newTestNotificationService(notifRepo, userNotifRepo)

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

// ============================================
// 全局通知：读取时惰性投递
// ============================================

func postGlobal(t *testing.T, svc NotificationService, title string) *dto.NotificationResp {
	t.Helper()
	resp, err := svc.Post(context.Background(), &dto.NotificationPostReq{
		Title:    title,
		IsGlobal: true,
	})
	if err != nil {
		t.Fatalf("unexpected post error: %v", err)
	}
	return resp
}

func TestNotificationService_GlobalNotificationLazyDelivery(t *testing.T) {
	t.Run("global notification reaches a user without any delivery row", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if len(userNotifRepo.records) != 0 {
			t.Fatalf("expected no delivery rows at publish time, got %d", len(userNotifRepo.records))
		}

		items, total, err := svc.GetUserNotifications(context.Background(), 7, 1, 10, false)
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ID != resp.ID {
			t.Fatalf("expected global notification in list, total=%d items=%+v", total, items)
		}
		if items[0].IsRead {
			t.Fatal("expected global notification to be unread")
		}

		count, err := svc.GetUnreadCount(context.Background(), 7)
		if err != nil {
			t.Fatalf("unexpected count error: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected unread count 1, got %d", count)
		}
	})

	t.Run("marking read materializes exactly one row", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if err := svc.MarkAsRead(context.Background(), 7, resp.ID); err != nil {
			t.Fatalf("unexpected mark error: %v", err)
		}
		// 幂等：重复标记不应新增行
		if err := svc.MarkAsRead(context.Background(), 7, resp.ID); err != nil {
			t.Fatalf("unexpected second mark error: %v", err)
		}

		if len(userNotifRepo.records[7]) != 1 {
			t.Fatalf("expected exactly 1 materialized row, got %d", len(userNotifRepo.records[7]))
		}
		if !userNotifRepo.records[7][resp.ID].IsRead {
			t.Fatal("expected materialized row to be read")
		}

		count, err := svc.GetUnreadCount(context.Background(), 7)
		if err != nil {
			t.Fatalf("unexpected count error: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected unread count 0, got %d", count)
		}
	})

	t.Run("dismissing hides the global notification from that user only", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if err := svc.DeleteUserNotification(context.Background(), 7, resp.ID); err != nil {
			t.Fatalf("unexpected dismiss error: %v", err)
		}

		items, total, err := svc.GetUserNotifications(context.Background(), 7, 1, 10, false)
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if total != 0 || len(items) != 0 {
			t.Fatalf("expected dismissed global to be hidden, total=%d len=%d", total, len(items))
		}

		otherItems, otherTotal, err := svc.GetUserNotifications(context.Background(), 8, 1, 10, false)
		if err != nil {
			t.Fatalf("unexpected list error for other user: %v", err)
		}
		if otherTotal != 1 || len(otherItems) != 1 || otherItems[0].IsRead {
			t.Fatalf("expected other user unaffected, total=%d items=%+v", otherTotal, otherItems)
		}
	})

	t.Run("mark all as read covers not yet materialized globals", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if err := svc.MarkAllAsRead(context.Background(), 7); err != nil {
			t.Fatalf("unexpected mark all error: %v", err)
		}

		rec, ok := userNotifRepo.records[7][resp.ID]
		if !ok || !rec.IsRead {
			t.Fatalf("expected global notification to be materialized and read, rec=%+v", rec)
		}
	})

	t.Run("batch mark as read covers not yet materialized globals", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if err := svc.BatchMarkAsRead(context.Background(), 7, []uint{resp.ID}); err != nil {
			t.Fatalf("unexpected batch mark error: %v", err)
		}
		if rec := userNotifRepo.records[7][resp.ID]; rec == nil || !rec.IsRead {
			t.Fatalf("expected global notification to be materialized and read, rec=%+v", rec)
		}
	})
}

func TestNotificationService_ExpiredGlobalNotVisible(t *testing.T) {
	t.Run("expired global notification is neither listed nor markable", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		expired := time.Now().Add(-time.Hour)
		notifRepo.notifs[55] = &notification.Notification{ID: 55, Title: "Expired", IsGlobal: true, ExpireAt: &expired}
		svc := newTestNotificationService(notifRepo, userNotifRepo)

		_, total, err := svc.GetUserNotifications(context.Background(), 7, 1, 10, false)
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if total != 0 {
			t.Fatalf("expected expired global to be hidden, total=%d", total)
		}

		if err := svc.MarkAsRead(context.Background(), 7, 55); !errors.Is(err, apperror.ErrNotificationNotDelivered) {
			t.Fatalf("expected ErrNotificationNotDelivered, got %v", err)
		}
	})
}

func TestNotificationService_GlobalDeliveryStats(t *testing.T) {
	t.Run("global recipient count uses enabled user total", func(t *testing.T) {
		notifRepo := newMockNotificationRepo()
		userNotifRepo := newMockUserNotificationRepo(notifRepo)
		userNotifRepo.enabledUsers = 4
		svc := newTestNotificationService(notifRepo, userNotifRepo)
		resp := postGlobal(t, svc, "Global")

		if err := svc.MarkAsRead(context.Background(), 7, resp.ID); err != nil {
			t.Fatalf("unexpected mark error: %v", err)
		}

		got, err := svc.Get(context.Background(), resp.ID)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}
		if got.RecipientCount != 4 {
			t.Fatalf("expected recipient count 4, got %d", got.RecipientCount)
		}
		if got.ReadCount != 1 {
			t.Fatalf("expected read count 1, got %d", got.ReadCount)
		}
		if got.UnreadCount != 3 {
			t.Fatalf("expected unread count 3, got %d", got.UnreadCount)
		}
	})
}
