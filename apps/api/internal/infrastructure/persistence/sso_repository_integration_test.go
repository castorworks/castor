package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/sso"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
)

func TestSSORepository_ConstraintsAndCascades(t *testing.T) {
	db := openMFATestDB(t)
	if err := db.AutoMigrate(&models.OIDCProviderModel{}, &models.UserIdentityModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE TABLE user_identities, oidc_providers RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	repo := NewSSORepository(db)
	ctx := context.Background()
	alice := models.UserModel{Username: "sso-alice", AccountSource: "INTERNAL"}
	bob := models.UserModel{Username: "sso-bob", AccountSource: "INTERNAL"}
	if err := db.Create(&alice).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&bob).Error; err != nil {
		t.Fatal(err)
	}

	p := &sso.Provider{Code: "corp", Name: shared.Text("Corp", "公司", "会社", "회사"), Issuer: "https://idp.test", ClientID: "c",
		ClientSecretCiphertext: "v1:x", Scopes: []string{"openid", "email"}, UsernameClaim: "preferred_username", IsEnabled: true}
	if err := repo.SaveProvider(ctx, p); err != nil || p.ID == 0 {
		t.Fatalf("SaveProvider = %+v, %v", p, err)
	}
	if got, err := repo.GetProviderByCode(ctx, "corp"); err != nil || got.Name.Ja != "会社" || len(got.Scopes) != 2 {
		t.Fatalf("GetProviderByCode = %+v, %v", got, err)
	}
	dup := &sso.Provider{Code: "corp", Name: p.Name, Issuer: "https://other.test", ClientID: "c", ClientSecretCiphertext: "v1:y", Scopes: []string{"openid"}}
	if err := repo.SaveProvider(ctx, dup); !errors.Is(err, shared.ErrDuplicate) {
		t.Fatalf("duplicate provider code err = %v", err)
	}

	if err := repo.CreateIdentity(ctx, &sso.Identity{UserID: alice.ID, ProviderID: p.ID, Subject: "s-1", Email: "a@idp.test"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateIdentity(ctx, &sso.Identity{UserID: bob.ID, ProviderID: p.ID, Subject: "s-1"}); !errors.Is(err, shared.ErrDuplicate) {
		t.Fatalf("one provider account linked to two users err = %v", err)
	}
	if err := repo.CreateIdentity(ctx, &sso.Identity{UserID: alice.ID, ProviderID: p.ID, Subject: "s-2"}); !errors.Is(err, shared.ErrDuplicate) {
		t.Fatalf("two accounts of one provider on one user err = %v", err)
	}
	found, err := repo.FindIdentity(ctx, p.ID, "s-1")
	if err != nil || found.UserID != alice.ID {
		t.Fatalf("FindIdentity = %+v, %v", found, err)
	}
	if err := repo.TouchIdentity(ctx, found.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := repo.ListIdentities(ctx, alice.ID); len(list) != 1 || list[0].LastLoginAt == nil {
		t.Fatalf("ListIdentities = %+v", list)
	}

	if err := db.Delete(&models.UserModel{}, alice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindIdentity(ctx, p.ID, "s-1"); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("identities must be deleted with the user, err = %v", err)
	}
	if err := repo.CreateIdentity(ctx, &sso.Identity{UserID: bob.ID, ProviderID: p.ID, Subject: "s-3"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteProvider(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := repo.ListIdentities(ctx, bob.ID); len(list) != 0 {
		t.Fatalf("identities must be deleted with the provider: %+v", list)
	}
}
