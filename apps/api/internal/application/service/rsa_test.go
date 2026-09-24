package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/redis/go-redis/v9"
)

func TestRsaServiceGetPublicKeyRegeneratesUnusableCurrentKeyPair(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	oldSvc := NewRsaService(client, "castor", RsaKeyConfig{PrivateKeySecret: "old-rsa-secret"}).(*rsaService)

	oldPublicKey, err := oldSvc.GenerateKeyPair(ctx)
	if err != nil {
		t.Fatalf("generate old key pair: %v", err)
	}

	keyPairID, err := client.Get(ctx, "castor:"+constant.RSA_CURRENT_KEY_PAIR_ID).Result()
	if err != nil {
		t.Fatalf("get current key pair id: %v", err)
	}

	newSvc := NewRsaService(client, "castor", RsaKeyConfig{PrivateKeySecret: "new-rsa-secret"}).(*rsaService)

	publicKey, err := newSvc.GetPublicKey(ctx)
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}
	if publicKey == oldPublicKey {
		t.Fatal("expected unusable key pair to be replaced")
	}

	newKeyPairID, err := client.Get(ctx, "castor:"+constant.RSA_CURRENT_KEY_PAIR_ID).Result()
	if err != nil {
		t.Fatalf("get regenerated key pair id: %v", err)
	}
	if newKeyPairID == keyPairID {
		t.Fatalf("expected current key pair id to change, still %q", keyPairID)
	}

	if _, err := newSvc.decryptUsablePrivateKey(ctx, newKeyPairID); err != nil {
		t.Fatalf("expected regenerated private key to be usable: %v", err)
	}

	oldPrivateKey, err := client.Get(ctx, fmt.Sprintf("castor:%s:private", keyPairID)).Result()
	if err != nil {
		t.Fatalf("expected old private key to remain for expiry cleanup: %v", err)
	}
	if oldPrivateKey == "" {
		t.Fatal("expected old private key value")
	}
}
