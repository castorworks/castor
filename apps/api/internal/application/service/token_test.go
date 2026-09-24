package service

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newBlacklistTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestTokenBlacklistService_AddAndCheck(t *testing.T) {
	ctx := t.Context()
	server, client := newBlacklistTestRedis(t)
	svc := NewTokenBlacklistService(client, "castor")

	if svc.IsBlacklisted(ctx, "token-a") {
		t.Fatal("a token that was never retired must not be blacklisted")
	}
	if err := svc.AddToBlacklist(ctx, "token-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if !svc.IsBlacklisted(ctx, "token-a") {
		t.Fatal("retired token must be blacklisted")
	}
	if !server.Exists("castor:token:blacklist:token-a") {
		t.Fatalf("blacklist entry must live under the instance namespace, keys = %v", server.Keys())
	}
	if ttl := server.TTL("castor:token:blacklist:token-a"); ttl != time.Minute {
		t.Fatalf("blacklist entry must expire with the token, ttl = %v", ttl)
	}
}

// 共用 Redis 的另一套部署注销了同一个 token 字符串，不影响本实例。
func TestTokenBlacklistService_IsolatedPerInstance(t *testing.T) {
	ctx := t.Context()
	_, client := newBlacklistTestRedis(t)
	tenantA := NewTokenBlacklistService(client, "tenant-a")
	tenantB := NewTokenBlacklistService(client, "tenant-b")

	if err := tenantA.AddToBlacklist(ctx, "shared-token", time.Minute); err != nil {
		t.Fatal(err)
	}
	if tenantB.IsBlacklisted(ctx, "shared-token") {
		t.Fatal("a token retired by another instance must not be blacklisted here")
	}
}

// Redis 故障时拒绝访问，防止已注销的 token 被放行。
func TestTokenBlacklistService_FailSafe(t *testing.T) {
	server, client := newBlacklistTestRedis(t)
	svc := NewTokenBlacklistService(client, "castor")
	server.Close()

	if !svc.IsBlacklisted(t.Context(), "token-a") {
		t.Fatal("a Redis failure must deny access")
	}
}
