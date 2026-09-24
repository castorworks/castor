package persistence

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// notificationTestSchema 让本文件的表与其它包的集成测试（bootstrap 使用 public）互不干扰。
const notificationTestSchema = "notification_repo_test"

// withSearchPath 把 search_path 追加到 DSN，兼容 URL 与 key=value 两种写法。
func withSearchPath(dsn, searchPath string) string {
	if strings.Contains(dsn, "://") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		return dsn + sep + "search_path=" + searchPath
	}
	return dsn + " search_path=" + searchPath
}

// 需要一次性的空库，绝不能指向开发库或生产库。
func openNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA IF NOT EXISTS " + notificationTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, notificationTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UserModel{}, &models.NotificationModel{}, &models.UserNotificationModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE user_notifications, notifications, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username string, enable bool) uint {
	t.Helper()
	m := &models.UserModel{Username: username, AccountSource: "local", Enable: enable}
	if err := db.Create(m).Error; err != nil {
		t.Fatal(err)
	}
	return m.ID
}

func countUserNotificationRows(t *testing.T, db *gorm.DB, userID, notificationID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.UserNotificationModel{}).
		Where("user_id = ? AND notification_id = ?", userID, notificationID).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

// TestUserNotificationRepositoryGlobalLazyDelivery 验证全局通知在读取时惰性投递：
// 发布时不写 user_notifications，用户仍能看到；用户操作时才实体化关联行。
func TestUserNotificationRepositoryGlobalLazyDelivery(t *testing.T) {
	db := openNotificationTestDB(t)
	ctx := context.Background()
	notifRepo := NewNotificationRepository(db)
	repo := NewUserNotificationRepository(db)

	alice := createTestUser(t, db, "alice", true)
	bob := createTestUser(t, db, "bob", true)
	createTestUser(t, db, "disabled", false)

	global := &notification.Notification{Title: "Global", Type: notification.TypeSystem, Level: notification.LevelInfo, IsGlobal: true}
	if err := notifRepo.Create(ctx, global); err != nil {
		t.Fatal(err)
	}
	targeted := &notification.Notification{Title: "Targeted", Type: notification.TypeSystem, Level: notification.LevelInfo}
	if err := notifRepo.Create(ctx, targeted); err != nil {
		t.Fatal(err)
	}
	if err := repo.BatchCreate(ctx, []notification.UserNotification{{UserID: alice, NotificationID: targeted.ID}}); err != nil {
		t.Fatal(err)
	}

	t.Run("global notification is listed for a user without any delivery row", func(t *testing.T) {
		if got := countUserNotificationRows(t, db, bob, global.ID); got != 0 {
			t.Fatalf("expected no delivery row for bob, got %d", got)
		}

		items, total, err := repo.GetsByUserID(ctx, bob, 1, 10, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 || len(items) != 1 {
			t.Fatalf("expected 1 visible notification for bob, total=%d len=%d", total, len(items))
		}
		if items[0].ID != global.ID || items[0].Title != "Global" {
			t.Fatalf("unexpected notification for bob: %+v", items[0])
		}
		if items[0].IsRead || items[0].IsDismissed {
			t.Fatalf("expected synthesized unread state, got %+v", items[0])
		}

		count, err := repo.GetUnreadCount(ctx, bob)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected unread count 1 for bob, got %d", count)
		}
	})

	t.Run("targeted and global notifications are merged for the addressed user", func(t *testing.T) {
		items, total, err := repo.GetsByUserID(ctx, alice, 1, 10, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 || len(items) != 2 {
			t.Fatalf("expected 2 visible notifications for alice, total=%d len=%d", total, len(items))
		}
		count, err := repo.GetUnreadCount(ctx, alice)
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("expected unread count 2 for alice, got %d", count)
		}
	})

	t.Run("GetVisible synthesizes state for a global without a row", func(t *testing.T) {
		item, err := repo.GetVisible(ctx, bob, global.ID)
		if err != nil {
			t.Fatal(err)
		}
		if item.UserID != bob || item.NotificationID != global.ID || item.IsRead || item.IsDismissed {
			t.Fatalf("unexpected synthesized state: %+v", item)
		}
		if _, err := repo.GetVisible(ctx, bob, targeted.ID); err != shared.ErrNotFound {
			t.Fatalf("expected ErrNotFound for a targeted notification bob never received, got %v", err)
		}
	})

	t.Run("marking read materializes exactly one row and is idempotent", func(t *testing.T) {
		if err := repo.MarkAsRead(ctx, alice, global.ID); err != nil {
			t.Fatal(err)
		}
		if err := repo.MarkAsRead(ctx, alice, global.ID); err != nil {
			t.Fatal(err)
		}
		if got := countUserNotificationRows(t, db, alice, global.ID); got != 1 {
			t.Fatalf("expected exactly 1 materialized row, got %d", got)
		}

		item, err := repo.GetVisible(ctx, alice, global.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !item.IsRead || item.ReadAt == nil {
			t.Fatalf("expected read state, got %+v", item)
		}

		unread, err := repo.GetUnreadCount(ctx, alice)
		if err != nil {
			t.Fatal(err)
		}
		if unread != 1 {
			t.Fatalf("expected alice unread count 1 (targeted only), got %d", unread)
		}
		items, _, err := repo.GetsByUserID(ctx, alice, 1, 10, "", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != targeted.ID {
			t.Fatalf("expected only the targeted notification unread, got %+v", items)
		}
	})

	t.Run("second user is unaffected by the first user's read state", func(t *testing.T) {
		if got := countUserNotificationRows(t, db, bob, global.ID); got != 0 {
			t.Fatalf("expected bob to still have no delivery row, got %d", got)
		}
		unread, err := repo.GetUnreadCount(ctx, bob)
		if err != nil {
			t.Fatal(err)
		}
		if unread != 1 {
			t.Fatalf("expected bob unread count 1, got %d", unread)
		}
	})

	t.Run("dismissing a global hides it only for that user", func(t *testing.T) {
		if err := repo.Dismiss(ctx, bob, global.ID); err != nil {
			t.Fatal(err)
		}
		if got := countUserNotificationRows(t, db, bob, global.ID); got != 1 {
			t.Fatalf("expected exactly 1 materialized row for bob, got %d", got)
		}

		items, total, err := repo.GetsByUserID(ctx, bob, 1, 10, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if total != 0 || len(items) != 0 {
			t.Fatalf("expected the dismissed global to be hidden for bob, total=%d len=%d", total, len(items))
		}
		unread, err := repo.GetUnreadCount(ctx, bob)
		if err != nil {
			t.Fatal(err)
		}
		if unread != 0 {
			t.Fatalf("expected bob unread count 0, got %d", unread)
		}
		if _, err := repo.GetVisible(ctx, bob, global.ID); err != shared.ErrNotFound {
			t.Fatalf("expected ErrNotFound after dismiss, got %v", err)
		}

		if _, total, err = repo.GetsByUserID(ctx, alice, 1, 10, "", false); err != nil || total != 2 {
			t.Fatalf("expected alice still to see 2 notifications, total=%d err=%v", total, err)
		}
	})

	t.Run("global delivery stats count enabled users as recipients", func(t *testing.T) {
		stats, err := repo.StatsByNotificationID(ctx, global.ID)
		if err != nil {
			t.Fatal(err)
		}
		// alice + bob 启用，disabled 用户不计入
		if stats.RecipientCount != 2 {
			t.Fatalf("expected recipient count 2, got %d", stats.RecipientCount)
		}
		if stats.ReadCount != 1 {
			t.Fatalf("expected read count 1, got %d", stats.ReadCount)
		}
		if stats.UnreadCount != 1 {
			t.Fatalf("expected unread count 1, got %d", stats.UnreadCount)
		}
		if stats.DismissedCount != 1 {
			t.Fatalf("expected dismissed count 1, got %d", stats.DismissedCount)
		}

		targetedStats, err := repo.StatsByNotificationID(ctx, targeted.ID)
		if err != nil {
			t.Fatal(err)
		}
		if targetedStats.RecipientCount != 1 || targetedStats.UnreadCount != 1 {
			t.Fatalf("unexpected targeted stats: %+v", targetedStats)
		}
	})
}

// TestUserNotificationRepositoryGlobalExpiryAndMarkAll 验证过期全局通知不可见，
// 以及 MarkAllAsRead 会补建尚未实体化的全局通知关联行。
func TestUserNotificationRepositoryGlobalExpiryAndMarkAll(t *testing.T) {
	db := openNotificationTestDB(t)
	ctx := context.Background()
	notifRepo := NewNotificationRepository(db)
	repo := NewUserNotificationRepository(db)

	carol := createTestUser(t, db, "carol", true)

	expired := time.Now().Add(-time.Hour)
	expiredGlobal := &notification.Notification{Title: "Expired", IsGlobal: true, ExpireAt: &expired}
	if err := notifRepo.Create(ctx, expiredGlobal); err != nil {
		t.Fatal(err)
	}
	active := &notification.Notification{Title: "Active", IsGlobal: true}
	if err := notifRepo.Create(ctx, active); err != nil {
		t.Fatal(err)
	}

	items, total, err := repo.GetsByUserID(ctx, carol, 1, 10, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != active.ID {
		t.Fatalf("expected only the active global to be visible, total=%d items=%+v", total, items)
	}
	if _, err := repo.GetVisible(ctx, carol, expiredGlobal.ID); err != shared.ErrNotFound {
		t.Fatalf("expected expired global to be invisible, got %v", err)
	}

	if err := repo.MarkAllAsRead(ctx, carol); err != nil {
		t.Fatal(err)
	}
	if got := countUserNotificationRows(t, db, carol, active.ID); got != 1 {
		t.Fatalf("expected the active global to be materialized once, got %d", got)
	}
	if got := countUserNotificationRows(t, db, carol, expiredGlobal.ID); got != 0 {
		t.Fatalf("expected no row for the expired global, got %d", got)
	}
	unread, err := repo.GetUnreadCount(ctx, carol)
	if err != nil {
		t.Fatal(err)
	}
	if unread != 0 {
		t.Fatalf("expected unread count 0 after MarkAllAsRead, got %d", unread)
	}
}

// TestUserNotificationRepositoryConcurrentMaterialization 验证并发标记已读只产生一行。
func TestUserNotificationRepositoryConcurrentMaterialization(t *testing.T) {
	db := openNotificationTestDB(t)
	ctx := context.Background()
	notifRepo := NewNotificationRepository(db)
	repo := NewUserNotificationRepository(db)

	dave := createTestUser(t, db, "dave", true)
	global := &notification.Notification{Title: "Global", IsGlobal: true}
	if err := notifRepo.Create(ctx, global); err != nil {
		t.Fatal(err)
	}

	errs := make(chan error, 8)
	for range 8 {
		go func() { errs <- repo.MarkAsRead(ctx, dave, global.ID) }()
	}
	for range 8 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if got := countUserNotificationRows(t, db, dave, global.ID); got != 1 {
		t.Fatalf("expected exactly 1 row after concurrent marking, got %d", got)
	}
}
