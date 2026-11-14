package domain

import (
	"context"
	"time"
)

// FlightCache defines the outbound contract for caching serialized flight results.
type FlightCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
