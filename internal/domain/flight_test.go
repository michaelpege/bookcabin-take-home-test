package domain

import (
	"errors"
	"testing"
)

func validFlight() Flight {
	return Flight{
		ID:           "GA400_Garuda Indonesia",
		Provider:     "Garuda Indonesia",
		Airline:      Airline{Name: "Garuda Indonesia", Code: "GA"},
		FlightNumber: "GA400",
		Departure: LocationPoint{
			Airport:   "CGK",
			City:      "Jakarta",
			Datetime:  "2025-12-15T06:00:00+07:00",
			Timestamp: 1765753200,
		},
		Arrival: LocationPoint{
			Airport:   "DPS",
			City:      "Denpasar",
			Datetime:  "2025-12-15T08:50:00+08:00",
			Timestamp: 1765759800,
		},
		Duration:       DurationInfo{TotalMinutes: 110, Formatted: "1h 50m"},
		Stops:          0,
		Price:          PriceInfo{Amount: 1250000, Currency: "IDR"},
		AvailableSeats: 28,
		CabinClass:     "economy",
		Amenities:      []string{},
		Baggage:        BaggageInfo{CarryOn: "1 piece", Checked: "2 pieces"},
	}
}

func TestFlightValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Flight)
		wantErr bool
	}{
		{name: "valid flight", mutate: func(_ *Flight) {}, wantErr: false},
		{name: "missing id", mutate: func(f *Flight) { f.ID = "" }, wantErr: true},
		{name: "same origin and destination", mutate: func(f *Flight) { f.Arrival.Airport = "CGK" }, wantErr: true},
		{name: "arrival before departure", mutate: func(f *Flight) { f.Arrival.Timestamp = f.Departure.Timestamp - 60 }, wantErr: true},
		{name: "zero duration", mutate: func(f *Flight) { f.Duration.TotalMinutes = 0 }, wantErr: true},
		{name: "negative stops", mutate: func(f *Flight) { f.Stops = -1 }, wantErr: true},
		{name: "zero price", mutate: func(f *Flight) { f.Price.Amount = 0 }, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := validFlight()
			tc.mutate(&f)
			err := f.Validate()
			if tc.wantErr && !errors.Is(err, ErrInvalidFlight) {
				t.Fatalf("expected ErrInvalidFlight, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSearchCriteriaValidate(t *testing.T) {
	tests := []struct {
		name     string
		criteria SearchCriteria
		wantErr  bool
	}{
		{
			name: "valid criteria",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: false,
		},
		{
			name: "invalid airport code",
			criteria: SearchCriteria{
				Origin: "CG", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "same origin and destination",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "CGK", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "bad date format",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "DPS", DepartureDate: "15-12-2025", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "zero passengers",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 0, CabinClass: "economy",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.criteria.Validate()
			if tc.wantErr && !errors.Is(err, ErrInvalidSearchCriteria) {
				t.Fatalf("expected ErrInvalidSearchCriteria, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
