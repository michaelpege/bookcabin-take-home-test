package domain

import (
	"context"
	"time"
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
