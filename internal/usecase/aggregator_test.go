package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

type fakeProvider struct {
	name    string
	delay   time.Duration
	flights []domain.Flight
	err     error
}

func (f fakeProvider) Name() string { return f.name }

func (f fakeProvider) SearchFlights(ctx context.Context, _ domain.SearchCriteria) ([]domain.Flight, error) {
	if f.delay > 0 {
		timer := time.NewTimer(f.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.flights, nil
}

func sampleFlight(id, providerName, fn string, price int64) domain.Flight {
	return domain.Flight{
		ID:           id,
		Provider:     providerName,
		Airline:      domain.Airline{Name: providerName, Code: fn[:2]},
		FlightNumber: fn,
		Departure: domain.LocationPoint{
			Airport: "CGK", City: "Jakarta", Datetime: "2025-12-15T06:00:00+07:00", Timestamp: 1765753200,
		},
		Arrival: domain.LocationPoint{
			Airport: "DPS", City: "Denpasar", Datetime: "2025-12-15T08:50:00+08:00", Timestamp: 1765759800,
		},
		Duration:       domain.DurationInfo{TotalMinutes: 110, Formatted: "1h 50m"},
		Stops:          0,
		Price:          domain.PriceInfo{Amount: price, Currency: "IDR"},
		AvailableSeats: 10,
		CabinClass:     "economy",
		Amenities:      []string{},
	}
}

func validCriteria() domain.SearchCriteria {
	return domain.SearchCriteria{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func TestFlightAggregatorConcurrentAndPartialFailure(t *testing.T) {
	discardLog := slog.New(slog.NewJSONHandler(io.Discard, nil))
	providers := []domain.FlightProvider{
		fakeProvider{
			name:    "Garuda Indonesia",
			delay:   40 * time.Millisecond,
			flights: []domain.Flight{sampleFlight("GA400_Garuda Indonesia", "Garuda Indonesia", "GA400", 1250000)},
		},
		fakeProvider{
			name:  "AirAsia",
			delay: 10 * time.Millisecond,
			err:   domain.ErrProviderUnavailable,
		},
	}

	agg := NewFlightAggregator(providers, discardLog)
	res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error on partial failure: %v", err)
	}
	if res.Metadata.ProvidersQueried != 2 {
		t.Fatalf("expected ProvidersQueried=2, got %d", res.Metadata.ProvidersQueried)
	}
	if res.Metadata.ProvidersSucceeded != 1 || res.Metadata.ProvidersFailed != 1 {
		t.Fatalf("unexpected provider counts: %+v", res.Metadata)
	}
	if res.Metadata.TotalResults != 1 || len(res.Flights) != 1 {
		t.Fatalf("expected 1 healthy flight, got %d", len(res.Flights))
	}
	if _, ok := res.Metadata.ProviderErrors["AirAsia"]; !ok {
		t.Fatalf("expected AirAsia error recorded in ProviderErrors, got %+v", res.Metadata.ProviderErrors)
	}
}

func TestFlightAggregatorRejectsInvalidCriteria(t *testing.T) {
	agg := NewFlightAggregator(nil, nil)
	_, err := agg.SearchFlights(context.Background(), domain.SearchCriteria{Origin: "CGK"}, SearchOptions{})
	if !errors.Is(err, domain.ErrSearchInvalidCriteria) {
		t.Fatalf("expected ErrSearchInvalidCriteria, got %v", err)
	}
}
