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
		{name: "contradictory duration vs timestamps", mutate: func(f *Flight) { f.Duration.TotalMinutes = 30 }, wantErr: true},
		{name: "negative stops", mutate: func(f *Flight) { f.Stops = -1 }, wantErr: true},
		{name: "zero price", mutate: func(f *Flight) { f.Price.Amount = 0 }, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := validFlight()
			tc.mutate(&f)
			err := f.Validate()
			if tc.wantErr && !errors.Is(err, ErrFlightInvalid) {
				t.Fatalf("expected ErrFlightInvalid, got %v", err)
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
			name:     "empty criteria (missing required fields)",
			criteria: SearchCriteria{},
			wantErr:  true,
		},
		{
			name: "missing origin",
			criteria: SearchCriteria{
				Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "missing destination",
			criteria: SearchCriteria{
				Origin: "CGK", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "missing departure_date",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "DPS", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "missing cabin_class",
			criteria: SearchCriteria{
				Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid airport code length",
			criteria: SearchCriteria{
				Origin: "CG", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "non-alpha airport code digits",
			criteria: SearchCriteria{
				Origin: "123", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
			},
			wantErr: true,
		},
		{
			name: "non-alpha airport code whitespace",
			criteria: SearchCriteria{
				Origin: "C G", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy",
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
		{
			name: "invalid return_date format",
			criteria: func() SearchCriteria {
				ret := "bad-date"
				return SearchCriteria{
					Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", ReturnDate: &ret, Passengers: 1, CabinClass: "economy",
				}
			}(),
			wantErr: true,
		},
		{
			name: "return_date before departure_date",
			criteria: func() SearchCriteria {
				ret := "2025-12-10"
				return SearchCriteria{
					Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", ReturnDate: &ret, Passengers: 1, CabinClass: "economy",
				}
			}(),
			wantErr: true,
		},
		{
			name: "valid same-day return_date",
			criteria: func() SearchCriteria {
				ret := "2025-12-15"
				return SearchCriteria{
					Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15", ReturnDate: &ret, Passengers: 1, CabinClass: "economy",
				}
			}(),
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.criteria.Validate()
			if tc.wantErr && !errors.Is(err, ErrSearchInvalidCriteria) {
				t.Fatalf("expected ErrSearchInvalidCriteria, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSearchCriteriaNormalize(t *testing.T) {
	ret := " 2025-12-20 "
	c := SearchCriteria{
		Origin:        " cgk ",
		Destination:   " dps ",
		DepartureDate: " 2025-12-15 ",
		ReturnDate:    &ret,
		Passengers:    1,
		CabinClass:    " ECONOMY ",
	}.Normalize()
	if c.Origin != "CGK" || c.Destination != "DPS" || c.DepartureDate != "2025-12-15" || c.CabinClass != "economy" {
		t.Fatalf("unexpected normalized criteria: %+v", c)
	}
	if c.ReturnDate == nil || *c.ReturnDate != "2025-12-20" {
		t.Fatalf("expected trimmed ReturnDate, got %v", c.ReturnDate)
	}

	blank := "   "
	emptyRet := SearchCriteria{
		Origin: "cgk", Destination: "dps", DepartureDate: "2025-12-15", ReturnDate: &blank, Passengers: 1, CabinClass: "economy",
	}.Normalize()
	if emptyRet.ReturnDate != nil {
		t.Fatalf("expected blank ReturnDate collapsed to nil, got %q", *emptyRet.ReturnDate)
	}
}

func TestSortByAndFilterOptionsValidate(t *testing.T) {
	if err := SortBy("best_value").Validate(); err != nil {
		t.Fatalf("expected valid sort_by, got %v", err)
	}
	if err := SortBy("invalid_mode").Validate(); !errors.Is(err, ErrSearchInvalidCriteria) {
		t.Fatalf("expected ErrSearchInvalidCriteria for invalid sort_by, got %v", err)
	}

	neg := int64(-1)
	low := int64(500000)
	high := int64(1000000)
	zero := 0
	negInt := -1
	invalidFilters := []struct {
		name string
		opts FilterOptions
	}{
		{name: "negative min_price", opts: FilterOptions{MinPrice: &neg}},
		{name: "negative max_price", opts: FilterOptions{MaxPrice: &neg}},
		{name: "min_price > max_price", opts: FilterOptions{MinPrice: &high, MaxPrice: &low}},
		{name: "negative max_stops", opts: FilterOptions{MaxStops: &negInt}},
		{name: "negative stops entry", opts: FilterOptions{Stops: []int{0, -1}}},
		{name: "non-positive max_duration_minutes", opts: FilterOptions{MaxDurationMinutes: &zero}},
		{name: "malformed departure_window", opts: FilterOptions{DepartureWindow: &TimeRange{Start: "99:99", End: "bad"}}},
		{name: "malformed arrival_window", opts: FilterOptions{ArrivalWindow: &TimeRange{Start: "12:0", End: "14:00"}}},
	}
	for _, tc := range invalidFilters {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); !errors.Is(err, ErrSearchInvalidCriteria) {
				t.Fatalf("expected ErrSearchInvalidCriteria, got %v", err)
			}
		})
	}
}
