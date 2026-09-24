package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=none dbname=none sslmode=disable"}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSafeOrderUsesModelColumnAllowlist(t *testing.T) {
	db := dryRunDB(t)
	got, err := safeOrder[models.UserModel](db, "username asc, ID desc")
	if err != nil || got != "username ASC, id DESC" {
		t.Fatalf("safeOrder() = %q, %v", got, err)
	}
	for _, order := range []string{
		"password desc",
		"id; DROP TABLE users",
		"(SELECT 1)",
		"no_such_column",
		"id desc, CASE WHEN 1=1 THEN id END",
	} {
		if _, err := safeOrder[models.UserModel](db, order); !errors.Is(err, query.ErrInvalidOrder) {
			t.Errorf("safeOrder(%q) error = %v, want ErrInvalidOrder", order, err)
		}
	}
}

func TestRepositoriesRejectRawOrderAtPersistenceBoundary(t *testing.T) {
	db := dryRunDB(t)
	ctx := context.Background()
	payload := "created_at DESC; DROP TABLE assets--"

	if _, _, err := NewAssetRepository(db).GetPublicAssets(ctx, 1, 10, payload); !errors.Is(err, query.ErrInvalidOrder) {
		t.Errorf("asset repository error = %v, want ErrInvalidOrder", err)
	}
	if _, _, err := NewUserRepository(db).Gets(ctx, 1, 10, "password DESC"); !errors.Is(err, query.ErrInvalidOrder) {
		t.Errorf("user repository error = %v, want ErrInvalidOrder", err)
	}
	if _, _, err := NewUserNotificationRepository(db).GetsByUserID(ctx, 1, 1, 10, payload, false); !errors.Is(err, query.ErrInvalidOrder) {
		t.Errorf("user notification repository error = %v, want ErrInvalidOrder", err)
	}
}
