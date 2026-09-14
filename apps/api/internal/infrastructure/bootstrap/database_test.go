package bootstrap

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hyperits/gosuite/security/hash"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/infrastructure/database"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// Requires a disposable, empty database, never a developer or production database.
func TestInitializeConcurrentAndRepeat(t *testing.T) {
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "InitialPassword123!")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := Initialize(ctx, db); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&models.UserModel{}).Where("username = ?", "system").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("system count=%d err=%v", count, err)
	}
	var admin models.UserModel
	if err := db.Where("username = ?", "system").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if !hash.BcryptMatchPassword("InitialPassword123!", admin.Password) {
		t.Fatal("initial password mismatch")
	}
	t.Setenv("CASTOR_DEFAULT_ADMIN_PASSWORD", "ChangedPassword123!")
	if _, err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&admin, admin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !hash.BcryptMatchPassword("InitialPassword123!", admin.Password) {
		t.Fatal("repeat initialization reset password")
	}
	if err := db.Model(&models.UserRoleModel{}).Where("user_id = ?", admin.ID).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("role count=%d err=%v", count, err)
	}
	var migration database.SchemaMigration
	if err := db.First(&migration, database.CurrentSchemaVersion).Error; err != nil {
		t.Fatalf("schema migration record missing: %v", err)
	}
	if migration.Name != "initial_schema" && migration.Name != "legacy_baseline" {
		t.Fatalf("unexpected schema migration name: %q", migration.Name)
	}

	var menus []models.MenuModel
	if err := db.Find(&menus).Error; err != nil {
		t.Fatal(err)
	}
	byCode := make(map[string]models.MenuModel, len(menus))
	for _, item := range menus {
		byCode[item.Code] = item
	}
	pageParents := map[string]string{
		"page_users":               "system_access",
		"page_roles":               "system_access",
		"page_resources":           "system_access",
		"page_authorization":       "system_access",
		"page_menus":               "system_access",
		"page_settings":            "system_configuration",
		"page_dictionaries":        "system_configuration",
		"page_assets":              "system_operations",
		"page_admin_notifications": "system_operations",
		"page_audit_logs":          "system_audit",
		"page_login_histories":     "system_audit",
	}
	for pageCode, parentCode := range pageParents {
		page, pageOK := byCode[pageCode]
		parent, parentOK := byCode[parentCode]
		if !pageOK || !parentOK || page.ParentID == nil || *page.ParentID != parent.ID {
			t.Fatalf("menu %s is not grouped under %s", pageCode, parentCode)
		}
	}
}
