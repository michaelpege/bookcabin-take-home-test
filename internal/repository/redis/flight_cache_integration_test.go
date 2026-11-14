//go:build integration

package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

func redisIntegrationAddr() string {
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		return v
	}
	return "localhost:6379"
}

func TestFlightCacheIntegrationRoundTrip(t *testing.T) {
	cache := NewFlightCache(redisIntegrationAddr(), WithOperationTimeout(2*time.Second))
	defer func() { _ = cache.Close() }()

	ctx := context.Background()
	if err := cache.Ping(ctx); err != nil {
		t.Fatalf("ping Redis at %s: %v", redisIntegrationAddr(), err)
	}

	missKey := "test:flight_cache:missing:" + time.Now().Format("150405.000000000")
	if _, err := cache.Get(ctx, missKey); !errors.Is(err, domain.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss for %q, got %v", missKey, err)
	}

	key := "test:flight_cache:roundtrip:" + time.Now().Format("150405.000000000")
	want := []byte(`{"flights":[{"id":"GA400_Garuda"}]}`)
	if err := cache.Set(ctx, key, want, 30*time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("cached payload mismatch: got %q, want %q", got, want)
	}
}
