package domain

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss indicates the requested key is not present in the cache.
var ErrCacheMiss = errors.New("cache miss")

// FlightCache defines the outbound contract for caching serialized flight results.
type FlightCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
