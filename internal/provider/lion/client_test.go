package lion

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

func TestLionContract(t *testing.T) {
	providertest.VerifyContract(t, NewClient(), defaultCriteria())
}

func TestLionNormalizationTimezonesAndStops(t *testing.T) {
	client := NewClient(WithDelay(0, 0))
	flights, err := client.SearchFlights(context.Background(), defaultCriteria())
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 Lion Air flights, got %d", len(flights))
	}

	// JT740: Asia/Jakarta (+07:00) -> Asia/Makassar (+08:00), 105m direct.
	jt740 := flights[0]
	if jt740.FlightNumber != "JT740" {
		t.Fatalf("expected 1st flight JT740, got %s", jt740.FlightNumber)
	}
	if jt740.Departure.Datetime != "2025-12-15T05:30:00+07:00" {
		t.Fatalf("expected +07:00 departure RFC3339, got %q", jt740.Departure.Datetime)
	}
	if jt740.Arrival.Datetime != "2025-12-15T08:15:00+08:00" {
		t.Fatalf("expected +08:00 arrival RFC3339, got %q", jt740.Arrival.Datetime)
	}
	if jt740.Duration.TotalMinutes != 105 || jt740.Duration.Formatted != "1h 45m" {
		t.Fatalf("unexpected duration: %+v", jt740.Duration)
	}
	if jt740.Stops != 0 {
		t.Fatalf("expected 0 stops for direct JT740, got %d", jt740.Stops)
	}
	if jt740.Amenities == nil || len(jt740.Amenities) != 0 {
		t.Fatalf("expected non-nil empty Amenities slice, got %#v", jt740.Amenities)
	}
	if jt740.Baggage.CarryOn != "7 kg" || jt740.Baggage.Checked != "20 kg" {
		t.Fatalf("unexpected baggage allowance: %+v", jt740.Baggage)
	}

	// JT650: connecting flight with stop_count=1 and IDR formatting.
	jt650 := flights[2]
	if jt650.FlightNumber != "JT650" {
		t.Fatalf("expected 3rd flight JT650, got %s", jt650.FlightNumber)
	}
	if jt650.Stops != 1 {
		t.Fatalf("expected JT650 stops=1, got %d", jt650.Stops)
	}
	if jt650.Price.Formatted != "Rp 780.000" {
		t.Fatalf("expected Rp 780.000, got %q", jt650.Price.Formatted)
	}
}

func TestLionCriteriaFiltering(t *testing.T) {
	client := NewClient(WithDelay(0, 0))

	// Only JT740 (45 seats) and JT650 (52 seats) have >= 40 seats; JT742 has 38.
	c := defaultCriteria()
	c.Passengers = 40
	flights, err := client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 2 {
		t.Fatalf("expected 2 flights with >=40 seats, got %d", len(flights))
	}

	c.Destination = "SUB"
	flights, err = client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if flights == nil || len(flights) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", flights)
	}
}

func TestLionContractErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "bad json", payload: `{invalid`},
		{name: "unsuccessful response", payload: `{"success":false,"data":{"available_flights":[]}}`},
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
