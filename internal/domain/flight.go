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

// Validate checks that airline name and IATA code are present.
func (a Airline) Validate() error {
	if strings.TrimSpace(a.Name) == "" || strings.TrimSpace(a.Code) == "" {
		return fmt.Errorf("%w: missing airline name or code", ErrInvalidFlight)
	}
	return nil
}

// LocationPoint represents a departure or arrival airport and localized time.
type LocationPoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

// Validate checks that airport code and positive timestamp are present.
func (l LocationPoint) Validate() error {
	if strings.TrimSpace(l.Airport) == "" {
		return fmt.Errorf("%w: missing airport code", ErrInvalidFlight)
	}
	if l.Timestamp <= 0 {
		return fmt.Errorf("%w: timestamp must be positive", ErrInvalidFlight)
	}
	return nil
}

// DurationInfo captures total trip duration in minutes and human-readable form.
type DurationInfo struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

// Validate checks that total duration is positive.
func (d DurationInfo) Validate() error {
	if d.TotalMinutes <= 0 {
		return fmt.Errorf("%w: duration must be positive", ErrInvalidFlight)
	}
	return nil
}

// PriceInfo represents fare pricing and optional formatted display string.
type PriceInfo struct {
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Formatted string `json:"formatted,omitempty"`
}

// Validate checks that price amount is positive and currency is specified.
func (p PriceInfo) Validate() error {
	if p.Amount <= 0 || strings.TrimSpace(p.Currency) == "" {
		return fmt.Errorf("%w: price amount and currency must be valid", ErrInvalidFlight)
	}
	return nil
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
// Fields are grouped logically by domain concept (identity, route, duration, price, metadata).
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

// Validate enforces Level-2 universal domain invariants by composing value-object
// checks with cross-field flight invariants.
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
	if err := f.Airline.Validate(); err != nil {
		return err
	}
	if err := f.Departure.Validate(); err != nil {
		return fmt.Errorf("departure: %w", err)
	}
	if err := f.Arrival.Validate(); err != nil {
		return fmt.Errorf("arrival: %w", err)
	}
	if strings.EqualFold(f.Departure.Airport, f.Arrival.Airport) {
		return fmt.Errorf("%w: origin and destination airport cannot match (%s)", ErrInvalidFlight, f.Departure.Airport)
	}
	if f.Arrival.Timestamp <= f.Departure.Timestamp {
		return fmt.Errorf("%w: arrival (%d) must be after departure (%d)", ErrInvalidFlight, f.Arrival.Timestamp, f.Departure.Timestamp)
	}
	if err := f.Duration.Validate(); err != nil {
		return err
	}
	if f.Stops < 0 {
		return fmt.Errorf("%w: stops cannot be negative", ErrInvalidFlight)
	}
	if err := f.Price.Validate(); err != nil {
		return err
	}
	if f.AvailableSeats < 0 {
		return fmt.Errorf("%w: available_seats cannot be negative", ErrInvalidFlight)
	}
	return nil
}
