package garuda

import (
	"context"
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

func TestGarudaContract(t *testing.T) {
	providertest.VerifyContract(t, NewClient(), defaultCriteria())
}

func TestGarudaNormalizationAndGA315Segments(t *testing.T) {
	client := NewClient(WithDelay(0, 0))
	flights, err := client.SearchFlights(context.Background(), defaultCriteria())
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 Garuda flights, got %d", len(flights))
	}

	// Verify GA315 multi-segment trap was resolved: SUB top-level arrival overridden by DPS.
	ga315 := flights[2]
	if ga315.FlightNumber != "GA315" {
		t.Fatalf("expected 3rd flight GA315, got %s", ga315.FlightNumber)
	}
	if ga315.Arrival.Airport != "DPS" || ga315.Arrival.City != "Denpasar" {
		t.Fatalf("expected GA315 final arrival DPS/Denpasar, got %s/%s", ga315.Arrival.Airport, ga315.Arrival.City)
	}
	if ga315.Stops != 1 {
		t.Fatalf("expected GA315 stops=1 from 2 segments, got %d", ga315.Stops)
	}
	if ga315.Duration.TotalMinutes != 225 || ga315.Duration.Formatted != "3h 45m" {
		t.Fatalf("expected GA315 duration 225m (3h 45m), got %dm (%s)", ga315.Duration.TotalMinutes, ga315.Duration.Formatted)
	}
	if ga315.Amenities == nil || len(ga315.Amenities) != 0 {
		t.Fatalf("expected non-nil empty Amenities slice, got %#v", ga315.Amenities)
	}
	if ga315.Price.Formatted != "Rp 1.850.000" {
		t.Fatalf("expected formatted price Rp 1.850.000, got %q", ga315.Price.Formatted)
	}
}

func TestGarudaCriteriaFiltering(t *testing.T) {
	client := NewClient(WithDelay(0, 0))

	// Only GA400 (28 seats) and GA315 (22 seats) have >= 20 seats; GA410 has 15.
	c := defaultCriteria()
	c.Passengers = 20
	flights, err := client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 2 {
		t.Fatalf("expected 2 flights with >=20 seats, got %d", len(flights))
	}

	// Non-matching route returns empty non-nil slice.
	c.Destination = "SIN"
	flights, err = client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if flights == nil || len(flights) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", flights)
	}
}

func TestGarudaContractErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "bad json", payload: `{invalid`},
		{name: "non-success status", payload: `{"status":"error","flights":[]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(WithDelay(0, 0), WithPayload([]byte(tc.payload)))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := client.SearchFlights(ctx, defaultCriteria()); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
