package airasia

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider/providertest"
)

func defaultCriteria() domain.SearchCriteria {
	return domain.SearchCriteria{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func TestAirAsiaContract(t *testing.T) {
	providertest.VerifyContract(t, NewClient(WithFailureRate(0)), defaultCriteria())
}

func TestAirAsiaNormalizationDurationAndStops(t *testing.T) {
	client := NewClient(WithDelay(0, 0), WithFailureRate(0))
	flights, err := client.SearchFlights(context.Background(), defaultCriteria())
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 4 {
		t.Fatalf("expected 4 AirAsia flights, got %d", len(flights))
	}

	// QZ520: 1.67h -> 100m (1h 40m), airline IATA extracted as QZ, baggage parsed.
	qz520 := flights[0]
	if qz520.FlightNumber != "QZ520" || qz520.Airline.Code != "QZ" {
		t.Fatalf("unexpected flight/airline identity: %+v", qz520.Airline)
	}
	if qz520.Duration.TotalMinutes != 100 || qz520.Duration.Formatted != "1h 40m" {
		t.Fatalf("expected 100m (1h 40m), got %+v", qz520.Duration)
	}
	if qz520.Amenities == nil || len(qz520.Amenities) != 0 {
		t.Fatalf("expected non-nil empty Amenities slice, got %#v", qz520.Amenities)
	}
	if qz520.Baggage.CarryOn != "Cabin baggage only" || qz520.Baggage.Checked != "Additional fee" {
		t.Fatalf("unexpected baggage: %+v", qz520.Baggage)
	}

	// QZ7250: 4.33h -> 260m (4h 20m), 1 stop (SOC), Rp 485.000.
	qz7250 := flights[3]
	if qz7250.FlightNumber != "QZ7250" {
		t.Fatalf("expected 4th flight QZ7250, got %s", qz7250.FlightNumber)
	}
	if qz7250.Stops != 1 {
		t.Fatalf("expected QZ7250 stops=1, got %d", qz7250.Stops)
	}
	if qz7250.Duration.TotalMinutes != 260 || qz7250.Duration.Formatted != "4h 20m" {
		t.Fatalf("expected 260m (4h 20m), got %+v", qz7250.Duration)
	}
	if qz7250.Price.Formatted != "Rp 485.000" {
		t.Fatalf("expected Rp 485.000, got %q", qz7250.Price.Formatted)
	}
}

func TestAirAsiaCriteriaFiltering(t *testing.T) {
	client := NewClient(WithDelay(0, 0), WithFailureRate(0))

	// Only QZ532 (72 seats) and QZ7250 (88 seats) have >= 70 seats.
	c := defaultCriteria()
	c.Passengers = 70
	flights, err := client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 2 {
		t.Fatalf("expected 2 flights with >=70 seats, got %d", len(flights))
	}

	c.Destination = "SOC"
	flights, err = client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if flights == nil || len(flights) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", flights)
	}
}

func TestAirAsiaSimulatedFailureAndErrorPayloads(t *testing.T) {
	alwaysFail := NewClient(WithDelay(0, 0), WithFailureRate(1.0))
	if _, err := alwaysFail.SearchFlights(context.Background(), defaultCriteria()); !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable on simulated failure, got %v", err)
	}

	tests := []struct {
		name    string
		payload string
	}{
		{name: "bad json", payload: `{invalid`},
		{name: "non-ok status", payload: `{"status":"error","flights":[]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(WithDelay(0, 0), WithFailureRate(0), WithPayload([]byte(tc.payload)))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := client.SearchFlights(ctx, defaultCriteria()); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
