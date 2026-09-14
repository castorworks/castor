package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本：原子化 INCR + EXPIRE
//   - 对 key 执行 INCR
//   - 如果 INCR 返回 1（即 key 刚创建），设置过期时间
//   - 返回当前计数值
//
// 这消除了 INCR 和 EXPIRE 之间的竞态条件（进程在 INCR 后但 EXPIRE 前崩溃
// 会导致 key 永不过期）。
// Script 自动处理 EVALSHA / NOSCRIPT 回退，且可安全地被并发调用。
var incrWithExpireScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

// RateLimiter 限流器
type RateLimiter struct {
	redisClient redis.UniversalClient
}

// NewRateLimiter 创建限流器
func NewRateLimiter(redisClient redis.UniversalClient) *RateLimiter {
	return &RateLimiter{redisClient: redisClient}
}

// IsRequestReachLimit 判断当前请求是否超出限制
// key: 限流的唯一标识
// duration: 限流时间窗口
// maxAllowed: 最大允许请求数
// 返回: (是否超出限制, 错误)
func (r *RateLimiter) IsRequestReachLimit(ctx context.Context, key string, duration time.Duration, maxAllowed int64) (bool, error) {
	cacheKey := fmt.Sprintf("rate_limit:%s", key)

	seconds := int64(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	// 使用原子 Lua 脚本执行 INCR + EXPIRE，消除竞态条件
	current, err := incrWithExpireScript.Run(ctx, r.redisClient, []string{cacheKey}, seconds).Int64()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return current > maxAllowed, nil
}

// GetCurrentCount 获取当前计数
func (r *RateLimiter) GetCurrentCount(ctx context.Context, key string) (int64, error) {
	cacheKey := fmt.Sprintf("rate_limit:%s", key)

	count, err := r.redisClient.Get(ctx, cacheKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}
