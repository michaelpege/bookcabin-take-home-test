package domain

import "context"

// FlightProvider defines the outbound contract implemented by each airline adapter
// and resilience decorator.
type FlightProvider interface {
	Name() string
	Search(ctx context.Context, criteria SearchCriteria) ([]Flight, error)
}
