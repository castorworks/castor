package service

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/hyperits/gosuite/logger"
	"github.com/redis/go-redis/v9"
)

// TokenBlacklistService Token 黑名单服务接口
type TokenBlacklistService interface {
	// AddToBlacklist 将 token 加入黑名单
	AddToBlacklist(ctx context.Context, token string, expiry time.Duration) error
	// IsBlacklisted 检查 token 是否在黑名单中（Redis 故障时 fail-safe 拒绝访问）
	IsBlacklisted(ctx context.Context, token string) bool
}

type tokenBlacklistService struct {
	redisClient redis.UniversalClient
	keys        rediskey.Namespace
}

// Token 黑名单 Redis key 前缀
const tokenBlacklistKeyPrefix = "token:blacklist:"

// NewTokenBlacklistService 创建 Token 黑名单服务
func NewTokenBlacklistService(redisClient redis.UniversalClient, ns rediskey.Namespace) TokenBlacklistService {
	return &tokenBlacklistService{
		redisClient: redisClient,
		keys:        ns,
	}
}

func (svc *tokenBlacklistService) AddToBlacklist(ctx context.Context, token string, expiry time.Duration) error {
	key := svc.keys.Key(tokenBlacklistKeyPrefix + token)
	return svc.redisClient.Set(ctx, key, "1", expiry).Err()
}

func (svc *tokenBlacklistService) IsBlacklisted(ctx context.Context, token string) bool {
	key := svc.keys.Key(tokenBlacklistKeyPrefix + token)
	result, err := svc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		// Fail-safe：Redis 故障时拒绝访问，防止已注销的 token 被放行
		logger.Errorf("Redis error checking token blacklist, denying access for safety: %v", err)
		return true
	}
	return result == "1"
}
