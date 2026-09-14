package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/redis/go-redis/v9"
)

// mockRedisClient implements redis.UniversalClient for testing
// Uses an in-memory map to simulate Redis behavior including Lua scripts
type mockRedisClient struct {
	redis.UniversalClient
	data    map[string]int64
	expires map[string]time.Duration
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data:    make(map[string]int64),
		expires: make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) ScriptLoad(_ context.Context, _ string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetVal("mock_sha")
	return cmd
}

func (m *mockRedisClient) EvalSha(_ context.Context, _ string, keys []string, args ...interface{}) *redis.Cmd {
	// Simulate the Lua script: INCR key, EXPIRE on first call
	key := keys[0]
	m.data[key]++
	current := m.data[key]

	if current == 1 && len(args) > 0 {
		seconds, _ := args[0].(int64)
		m.expires[key] = time.Duration(seconds) * time.Second
	}

	cmd := redis.NewCmd(context.Background())
	cmd.SetVal(current)
	return cmd
}

func (m *mockRedisClient) Get(_ context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	if val, ok := m.data[key]; ok {
		cmd.SetVal(time.Duration(val).String())
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func TestRateLimiter_IsRequestReachLimit(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		duration   time.Duration
		maxAllowed int64
		callCount  int
		wantLimit  bool
	}{
		{
			name:       "first request within limit",
			key:        "test:first",
			duration:   1 * time.Minute,
			maxAllowed: 5,
			callCount:  1,
			wantLimit:  false,
		},
		{
			name:       "requests at limit boundary",
			key:        "test:boundary",
			duration:   1 * time.Minute,
			maxAllowed: 3,
			callCount:  3,
			wantLimit:  false,
		},
		{
			name:       "requests exceed limit",
			key:        "test:exceed",
			duration:   1 * time.Minute,
			maxAllowed: 2,
			callCount:  3,
			wantLimit:  true,
		},
		{
			name:       "single allowed request exceeded",
			key:        "test:single",
			duration:   1 * time.Minute,
			maxAllowed: 1,
			callCount:  2,
			wantLimit:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockRedisClient()
			limiter := NewRateLimiter(mockClient)

			var reached bool
			var err error

			for i := 0; i < tt.callCount; i++ {
				reached, err = limiter.IsRequestReachLimit(context.Background(), tt.key, tt.duration, tt.maxAllowed)
				if err != nil {
					t.Fatalf("IsRequestReachLimit() unexpected error: %v", err)
				}
			}

			if reached != tt.wantLimit {
				t.Errorf("IsRequestReachLimit() after %d calls = %v, want %v", tt.callCount, reached, tt.wantLimit)
			}
		})
	}
}

func TestRateLimiter_IsRequestReachLimit_SetsExpiry(t *testing.T) {
	mockClient := newMockRedisClient()
	limiter := NewRateLimiter(mockClient)

	duration := 5 * time.Minute
	key := "test:expiry"

	_, err := limiter.IsRequestReachLimit(context.Background(), key, duration, 10)
	if err != nil {
		t.Fatalf("IsRequestReachLimit() unexpected error: %v", err)
	}

	cacheKey := "rate_limit:" + key
	expectedExpiry := time.Duration(int64(duration.Seconds())) * time.Second
	if exp, ok := mockClient.expires[cacheKey]; !ok {
		t.Error("expected expiry to be set on first request")
	} else if exp != expectedExpiry {
		t.Errorf("expected expiry %v, got %v", expectedExpiry, exp)
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	mockClient := newMockRedisClient()
	limiter := NewRateLimiter(mockClient)

	// Exhaust limit for key1
	for i := 0; i < 3; i++ {
		limiter.IsRequestReachLimit(context.Background(), "key1", time.Minute, 2)
	}

	// key2 should still be within limit
	reached, err := limiter.IsRequestReachLimit(context.Background(), "key2", time.Minute, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reached {
		t.Error("key2 should not be rate limited when only key1 was exhausted")
	}
}

func TestRateLimiter_ConcurrentRequestsAgainstRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	limiter := NewRateLimiter(client)

	const workers = 32
	var wg sync.WaitGroup
	var blocked atomic.Int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reached, err := limiter.IsRequestReachLimit(context.Background(), "concurrent", time.Minute, 10)
			if err != nil {
				t.Errorf("IsRequestReachLimit() error = %v", err)
				return
			}
			if reached {
				blocked.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := blocked.Load(); got != workers-10 {
		t.Fatalf("blocked = %d, want %d", got, workers-10)
	}
}
