package cache

import (
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/db/redis"
	rds "github.com/redis/go-redis/v9"
)

// NewRedis 创建 Redis 客户端
func NewRedis() (rds.UniversalClient, error) {
	client, err := redis.NewClient(&config.C.Redis)
	if err != nil {
		return nil, err
	}
	return client.UniversalClient(), nil
}
