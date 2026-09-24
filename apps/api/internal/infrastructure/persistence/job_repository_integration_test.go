package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// jobTestSchema 让本文件的表与其它包的集成测试互不干扰。
const jobTestSchema = "job_repo_test"

// 需要一次性的空库，绝不能指向开发库或生产库。
func openJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA IF NOT EXISTS " + jobTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}
	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, jobTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.ScheduledJobModel{}, &models.JobRunModel{}, &models.UserModel{}, &models.AuthorizationSessionModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE scheduled_jobs, job_runs, rbac_sessions, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestJobRepository_SettingsAreAddOnly(t *testing.T) {
	repo := NewJobRepository(openJobTestDB(t))
	ctx := context.Background()
	if err := repo.EnsureJobs(ctx, []job.Job{{Key: "a", Cron: "0 * * * *", IsEnabled: true}}); err != nil {
		t.Fatal(err)
	}
	item, err := repo.GetJob(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	item.Cron, item.IsEnabled = "*/5 * * * *", false
	if err := repo.UpdateJob(ctx, item); err != nil {
		t.Fatal(err)
	}
	// 重启时再次补齐：已有设置保留，新任务补上
	if err := repo.EnsureJobs(ctx, []job.Job{{Key: "a", Cron: "0 * * * *", IsEnabled: true}, {Key: "b", Cron: "0 3 * * *", IsEnabled: true}}); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Key != "a" || items[0].Cron != "*/5 * * * *" || items[0].IsEnabled || !items[1].IsEnabled {
		t.Fatalf("ListJobs() = %+v", items)
	}
	if _, err := repo.GetJob(ctx, "missing"); err != shared.ErrNotFound {
		t.Fatalf("GetJob(missing) err = %v", err)
	}
}

func TestJobRepository_RunsLatestStaleAndPrune(t *testing.T) {
	repo := NewJobRepository(openJobTestDB(t))
	ctx := context.Background()
	base := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	create := func(key string, offset time.Duration, status job.Status) *job.Run {
		t.Helper()
		run := &job.Run{JobKey: key, Trigger: job.TriggerSchedule, Status: status, StartedAt: base.Add(offset)}
		if err := repo.CreateRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	for i := 0; i < 5; i++ {
		create("a", time.Duration(i)*time.Minute, job.StatusSucceeded)
	}
	lastA := create("a", 10*time.Minute, job.StatusRunning)
	staleB := create("b", 0, job.StatusRunning)
	freshB := create("b", 2*time.Hour, job.StatusRunning)

	if err := repo.FinishRun(ctx, lastA.ID, job.StatusSucceeded, 42, "", base.Add(11*time.Minute)); err != nil {
		t.Fatal(err)
	}
	latest, err := repo.LatestRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := latest["a"]; got.ID != lastA.ID || got.Affected != 42 || got.Status != job.StatusSucceeded || got.FinishedAt == nil {
		t.Fatalf("latest a = %+v", got)
	}
	if latest["b"].ID != freshB.ID {
		t.Fatalf("latest b = %+v", latest["b"])
	}

	// 只收尾 before 之前开始的 RUNNING 记录
	n, err := repo.FailStaleRuns(ctx, "b", "interrupted", base.Add(time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("FailStaleRuns() = %d, %v", n, err)
	}
	runs, total, err := repo.ListRuns(ctx, 1, 20, "started_at desc")
	if err != nil || total != 8 {
		t.Fatalf("ListRuns() total = %d, %v", total, err)
	}
	status := map[uint]job.Run{}
	for _, run := range runs {
		status[run.ID] = run
	}
	if status[staleB.ID].Status != job.StatusFailed || status[staleB.ID].Message != "interrupted" || status[freshB.ID].Status != job.StatusRunning {
		t.Fatalf("stale handling wrong: stale=%+v fresh=%+v", status[staleB.ID], status[freshB.ID])
	}

	if err := repo.PruneRuns(ctx, "a", 2); err != nil {
		t.Fatal(err)
	}
	_, total, err = repo.ListRuns(ctx, 1, 20, "")
	if err != nil || total != 4 {
		t.Fatalf("after prune total = %d, %v; want 2 of a + 2 of b", total, err)
	}
	latest, _ = repo.LatestRuns(ctx)
	if latest["a"].ID != lastA.ID {
		t.Fatal("prune must keep the newest runs")
	}
}

func TestPurgeSessionsKeepsRecentAndLive(t *testing.T) {
	db := openJobTestDB(t)
	ctx := context.Background()
	u := models.UserModel{Username: "erin", AccountSource: "INTERNAL"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAuthorizationRepository(db)
	now := time.Now()
	create := func(id string, expires time.Time) {
		t.Helper()
		s := &permission.AuthorizationSession{ID: id, UserID: u.ID, Username: u.Username, CreatedAt: now.Add(-90 * 24 * time.Hour), ExpiresAt: expires, Version: 1}
		if err := repo.CreateSession(ctx, s, nil); err != nil {
			t.Fatal(err)
		}
	}
	create("00000000-0000-0000-0000-000000000001", now.Add(time.Hour))        // live
	create("00000000-0000-0000-0000-000000000002", now.Add(-60*24*time.Hour)) // long expired
	create("00000000-0000-0000-0000-000000000003", now.Add(-24*time.Hour))    // recently expired
	create("00000000-0000-0000-0000-000000000004", now.Add(time.Hour))        // revoked long ago
	create("00000000-0000-0000-0000-000000000005", now.Add(time.Hour))        // revoked just now
	if err := db.Model(&models.AuthorizationSessionModel{}).Where("id = ?", "00000000-0000-0000-0000-000000000004").
		UpdateColumn("revoked_at", now.Add(-60*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.RevokeSession(ctx, "00000000-0000-0000-0000-000000000005"); err != nil {
		t.Fatal(err)
	}

	n, err := repo.PurgeSessions(ctx, now.Add(-30*24*time.Hour))
	if err != nil || n != 2 {
		t.Fatalf("PurgeSessions() = %d, %v; want 2", n, err)
	}
	var left []string
	if err := db.Model(&models.AuthorizationSessionModel{}).Order("id").Pluck("id", &left).Error; err != nil {
		t.Fatal(err)
	}
	if len(left) != 3 || left[0] != "00000000-0000-0000-0000-000000000001" || left[1] != "00000000-0000-0000-0000-000000000003" || left[2] != "00000000-0000-0000-0000-000000000005" {
		t.Fatalf("remaining sessions = %v", left)
	}
}
