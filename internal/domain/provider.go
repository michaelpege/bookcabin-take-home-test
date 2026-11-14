package domain

import (
	"context"
	"errors"
)

// ErrProviderUnavailable indicates an upstream airline provider failed or timed out.
var ErrProviderUnavailable = errors.New("provider unavailable")

// FlightProvider defines the outbound contract implemented by each airline adapter
// and resilience decorator.
type FlightProvider interface {
	Name() string
	Search(ctx context.Context, criteria SearchCriteria) ([]Flight, error)
}
