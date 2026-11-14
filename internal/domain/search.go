package domain

import (
	"fmt"
	"strings"
	"time"
)

// SortBy defines supported post-aggregation sort orders.
type SortBy string

const (
	SortPriceAsc      SortBy = "price_asc"
	SortPriceDesc     SortBy = "price_desc"
	SortDurationAsc   SortBy = "duration_asc"
	SortDurationDesc  SortBy = "duration_desc"
	SortDepartureAsc  SortBy = "departure_asc"
	SortDepartureDesc SortBy = "departure_desc"
	SortArrivalAsc    SortBy = "arrival_asc"
	SortArrivalDesc   SortBy = "arrival_desc"
	SortBestValue     SortBy = "best_value"
)

// SearchCriteria holds the canonical flight search parameters.
type SearchCriteria struct {
	Origin        string  `json:"origin"`
	Destination   string  `json:"destination"`
	DepartureDate string  `json:"departure_date"`
	ReturnDate    *string `json:"return_date,omitempty"`
	Passengers    int     `json:"passengers"`
	CabinClass    string  `json:"cabin_class"`
}

// Validate checks that required search parameters are well-formed.
func (c SearchCriteria) Validate() error {
	origin := strings.TrimSpace(c.Origin)
	dest := strings.TrimSpace(c.Destination)
	if len(origin) != 3 || len(dest) != 3 {
		return fmt.Errorf("%w: origin and destination must be 3-letter IATA codes", ErrInvalidSearchCriteria)
	}
	if strings.EqualFold(origin, dest) {
		return fmt.Errorf("%w: origin and destination cannot be the same", ErrInvalidSearchCriteria)
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(c.DepartureDate)); err != nil {
		return fmt.Errorf("%w: departure_date must be YYYY-MM-DD", ErrInvalidSearchCriteria)
	}
	if c.Passengers < 1 {
		return fmt.Errorf("%w: passengers must be at least 1", ErrInvalidSearchCriteria)
	}
	if strings.TrimSpace(c.CabinClass) == "" {
		return fmt.Errorf("%w: cabin_class is required", ErrInvalidSearchCriteria)
	}
	return nil
}

// TimeRange represents an inclusive HH:MM local time window (e.g., "06:00" to "12:00").
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// FilterOptions specifies post-aggregation filtering predicates.
type FilterOptions struct {
	MinPrice           *int64     `json:"min_price,omitempty"`
	MaxPrice           *int64     `json:"max_price,omitempty"`
	Stops              []int      `json:"stops,omitempty"`
	MaxStops           *int       `json:"max_stops,omitempty"`
	DepartureWindow    *TimeRange `json:"departure_window,omitempty"`
	ArrivalWindow      *TimeRange `json:"arrival_window,omitempty"`
	Airlines           []string   `json:"airlines,omitempty"`
	MaxDurationMinutes *int       `json:"max_duration_minutes,omitempty"`
}

// SearchMetadata reports aggregation, latency, cache, and partial-failure stats.
type SearchMetadata struct {
	TotalResults       int               `json:"total_results"`
	ProvidersQueried   int               `json:"providers_queried"`
	ProvidersSucceeded int               `json:"providers_succeeded"`
	ProvidersFailed    int               `json:"providers_failed"`
	SearchTimeMS       int64             `json:"search_time_ms"`
	CacheHit           bool              `json:"cache_hit"`
	ProviderErrors     map[string]string `json:"provider_errors,omitempty"`
}

// SearchResult is the unified response returned by the flight search usecase.
type SearchResult struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       SearchMetadata `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}
