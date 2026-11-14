package domain

import "context"

// FlightProvider defines the outbound contract implemented by each airline adapter
// and resilience decorator.
type FlightProvider interface {
	Name() string
	SearchFlights(ctx context.Context, criteria SearchCriteria) ([]Flight, error)
}
