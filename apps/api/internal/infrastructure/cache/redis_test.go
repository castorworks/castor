package cache

import (
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestClaimNamespace(t *testing.T) {
	ctx := t.Context()
	server, client := newTestRedis(t)

	// 首次认领登记 owner，且不设过期。
	if err := ClaimNamespace(ctx, client, "tenant-a", "db-a"); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if got, _ := server.Get("tenant-a:instance"); got != "db-a" {
		t.Fatalf("owner = %q, want db-a", got)
	}
	if ttl := server.TTL("tenant-a:instance"); ttl != 0 {
		t.Fatalf("owner key must not expire, ttl = %v", ttl)
	}

	// 同一部署的其他副本、重启：同一 owner 再次认领成功。
	if err := ClaimNamespace(ctx, client, "tenant-a", "db-a"); err != nil {
		t.Fatalf("replica claim: %v", err)
	}

	// 另一套部署误用同一 InstanceID：拒绝，并说明如何处理。
	err := ClaimNamespace(ctx, client, "tenant-a", "db-b")
	if err == nil || !strings.Contains(err.Error(), "InstanceID") {
		t.Fatalf("claim by another deployment must fail with guidance, got %v", err)
	}

	// 不同 InstanceID 互不影响。
	if err := ClaimNamespace(ctx, client, "tenant-b", "db-b"); err != nil {
		t.Fatalf("claim of a different namespace: %v", err)
	}

	// 运维删除旧部署的登记后，新部署可以接手。
	server.Del("tenant-a:instance")
	if err := ClaimNamespace(ctx, client, rediskey.Namespace("tenant-a"), "db-b"); err != nil {
		t.Fatalf("claim after the old owner was removed: %v", err)
	}
}
