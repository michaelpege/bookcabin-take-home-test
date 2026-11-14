package batik

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

func TestBatikContract(t *testing.T) {
	providertest.VerifyContract(t, NewClient(), defaultCriteria())
}

func TestBatikNormalizationOffsetsAndConnections(t *testing.T) {
	client := NewClient(WithDelay(0, 0))
	flights, err := client.SearchFlights(context.Background(), defaultCriteria())
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 Batik Air flights, got %d", len(flights))
	}

	// ID6514: numeric offsets (+0700 / +0800) normalized to RFC3339, IATA "Y" -> "economy".
	id6514 := flights[0]
	if id6514.FlightNumber != "ID6514" {
		t.Fatalf("expected 1st flight ID6514, got %s", id6514.FlightNumber)
	}
	if id6514.Departure.City != "Jakarta" || id6514.Arrival.City != "Denpasar" {
		t.Fatalf("unexpected resolved cities: %s -> %s", id6514.Departure.City, id6514.Arrival.City)
	}
	if id6514.Departure.Datetime != "2025-12-15T07:15:00+07:00" || id6514.Arrival.Datetime != "2025-12-15T10:00:00+08:00" {
		t.Fatalf("unexpected datetimes: %s -> %s", id6514.Departure.Datetime, id6514.Arrival.Datetime)
	}
	if id6514.CabinClass != "economy" {
		t.Fatalf("expected cabinClass=economy from IATA Y, got %q", id6514.CabinClass)
	}
	if id6514.Baggage.CarryOn != "7kg" || id6514.Baggage.Checked != "20kg" {
		t.Fatalf("unexpected baggage: %+v", id6514.Baggage)
	}
	if len(id6514.Amenities) != 2 || id6514.Amenities[0] != "snack" {
		t.Fatalf("expected lowercase amenities [snack beverage], got %#v", id6514.Amenities)
	}

	// ID7042: connecting flight derives authoritative UTC elapsed duration (245m = 4h 5m).
	id7042 := flights[2]
	if id7042.FlightNumber != "ID7042" {
		t.Fatalf("expected 3rd flight ID7042, got %s", id7042.FlightNumber)
	}
	if id7042.Stops != 1 {
		t.Fatalf("expected ID7042 stops=1, got %d", id7042.Stops)
	}
	if id7042.Duration.TotalMinutes != 245 || id7042.Duration.Formatted != "4h 5m" {
		t.Fatalf("expected 245m (4h 5m) UTC elapsed duration, got %+v", id7042.Duration)
	}
	if id7042.Price.Formatted != "Rp 950.000" {
		t.Fatalf("expected Rp 950.000, got %q", id7042.Price.Formatted)
	}
}

func TestBatikCriteriaFiltering(t *testing.T) {
	client := NewClient(WithDelay(0, 0))

	// Only ID6514 (32 seats) and ID7042 (41 seats) have >= 25 seats; ID6520 has 18.
	c := defaultCriteria()
	c.Passengers = 25
	flights, err := client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if len(flights) != 2 {
		t.Fatalf("expected 2 flights with >=25 seats, got %d", len(flights))
	}

	c.CabinClass = "business"
	flights, err = client.SearchFlights(context.Background(), c)
	if err != nil {
		t.Fatalf("SearchFlights failed: %v", err)
	}
	if flights == nil || len(flights) != 0 {
		t.Fatalf("expected empty non-nil slice for business class, got %#v", flights)
	}
}

func TestBatikContractErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "bad json", payload: `{invalid`},
		{name: "non-200 status", payload: `{"code":500,"message":"upstream error","results":[]}`},
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

func TestBatikRejectsInvalidConnections(t *testing.T) {
	payload := `{
		"code": 200,
		"message": "OK",
		"results": [{
			"flightNumber": "ID9999",
			"airlineName": "Batik Air",
			"airlineIATA": "ID",
			"origin": "CGK",
			"destination": "DPS",
			"departureDateTime": "2025-12-15T07:15:00+0700",
			"arrivalDateTime": "2025-12-15T12:15:00+0800",
			"travelTime": "4h 0m",
			"numberOfStops": 1,
			"connections": [{"stopAirport": "", "stopDuration": "bad"}],
			"fare": {"basePrice": 900000, "taxes": 100000, "totalPrice": 1000000, "currencyCode": "IDR", "class": "Y"},
			"seatsAvailable": 10
		}]
	}`
	client := NewClient(WithDelay(0, 0), WithPayload([]byte(payload)))
	flights, err := client.SearchFlights(context.Background(), defaultCriteria())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flights) != 0 {
		t.Fatalf("expected flight with malformed connection dropped, got %+v", flights)
	}
}
