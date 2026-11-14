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

func TestFlightAggregatorReturnsErrorWhenAllProvidersFail(t *testing.T) {
	discardLog := slog.New(slog.NewJSONHandler(io.Discard, nil))
	providers := []domain.FlightProvider{
		fakeProvider{name: "Garuda Indonesia", err: domain.ErrProviderUnavailable},
		fakeProvider{name: "Lion Air", err: domain.ErrProviderUnavailable},
	}

	agg := NewFlightAggregator(providers, discardLog)
	res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if !errors.Is(err, domain.ErrAllProvidersFailed) {
		t.Fatalf("expected ErrAllProvidersFailed, got %v", err)
	}
	if res.Metadata.ProvidersQueried != 2 || res.Metadata.ProvidersSucceeded != 0 || res.Metadata.ProvidersFailed != 2 {
		t.Fatalf("unexpected provider metadata: %+v", res.Metadata)
	}
	if len(res.Metadata.ProviderErrors) != 2 {
		t.Fatalf("expected 2 provider errors recorded, got %+v", res.Metadata.ProviderErrors)
	}
}

func TestSearchFlightsPreservesComparisonAndScoreUnderFiltering(t *testing.T) {
	discardLog := slog.New(slog.NewJSONHandler(io.Discard, nil))
	providers := []domain.FlightProvider{
		fakeProvider{
			name: "Garuda Indonesia",
			flights: []domain.Flight{
				sampleFlight("GA400_Garuda", "Garuda Indonesia", "GA400", 1250000),
				sampleFlight("GA410_Garuda", "Garuda Indonesia", "GA410", 1450000),
			},
		},
		fakeProvider{
			name:    "AirAsia",
			flights: []domain.Flight{sampleFlight("QZ7250_AirAsia", "AirAsia", "QZ7250", 485000)},
		},
	}

	agg := NewFlightAggregator(providers, discardLog)
	minPrice := int64(1400000)
	res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{
		Filters: domain.FilterOptions{
			Airlines: []string{"Garuda Indonesia"},
			MinPrice: &minPrice,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Flights) != 1 || res.Flights[0].FlightNumber != "GA410" {
		t.Fatalf("expected only GA410 surviving filter, got %+v", res.Flights)
	}
	ga410 := res.Flights[0]
	if ga410.PriceComparison == nil || ga410.PriceComparison.CheapestProvider != "AirAsia" ||
		ga410.PriceComparison.CheapestAmount != 485000 || ga410.PriceComparison.ProviderCount != 2 {
		t.Fatalf("expected full-market PriceComparison preserved after filter, got %+v", ga410.PriceComparison)
	}
	if ga410.BestValueScore == nil || *ga410.BestValueScore >= 1.0 {
		t.Fatalf("expected market-relative BestValueScore < 1.0 for most expensive flight, got %v", ga410.BestValueScore)
	}
}

func TestFlightAggregatorFiltersMismatchedProviderFlights(t *testing.T) {
	wrongDest := sampleFlight("GA999_Garuda", "Garuda Indonesia", "GA999", 1000000)
	wrongDest.Arrival.Airport = "SIN"
	wrongCabin := sampleFlight("GA998_Garuda", "Garuda Indonesia", "GA998", 1000000)
	wrongCabin.CabinClass = "business"
	lowSeats := sampleFlight("GA997_Garuda", "Garuda Indonesia", "GA997", 1000000)
	lowSeats.AvailableSeats = 1
	validMatch := sampleFlight("GA400_Garuda", "Garuda Indonesia", "GA400", 1250000)
	validMatch.AvailableSeats = 4

	agg := NewFlightAggregator([]domain.FlightProvider{
		fakeProvider{
			name:    "Garuda Indonesia",
			flights: []domain.Flight{wrongDest, wrongCabin, lowSeats, validMatch},
		},
	}, nil)

	c := validCriteria()
	c.Passengers = 2
	res, err := agg.SearchFlights(context.Background(), c, SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Flights) != 1 || res.Flights[0].FlightNumber != "GA400" {
		t.Fatalf("expected only criteria-matching flight GA400, got %+v", res.Flights)
	}
}

func TestFlightAggregatorRejectsInvalidSortAndFilters(t *testing.T) {
	agg := NewFlightAggregator(nil, nil)

	if _, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{
		SortBy: domain.SortBy("invalid_mode"),
	}); !errors.Is(err, domain.ErrSearchInvalidCriteria) {
		t.Fatalf("expected ErrSearchInvalidCriteria for invalid sortBy, got %v", err)
	}

	if _, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{
		Filters: domain.FilterOptions{
			DepartureWindow: &domain.TimeRange{Start: "99:99", End: "bad"},
		},
	}); !errors.Is(err, domain.ErrSearchInvalidCriteria) {
		t.Fatalf("expected ErrSearchInvalidCriteria for malformed departure_window, got %v", err)
	}
}
