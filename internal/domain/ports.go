package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrProviderUnavailable indicates an upstream airline provider failed or timed out.
	ErrProviderUnavailable = errors.New("provider unavailable")
	// ErrCacheMiss indicates the requested key is not present in the cache.
	ErrCacheMiss = errors.New("cache miss")
)

// FlightProvider defines the outbound port implemented by each airline adapter
// and resilience decorator.
type FlightProvider interface {
	Name() string
	Search(ctx context.Context, criteria SearchCriteria) ([]Flight, error)
}

// FlightCache defines the outbound port for caching serialized flight results.
type FlightCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
