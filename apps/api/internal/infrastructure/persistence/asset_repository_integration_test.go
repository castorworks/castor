package persistence

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

// assetTestSchema 让本文件的表与其它包的集成测试互不干扰。
const assetTestSchema = "asset_repo_test"

// 需要一次性的空库，绝不能指向开发库或生产库。
func openAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA IF NOT EXISTS " + assetTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := gorm.Open(postgres.Open(withSearchPath(dsn, assetTestSchema)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.AssetModel{}, &models.AssetReferenceModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE asset_references, assets RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func createTestAsset(t *testing.T, repo asset.Repository, objectKey, hash string, scope asset.AssetScope) *asset.Asset {
	t.Helper()
	item := &asset.Asset{Filename: objectKey, ObjectKey: objectKey, Hash: hash, Status: asset.StatusActive, Scope: scope, IsPublic: true}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	return item
}

// 哈希唯一索引是并发去重的裁决者；空哈希（不参与去重的资产）不受约束。
func TestAssetRepository_HashIsUniqueUnlessEmpty(t *testing.T) {
	ctx := context.Background()
	repo := NewAssetRepository(openAssetTestDB(t))
	createTestAsset(t, repo, "a.png", "h1", asset.ScopeLibrary)

	duplicate := &asset.Asset{Filename: "b.png", ObjectKey: "b.png", Hash: "h1"}
	if err := repo.Create(ctx, duplicate); err == nil {
		t.Fatal("a second asset with the same hash must be rejected")
	}
	createTestAsset(t, repo, "c.png", "", asset.ScopeLibrary)
	createTestAsset(t, repo, "d.png", "", asset.ScopeLibrary)
}

func TestAssetRepository_References(t *testing.T) {
	ctx := context.Background()
	repo := NewAssetRepository(openAssetTestDB(t))
	cover := createTestAsset(t, repo, "cover.png", "h-cover", asset.ScopeAttachment)
	shared0 := createTestAsset(t, repo, "shared.png", "h-shared", asset.ScopeLibrary)
	unused := createTestAsset(t, repo, "unused.png", "h-unused", asset.ScopeLibrary)

	article1Cover := asset.Reference{OwnerType: "article", OwnerID: 1, Field: "cover"}
	article1Banner := asset.Reference{OwnerType: "article", OwnerID: 1, Field: "banner"}
	article2Cover := asset.Reference{OwnerType: "article", OwnerID: 2, Field: "cover"}
	for _, link := range []struct {
		id  uint
		ref asset.Reference
	}{
		{cover.ID, article1Cover},
		{cover.ID, article1Cover}, // 重复登记必须幂等
		{shared0.ID, article1Banner},
		{shared0.ID, article2Cover},
	} {
		if err := repo.AddReference(ctx, link.id, link.ref, "v1"); err != nil {
			t.Fatalf("AddReference(%d, %+v) error = %v", link.id, link.ref, err)
		}
	}

	if count, _ := repo.CountReferences(ctx, cover.ID); count != 1 {
		t.Fatalf("CountReferences(cover) = %d, want 1 (idempotent add)", count)
	}
	// 重复登记只更新显示名；显示名属于引用，不属于资产。
	if err := repo.AddReference(ctx, cover.ID, article1Cover, "cover-final.png"); err != nil {
		t.Fatal(err)
	}
	if name, err := repo.GetReferenceName(ctx, cover.ID, article1Cover); err != nil || name != "cover-final.png" {
		t.Fatalf("GetReferenceName = %q, %v", name, err)
	}
	if _, err := repo.GetReferenceName(ctx, unused.ID, article1Cover); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("GetReferenceName(unreferenced) err = %v, want ErrNotFound", err)
	}
	attached, err := repo.GetAttached(ctx, asset.Field{OwnerType: "article", Name: "cover"}, []uint{1, 2, 3})
	if err != nil || len(attached) != 2 || attached[0].OwnerID != 1 || attached[0].Name != "cover-final.png" || attached[1].Asset.ID != shared0.ID {
		t.Fatalf("GetAttached = %+v, %v", attached, err)
	}
	refs, err := repo.GetReferences(ctx, shared0.ID)
	if err != nil || len(refs) != 2 || refs[0] != article1Banner || refs[1] != article2Cover {
		t.Fatalf("GetReferences(shared) = %+v, %v", refs, err)
	}
	referenced, _ := repo.GetReferencedIDs(ctx, []uint{cover.ID, shared0.ID, unused.ID})
	slices.Sort(referenced)
	if !slices.Equal(referenced, []uint{cover.ID, shared0.ID}) {
		t.Fatalf("GetReferencedIDs = %v", referenced)
	}
	if ids, _ := repo.GetIDsByReference(ctx, article1Cover); !slices.Equal(ids, []uint{cover.ID}) {
		t.Fatalf("GetIDsByReference = %v", ids)
	}
	owned, _ := repo.GetIDsByOwner(ctx, "article", 1)
	slices.Sort(owned)
	if !slices.Equal(owned, []uint{cover.ID, shared0.ID}) {
		t.Fatalf("GetIDsByOwner = %v", owned)
	}

	if err := repo.RemoveOwnerReferences(ctx, "article", 1); err != nil {
		t.Fatal(err)
	}
	if count, _ := repo.CountReferences(ctx, shared0.ID); count != 1 {
		t.Fatalf("other owners' references must survive, shared has %d", count)
	}
	if err := repo.RemoveReference(ctx, shared0.ID, article2Cover); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveReference(ctx, shared0.ID, article2Cover); err != nil {
		t.Fatalf("removing a missing reference must not fail: %v", err)
	}

	if err := repo.UpdateScope(ctx, cover.ID, asset.ScopeLibrary); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Get(ctx, cover.ID); got.Scope != asset.ScopeLibrary {
		t.Fatalf("UpdateScope: scope = %q", got.Scope)
	}
	if err := repo.BatchDelete(ctx, []uint{cover.ID, unused.ID}); err != nil {
		t.Fatal(err)
	}
	if _, total, _ := repo.Gets(ctx, 1, 10, ""); total != 1 {
		t.Fatalf("assets left = %d, want 1", total)
	}
}

func TestAssetRepository_GetOrphanAttachments(t *testing.T) {
	ctx := context.Background()
	db := openAssetTestDB(t)
	repo := NewAssetRepository(db)
	abandoned := createTestAsset(t, repo, "abandoned.png", "h-abandoned", asset.ScopeAttachment)
	fresh := createTestAsset(t, repo, "fresh.png", "h-fresh", asset.ScopeAttachment)
	inUse := createTestAsset(t, repo, "in-use.png", "h-in-use", asset.ScopeAttachment)
	library := createTestAsset(t, repo, "library.png", "h-library", asset.ScopeLibrary)
	if err := repo.AddReference(ctx, inUse.ID, asset.Reference{OwnerType: "user", OwnerID: 1, Field: "avatar"}, ""); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := db.Exec("UPDATE assets SET updated_at = ? WHERE id IN ?", old, []uint{abandoned.ID, inUse.ID, library.ID}).Error; err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	orphans, err := repo.GetOrphanAttachments(ctx, cutoff, 10)
	if err != nil || len(orphans) != 1 || orphans[0].ID != abandoned.ID {
		t.Fatalf("GetOrphanAttachments = %+v, %v; want only the abandoned upload (fresh=%d)", orphans, err, fresh.ID)
	}

	if err := repo.Touch(ctx, abandoned.ID); err != nil {
		t.Fatal(err)
	}
	if orphans, _ := repo.GetOrphanAttachments(ctx, cutoff, 10); len(orphans) != 0 {
		t.Fatalf("a touched attachment must get a new grace period, got %+v", orphans)
	}
}

// 免认证的公开列表只列资产库：业务附件（如用户头像）即使公开也不出现。
func TestAssetRepository_PublicListExcludesAttachments(t *testing.T) {
	ctx := context.Background()
	repo := NewAssetRepository(openAssetTestDB(t))
	createTestAsset(t, repo, "banner.png", "h-banner", asset.ScopeLibrary)
	createTestAsset(t, repo, "avatar.png", "h-avatar", asset.ScopeAttachment)

	items, total, err := repo.GetPublicAssets(ctx, 1, 10, "")
	if err != nil || total != 1 || items[0].ObjectKey != "banner.png" {
		t.Fatalf("GetPublicAssets = %+v, %d, %v", items, total, err)
	}
}
