package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
)

// statsTestSchema 让本文件的表与其它包的集成测试互不干扰。
const statsTestSchema = "stats_repo_test"

// 需要一次性的空库，绝不能指向开发库或生产库。
func openStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA IF NOT EXISTS " + statsTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}
	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, statsTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UserModel{}, &models.LoginHistoryModel{}, &models.AuthorizationSessionModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE rbac_sessions, login_histories, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCountByMonthFillsGapsAndFilters(t *testing.T) {
	db := openStatsTestDB(t)
	ctx := context.Background()
	var now time.Time
	if err := db.Raw("SELECT date_trunc('month', now())").Scan(&now).Error; err != nil {
		t.Fatal(err)
	}
	thisMonth := now.Add(time.Hour)
	twoMonthsAgo := now.AddDate(0, -2, 0).Add(time.Hour)
	longAgo := now.AddDate(-1, 0, 0)
	rows := []models.LoginHistoryModel{
		{CreatedAt: thisMonth, Username: "a", Success: true},
		{CreatedAt: thisMonth, Username: "a", Success: true},
		{CreatedAt: thisMonth, Username: "a", Success: false}, // 失败的尝试不算登录
		{CreatedAt: twoMonthsAgo, Username: "b", Success: true},
		{CreatedAt: longAgo, Username: "c", Success: true}, // 超出窗口
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewLoginHistoryRepository(db).CountSuccessfulByMonth(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("CountSuccessfulByMonth(3) returned %d months: %+v", len(got), got)
	}
	want := []int64{1, 0, 2}
	for i, month := range got {
		if month.Count != want[i] {
			t.Fatalf("month %d (%s) count = %d, want %d; all = %+v", i, month.Month, month.Count, want[i], got)
		}
	}
	if got[2].Month != now.Format("2006-01") {
		t.Fatalf("last month = %s, want the current month %s", got[2].Month, now.Format("2006-01"))
	}

	// 数据范围条件追加在被统计的表上：只统计部门 5 的用户（这里没有），登录数应全部为 0。
	scoped, err := NewLoginHistoryRepository(db).CountSuccessfulByMonth(ctx, 3, *query.UserScope("user_id", false, 999, []uint{5}))
	if err != nil {
		t.Fatal(err)
	}
	for _, month := range scoped {
		if month.Count != 0 {
			t.Fatalf("scoped trend must exclude users outside the scope: %+v", scoped)
		}
	}
	if today, err := NewLoginHistoryRepository(db).CountSuccessfulToday(ctx, *query.NewOption("username = ?", "b")); err != nil || today != 0 {
		t.Fatalf("CountSuccessfulToday with a condition = %d, %v", today, err)
	}

	users, err := NewUserRepository(db).CountCreatedByMonth(ctx, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 6 {
		t.Fatalf("CountCreatedByMonth(6) returned %d months", len(users))
	}
	for _, month := range users {
		if month.Count != 0 {
			t.Fatalf("empty users table must yield zero-filled months, got %+v", users)
		}
	}
}

func TestListActiveSessionsExcludesRevokedAndExpired(t *testing.T) {
	db := openStatsTestDB(t)
	ctx := context.Background()
	u := models.UserModel{Username: "dora", AccountSource: "INTERNAL"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAuthorizationRepository(db)
	now := time.Now()
	create := func(id string, expires time.Time) {
		t.Helper()
		s := &permission.AuthorizationSession{ID: id, UserID: u.ID, Username: u.Username, IpAddr: "10.0.0.1", CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: expires, Version: 1}
		if err := repo.CreateSession(ctx, s, nil); err != nil {
			t.Fatal(err)
		}
	}
	create("00000000-0000-0000-0000-000000000001", now.Add(time.Hour))
	create("00000000-0000-0000-0000-000000000002", now.Add(time.Hour))
	create("00000000-0000-0000-0000-000000000003", now.Add(-time.Minute))
	if err := repo.RevokeSession(ctx, "00000000-0000-0000-0000-000000000002"); err != nil {
		t.Fatal(err)
	}

	items, total, err := repo.ListActiveSessions(ctx, 1, 10, "created_at desc")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("ListActiveSessions() = %+v (total %d), want only the live session", items, total)
	}
	if items[0].IpAddr != "10.0.0.1" || items[0].Username != "dora" || !items[0].LastActiveAt.Equal(items[0].CreatedAt) {
		t.Fatalf("session origin not persisted: %+v", items[0])
	}

	later := now.Truncate(time.Microsecond)
	if err := repo.TouchSession(ctx, items[0].ID, later); err != nil {
		t.Fatal(err)
	}
	touched, err := repo.GetSession(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !touched.LastActiveAt.Equal(later) {
		t.Fatalf("LastActiveAt = %v, want %v", touched.LastActiveAt, later)
	}
}
