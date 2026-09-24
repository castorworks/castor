package external

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T, maxAttempts int) (*VerificationCodeStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return newVerificationCodeStore(rdb, "castor", time.Minute, 6, maxAttempts), mr
}

func TestVerificationCodeStoreConsumesCode(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t, 5)
	ctx := context.Background()
	code, err := store.Generate(ctx, "auth", "user@example.com")
	if err != nil || len(code) != 6 {
		t.Fatalf("Generate() = %q, %v", code, err)
	}
	if ok, err := store.Verify(ctx, "other", "user@example.com", code); ok || err != nil {
		t.Fatalf("code must be scoped by purpose: ok=%v err=%v", ok, err)
	}
	if ok, err := store.Verify(ctx, "auth", "user@example.com", code); !ok || err != nil {
		t.Fatalf("valid code rejected: ok=%v err=%v", ok, err)
	}
	if ok, _ := store.Verify(ctx, "auth", "user@example.com", code); ok {
		t.Fatal("code must be single-use")
	}
}

func TestVerificationCodeStoreInvalidatesAfterFailedAttempts(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t, 5)
	ctx := context.Background()
	code, err := store.Generate(ctx, "auth", "13800000000")
	if err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		if ok, err := store.Verify(ctx, "auth", "13800000000", wrong); ok || err != nil {
			t.Fatalf("attempt %d: ok=%v err=%v", i, ok, err)
		}
	}
	if ok, _ := store.Verify(ctx, "auth", "13800000000", code); ok {
		t.Fatal("correct code must be rejected once the attempt limit is exhausted")
	}

	// A newly issued code resets the counter.
	code, err = store.Generate(ctx, "auth", "13800000000")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Verify(ctx, "auth", "13800000000", code); !ok || err != nil {
		t.Fatalf("fresh code rejected: ok=%v err=%v", ok, err)
	}
}

func TestVerificationCodeStoreConcurrentGuessesAreBounded(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t, 5)
	ctx := context.Background()
	code, err := store.Generate(ctx, "auth", "race@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.Verify(ctx, "auth", "race@example.com", "bad")
		}()
	}
	wg.Wait()
	if ok, _ := store.Verify(ctx, "auth", "race@example.com", code); ok {
		t.Fatal("parallel guesses must exhaust the attempt budget")
	}
}
