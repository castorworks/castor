package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/infrastructure/database"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/hyperits/gosuite/db/redis"
	rds "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ownerKey 记录命名空间归属哪套部署（值为该部署数据库里的部署标识），永不过期。
const ownerKey = "instance"

// claimTimeout 启动时认领命名空间的时限
const claimTimeout = 10 * time.Second

// NewNamespace 返回本实例在共享 Redis 中的 key 前缀
func NewNamespace() rediskey.Namespace {
	return rediskey.Namespace(config.C.General.InstanceID)
}

// NewRedis 创建 Redis 客户端，并认领本实例的命名空间：另一套部署（数据库不同）
// 已在用同一个 InstanceID 时拒绝启动，而不是让两边悄悄读写对方的 key。
func NewRedis(ns rediskey.Namespace, db *gorm.DB) (rds.UniversalClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), claimTimeout)
	defer cancel()
	owner, err := database.DeploymentID(ctx, db)
	if err != nil {
		return nil, err
	}
	client, err := redis.NewClient(&config.C.Redis)
	if err != nil {
		return nil, err
	}
	rdb := client.UniversalClient()
	if err := ClaimNamespace(ctx, rdb, ns, owner); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}

// ClaimNamespace 把命名空间登记到 owner 名下；已被其他 owner 登记时返回错误。
// 同一部署的多个副本 owner 相同，可以并发认领。
func ClaimNamespace(ctx context.Context, rdb rds.UniversalClient, ns rediskey.Namespace, owner string) error {
	key := ns.Key(ownerKey)
	for range 2 {
		claimed, err := rdb.SetNX(ctx, key, owner, 0).Result()
		if err != nil {
			return fmt.Errorf("claim redis namespace %q: %w", ns, err)
		}
		if claimed {
			return nil
		}
		current, err := rdb.Get(ctx, key).Result()
		if errors.Is(err, rds.Nil) {
			continue // 登记恰好被删除，重新认领
		}
		if err != nil {
			return fmt.Errorf("read redis namespace owner %q: %w", key, err)
		}
		if current == owner {
			return nil
		}
		return fmt.Errorf("redis namespace %q belongs to deployment %s, but this database is deployment %s: "+
			"give every deployment sharing this Redis its own General.InstanceID (env CASTOR_INSTANCE_ID); "+
			"if that deployment has been removed, delete the key %q and restart",
			ns, current, owner, key)
	}
	return fmt.Errorf("claim redis namespace %q: owner key keeps disappearing", ns)
}
