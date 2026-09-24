package persistence

import (
	"context"
	"os"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// mfaTestSchema 让本文件的表与其它包的集成测试互不干扰。
const mfaTestSchema = "mfa_repo_test"

func openMFATestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA IF NOT EXISTS " + mfaTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}
	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, mfaTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UserModel{}, &models.UserTOTPModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE user_totp, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMFARepository_Lifecycle(t *testing.T) {
	db := openMFATestDB(t)
	repo := NewMFARepository(db)
	ctx := context.Background()
	u := models.UserModel{Username: "mfa-user", AccountSource: "INTERNAL"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := repo.Get(ctx, u.ID); err != shared.ErrNotFound {
		t.Fatalf("Get before setup err = %v", err)
	}
	if ok, err := repo.SavePending(ctx, u.ID, "v1:first"); !ok || err != nil {
		t.Fatalf("SavePending = %v, %v", ok, err)
	}
	if ok, err := repo.SavePending(ctx, u.ID, "v1:second"); !ok || err != nil {
		t.Fatalf("an unconfirmed secret must be replaceable: %v, %v", ok, err)
	}
	if ok, err := repo.Enable(ctx, u.ID, []string{"h1", "h2", "h3"}, 100); !ok || err != nil {
		t.Fatalf("Enable = %v, %v", ok, err)
	}
	if ok, _ := repo.Enable(ctx, u.ID, []string{"x"}, 200); ok {
		t.Fatal("enabling twice must not overwrite")
	}
	if ok, err := repo.SavePending(ctx, u.ID, "v1:attacker"); ok || err != nil {
		t.Fatalf("an enabled secret must not be replaced by a new setup: %v, %v", ok, err)
	}
	item, err := repo.Get(ctx, u.ID)
	if err != nil || !item.Enabled || item.SecretCiphertext != "v1:second" || item.LastUsedStep != 100 || len(item.RecoveryCodeHashes) != 3 || item.EnabledAt == nil {
		t.Fatalf("Get = %+v, %v", item, err)
	}

	if ok, _ := repo.AdvanceStep(ctx, u.ID, 100); ok {
		t.Fatal("the same step must not be accepted twice")
	}
	// 同一时间步并发提交：只能有一个成功。
	var wg sync.WaitGroup
	wins := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.AdvanceStep(ctx, u.ID, 101)
			wins <- ok && err == nil
		}()
	}
	wg.Wait()
	close(wins)
	accepted := 0
	for ok := range wins {
		if ok {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("concurrent submissions of one code accepted %d times, want 1", accepted)
	}

	// 恢复码并发消费：同一个码只能用一次。
	wins = make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.ConsumeRecoveryCode(ctx, u.ID, "h2")
			wins <- ok && err == nil
		}()
	}
	wg.Wait()
	close(wins)
	accepted = 0
	for ok := range wins {
		if ok {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("recovery code consumed %d times, want 1", accepted)
	}
	if item, _ := repo.Get(ctx, u.ID); len(item.RecoveryCodeHashes) != 2 {
		t.Fatalf("remaining hashes = %v", item.RecoveryCodeHashes)
	}
	if err := repo.ReplaceRecoveryCodes(ctx, u.ID, []string{"n1"}); err != nil {
		t.Fatal(err)
	}
	if enabled, err := repo.EnabledAmong(ctx, []uint{u.ID, 999}); err != nil || !enabled[u.ID] || enabled[999] {
		t.Fatalf("EnabledAmong = %v, %v", enabled, err)
	}

	// 删除用户时一并删除两步验证设置。
	if err := db.Delete(&models.UserModel{}, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, u.ID); err != shared.ErrNotFound {
		t.Fatalf("TOTP settings must be deleted with the user, err = %v", err)
	}
}
