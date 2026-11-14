package domain

import (
	"fmt"
	"strings"
)

// Airline identifies the operating carrier.
type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// LocationPoint represents a departure or arrival airport and localized time.
type LocationPoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

// DurationInfo captures total trip duration in minutes and human-readable form.
type DurationInfo struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

// PriceInfo represents fare pricing and optional formatted display string.
type PriceInfo struct {
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Formatted string `json:"formatted,omitempty"`
}

// BaggageInfo describes carry-on and checked baggage allowances.
type BaggageInfo struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

// PriceComparison summarizes cross-provider pricing for the same flight.
type PriceComparison struct {
	CheapestProvider string `json:"cheapest_provider"`
	CheapestAmount   int64  `json:"cheapest_amount"`
	PriceDiff        int64  `json:"price_diff"`
	ProviderCount    int    `json:"provider_count"`
}

// Flight is the canonical normalized flight entity across all airline providers.
type Flight struct {
	ID              string           `json:"id"`
	Provider        string           `json:"provider"`
	Airline         Airline          `json:"airline"`
	FlightNumber    string           `json:"flight_number"`
	Departure       LocationPoint    `json:"departure"`
	Arrival         LocationPoint    `json:"arrival"`
	Duration        DurationInfo     `json:"duration"`
	Stops           int              `json:"stops"`
	Price           PriceInfo        `json:"price"`
	AvailableSeats  int              `json:"available_seats"`
	CabinClass      string           `json:"cabin_class"`
	Aircraft        *string          `json:"aircraft"`
	Amenities       []string         `json:"amenities"`
	Baggage         BaggageInfo      `json:"baggage"`
	BestValueScore  *float64         `json:"best_value_score,omitempty"`
	PriceComparison *PriceComparison `json:"price_comparison,omitempty"`
}

// Validate enforces Level-2 universal domain invariants on a normalized Flight.
func (f Flight) Validate() error {
	if strings.TrimSpace(f.ID) == "" {
		return fmt.Errorf("%w: missing flight id", ErrInvalidFlight)
	}
	if strings.TrimSpace(f.Provider) == "" {
		return fmt.Errorf("%w: missing provider", ErrInvalidFlight)
	}
	if strings.TrimSpace(f.FlightNumber) == "" {
		return fmt.Errorf("%w: missing flight_number", ErrInvalidFlight)
	}
	if strings.TrimSpace(f.Airline.Name) == "" || strings.TrimSpace(f.Airline.Code) == "" {
		return fmt.Errorf("%w: missing airline name or code", ErrInvalidFlight)
	}
	if strings.TrimSpace(f.Departure.Airport) == "" || strings.TrimSpace(f.Arrival.Airport) == "" {
		return fmt.Errorf("%w: missing departure or arrival airport", ErrInvalidFlight)
	}
	if strings.EqualFold(f.Departure.Airport, f.Arrival.Airport) {
		return fmt.Errorf("%w: origin and destination airport cannot match (%s)", ErrInvalidFlight, f.Departure.Airport)
	}
	if f.Departure.Timestamp <= 0 || f.Arrival.Timestamp <= 0 {
		return fmt.Errorf("%w: departure and arrival timestamps must be positive", ErrInvalidFlight)
	}
	if f.Arrival.Timestamp <= f.Departure.Timestamp {
		return fmt.Errorf("%w: arrival (%d) must be after departure (%d)", ErrInvalidFlight, f.Arrival.Timestamp, f.Departure.Timestamp)
	}
	if f.Duration.TotalMinutes <= 0 {
		return fmt.Errorf("%w: duration must be positive", ErrInvalidFlight)
	}
	if f.Stops < 0 {
		return fmt.Errorf("%w: stops cannot be negative", ErrInvalidFlight)
	}
	if f.Price.Amount <= 0 || strings.TrimSpace(f.Price.Currency) == "" {
		return fmt.Errorf("%w: price amount and currency must be valid", ErrInvalidFlight)
	}
	if f.AvailableSeats < 0 {
		return fmt.Errorf("%w: available_seats cannot be negative", ErrInvalidFlight)
	}
	return nil
}
