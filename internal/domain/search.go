package domain

import (
	"fmt"
	"strconv"
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

// Validate checks that sortBy is either empty (defaulting to price_asc) or a known SortBy mode.
func (s SortBy) Validate() error {
	switch SortBy(strings.TrimSpace(string(s))) {
	case "",
		SortPriceAsc,
		SortPriceDesc,
		SortDurationAsc,
		SortDurationDesc,
		SortDepartureAsc,
		SortDepartureDesc,
		SortArrivalAsc,
		SortArrivalDesc,
		SortBestValue:
		return nil
	default:
		return fmt.Errorf("%w: unsupported sort_by %q", ErrSearchInvalidCriteria, s)
	}
}

// SearchCriteria holds the canonical flight search parameters.
type SearchCriteria struct {
	Origin        string  `json:"origin"`
	Destination   string  `json:"destination"`
	DepartureDate string  `json:"departure_date"`
	ReturnDate    *string `json:"return_date,omitempty"`
	Passengers    int     `json:"passengers"`
	CabinClass    string  `json:"cabin_class"`
}

// Normalize returns a canonicalized copy of SearchCriteria with trimmed fields,
// uppercased IATA codes, lowercased cabin_class, and empty return_date collapsed to nil.
func (c SearchCriteria) Normalize() SearchCriteria {
	normalized := SearchCriteria{
		Origin:        strings.ToUpper(strings.TrimSpace(c.Origin)),
		Destination:   strings.ToUpper(strings.TrimSpace(c.Destination)),
		DepartureDate: strings.TrimSpace(c.DepartureDate),
		Passengers:    c.Passengers,
		CabinClass:    strings.ToLower(strings.TrimSpace(c.CabinClass)),
	}
	if c.ReturnDate != nil {
		if ret := strings.TrimSpace(*c.ReturnDate); ret != "" {
			normalized.ReturnDate = &ret
		}
	}
	return normalized
}

// Validate checks that required search parameters are well-formed.
func (c SearchCriteria) Validate() error {
	origin := strings.TrimSpace(c.Origin)
	dest := strings.TrimSpace(c.Destination)
	if !isValidIATACode(origin) || !isValidIATACode(dest) {
		return fmt.Errorf("%w: origin and destination must be 3-letter IATA codes", ErrSearchInvalidCriteria)
	}
	if strings.EqualFold(origin, dest) {
		return fmt.Errorf("%w: origin and destination cannot be the same", ErrSearchInvalidCriteria)
	}
	depDate, err := time.Parse("2006-01-02", strings.TrimSpace(c.DepartureDate))
	if err != nil {
		return fmt.Errorf("%w: departure_date must be YYYY-MM-DD", ErrSearchInvalidCriteria)
	}
	if c.ReturnDate != nil {
		if retStr := strings.TrimSpace(*c.ReturnDate); retStr != "" {
			retDate, err := time.Parse("2006-01-02", retStr)
			if err != nil {
				return fmt.Errorf("%w: return_date must be YYYY-MM-DD", ErrSearchInvalidCriteria)
			}
			if retDate.Before(depDate) {
				return fmt.Errorf("%w: return_date cannot be before departure_date", ErrSearchInvalidCriteria)
			}
		}
	}
	if c.Passengers < 1 {
		return fmt.Errorf("%w: passengers must be at least 1", ErrSearchInvalidCriteria)
	}
	if strings.TrimSpace(c.CabinClass) == "" {
		return fmt.Errorf("%w: cabin_class is required", ErrSearchInvalidCriteria)
	}
	return nil
}

func isValidIATACode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		c := code[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

// TimeRange represents an inclusive HH:MM local time window (e.g., "06:00" to "12:00").
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Validate checks that both Start and End are valid 24-hour HH:MM timestamps.
func (t TimeRange) Validate() error {
	if !isValidHHMM(t.Start) || !isValidHHMM(t.End) {
		return fmt.Errorf("%w: time window must use valid HH:MM bounds", ErrSearchInvalidCriteria)
	}
	return nil
}

func isValidHHMM(raw string) bool {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 || len(parts[1]) != 2 {
		return false
	}
	h, errH := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	return errH == nil && errM == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
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

// Validate checks that filter bounds and nested time windows are coherent and non-negative.
func (f FilterOptions) Validate() error {
	if f.MinPrice != nil && *f.MinPrice < 0 {
		return fmt.Errorf("%w: min_price cannot be negative", ErrSearchInvalidCriteria)
	}
	if f.MaxPrice != nil && *f.MaxPrice < 0 {
		return fmt.Errorf("%w: max_price cannot be negative", ErrSearchInvalidCriteria)
	}
	if f.MinPrice != nil && f.MaxPrice != nil && *f.MinPrice > *f.MaxPrice {
		return fmt.Errorf("%w: min_price (%d) cannot exceed max_price (%d)", ErrSearchInvalidCriteria, *f.MinPrice, *f.MaxPrice)
	}
	if f.MaxStops != nil && *f.MaxStops < 0 {
		return fmt.Errorf("%w: max_stops cannot be negative", ErrSearchInvalidCriteria)
	}
	for _, s := range f.Stops {
		if s < 0 {
			return fmt.Errorf("%w: stops cannot contain negative values (%d)", ErrSearchInvalidCriteria, s)
		}
	}
	if f.MaxDurationMinutes != nil && *f.MaxDurationMinutes <= 0 {
		return fmt.Errorf("%w: max_duration_minutes must be positive", ErrSearchInvalidCriteria)
	}
	if f.DepartureWindow != nil {
		if err := f.DepartureWindow.Validate(); err != nil {
			return fmt.Errorf("departure_window: %w", err)
		}
	}
	if f.ArrivalWindow != nil {
		if err := f.ArrivalWindow.Validate(); err != nil {
			return fmt.Errorf("arrival_window: %w", err)
		}
	}
	return nil
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
