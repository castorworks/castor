package database

import (
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// migrationTestSchema 让本文件的表与其它包的集成测试互不干扰。
const migrationTestSchema = "migration_test"

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
func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// 每次运行都从干净的 schema 开始，保证可重复执行。
	if err := admin.Exec("DROP SCHEMA IF EXISTS " + migrationTestSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA " + migrationTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, migrationTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// TestMigrateCreatesUserContactPartialUniqueIndexes 验证联系方式“可以为空、一旦填写则全局唯一”
// 由部分唯一索引真正强制。
func TestMigrateCreatesUserContactPartialUniqueIndexes(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	// 迁移必须可重复执行（init-db 允许多次运行）。
	if err := Migrate(db); err != nil {
		t.Fatalf("repeat Migrate() error = %v", err)
	}

	var applied []SchemaMigration
	if err := db.Order("version ASC").Find(&applied).Error; err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 || applied[len(applied)-1].Version != CurrentSchemaVersion {
		t.Fatalf("applied versions = %+v, want the newest to be %d", applied, CurrentSchemaVersion)
	}

	for _, index := range userContactUniqueIndexes {
		var count int64
		if err := db.Raw(
			"SELECT count(*) FROM pg_indexes WHERE schemaname = ? AND tablename = 'users' AND indexname = ?",
			migrationTestSchema, index.Name,
		).Scan(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("index %s is missing after migration", index.Name)
		}
	}

	// 多个用户可以同时没有联系方式。
	for _, username := range []string{"empty-a", "empty-b"} {
		if err := db.Create(&models.UserModel{Username: username, AccountSource: "INTERNAL"}).Error; err != nil {
			t.Fatalf("users without contacts must coexist, creating %s failed: %v", username, err)
		}
	}

	// 非空联系方式全局唯一。
	if err := db.Create(&models.UserModel{
		Username: "owner", AccountSource: "INTERNAL",
		Email: "dup@example.com", EmailVerified: true, Mobile: "13800138000", MobileVerified: true,
	}).Error; err != nil {
		t.Fatalf("first contact owner must be accepted: %v", err)
	}
	if err := db.Create(&models.UserModel{
		Username: "duplicate-email", AccountSource: "INTERNAL", Email: "dup@example.com",
	}).Error; err == nil {
		t.Fatal("a duplicate non-empty email must be rejected by idx_users_email_unique")
	}
	if err := db.Create(&models.UserModel{
		Username: "duplicate-mobile", AccountSource: "INTERNAL", Mobile: "13800138000",
	}).Error; err == nil {
		t.Fatal("a duplicate non-empty mobile must be rejected by idx_users_mobile_unique")
	}

	// 不同的非空值互不冲突。
	if err := db.Create(&models.UserModel{
		Username: "other", AccountSource: "INTERNAL", Email: "other@example.com", Mobile: "13800138001",
	}).Error; err != nil {
		t.Fatalf("distinct contacts must be accepted: %v", err)
	}
}

// 而运营改过的内容与自建的字典项原样保留。
func TestSeedDictionaryReconcilesExistingDatabase(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// 模拟"代码里刚加了一个默认项"：库里还没有它。
	const added = "AUDIT_LOG_CLEANUP"
	if err := db.Where("type_code = ? AND value = ?", "audit_log_type", added).
		Delete(&models.DictItemModel{}).Error; err != nil {
		t.Fatal(err)
	}
	// 系统标记由对账补齐：先清掉，确认会被写回。
	if err := db.Model(&models.DictItemModel{}).Where("1 = 1").UpdateColumn("is_system", false).Error; err != nil {
		t.Fatal(err)
	}
	// 运营的改动：改标签、停用、把一个类型设为公开。
	const customLabel = "Operator renamed this"
	if err := db.Model(&models.DictItemModel{}).
		Where("type_code = ? AND value = ?", "audit_log_type", "LOGIN").
		Updates(map[string]any{"label_en": customLabel, "is_enabled": false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.DictTypeModel{}).Where("code = ?", "login_method").
		UpdateColumn("is_public", true).Error; err != nil {
		t.Fatal(err)
	}
	// 运营自建的字典项不属于种子。
	custom := models.DictItemModel{
		TypeCode: "priority", Value: "CUSTOM", IsEnabled: true,
		LabelEn: "Custom", LabelZh: "自定义", LabelJa: "カスタム", LabelKo: "사용자 정의",
	}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatal(err)
	}

	if err := SeedDictionary(db); err != nil {
		t.Fatalf("SeedDictionary() error = %v", err)
	}
	if err := SeedDictionary(db); err != nil {
		t.Fatalf("repeat SeedDictionary() error = %v", err)
	}

	var restored models.DictItemModel
	if err := db.Where("type_code = ? AND value = ?", "audit_log_type", added).First(&restored).Error; err != nil {
		t.Fatalf("missing default item was not restored: %v", err)
	}
	if !restored.IsSystem {
		t.Error("restored default item must be a system item")
	}

	var login models.DictItemModel
	if err := db.Where("type_code = ? AND value = ?", "audit_log_type", "LOGIN").First(&login).Error; err != nil {
		t.Fatal(err)
	}
	if login.LabelEn != customLabel || login.IsEnabled {
		t.Errorf("operator edits were overwritten: label_en=%q is_enabled=%v", login.LabelEn, login.IsEnabled)
	}
	if !login.IsSystem {
		t.Error("pre-existing seed rows must be marked as system items")
	}

	var loginMethod models.DictTypeModel
	if err := db.Where("code = ?", "login_method").First(&loginMethod).Error; err != nil {
		t.Fatal(err)
	}
	if !loginMethod.IsPublic {
		t.Error("operator's is_public choice was overwritten")
	}

	var kept models.DictItemModel
	if err := db.First(&kept, custom.ID).Error; err != nil {
		t.Fatal(err)
	}
	if kept.IsSystem {
		t.Error("operator-created items must not become system items")
	}

	var total int64
	if err := db.Model(&models.DictItemModel{}).Where("type_code = ? AND value = ?", "audit_log_type", "LOGIN").
		Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("reconcile duplicated an existing item: count = %d", total)
	}
}

// TestDeploymentIdentityIsStable 验证每个库恰好一个部署标识，
// 重复执行 init-db 不会换掉它（换了就等于宣称自己是另一套部署，Redis 命名空间认领会失败）。
func TestDeploymentIdentityIsStable(t *testing.T) {
	db := openMigrationTestDB(t)
	ctx := t.Context()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	first, err := DeploymentID(ctx, db)
	if err != nil {
		t.Fatalf("DeploymentID() error = %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("repeat Migrate() error = %v", err)
	}
	second, err := DeploymentID(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("deployment identity changed across init-db runs: %s -> %s", first, second)
	}

	var rows int64
	if err := db.Raw("SELECT count(*) FROM deployment").Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("deployment has %d rows, want exactly 1", rows)
	}
	if err := db.Exec("INSERT INTO deployment (singleton, id) VALUES (false, gen_random_uuid())").Error; err == nil {
		t.Fatal("a second deployment identity row must be rejected")
	}
}

// 比程序更新的库（例如回滚到旧镜像）必须拒绝迁移，而不是在不认识的结构上继续写。
func TestMigrateRejectsNewerSchema(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := db.Create(&SchemaMigration{Version: CurrentSchemaVersion + 1, Name: "from_the_future", AppliedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err == nil || !strings.Contains(err.Error(), "newer than application") {
		t.Fatalf("Migrate() on a newer database error = %v", err)
	}
}

// 有成员的部门、被业务记录引用的资产都删不掉：外键是并发下的最终防线。
func TestInitialSchemaRestrictsDeletes(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	department := models.DepartmentModel{Code: "ops", Name: "Ops", IsEnabled: true}
	if err := db.Create(&department).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserModel{Username: "member", AccountSource: "INTERNAL", DepartmentID: &department.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&department).Error; err == nil {
		t.Fatal("deleting a department that still has members must be rejected by " + usersDepartmentForeignKey)
	}

	item := models.AssetModel{ObjectKey: "a.png", Filename: "a.png", Status: asset.StatusActive, Scope: asset.ScopeAttachment}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AssetReferenceModel{AssetID: item.ID, OwnerType: "notification", OwnerID: 1, Field: "attachments"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&item).Error; err == nil {
		t.Fatal("deleting a referenced asset must be rejected by fk_asset_references_asset")
	}
}
