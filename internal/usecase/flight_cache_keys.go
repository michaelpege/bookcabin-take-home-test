package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
)

const (
	defaultBaseCacheTTL     = 2 * time.Minute
	defaultProviderCacheTTL = 5 * time.Minute
)

type cachedBaseSearch struct {
	Flights  []domain.Flight       `json:"flights"`
	Metadata domain.SearchMetadata `json:"metadata"`
}

type noopFlightCache struct{}

func (noopFlightCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, domain.ErrCacheMiss
}

func (noopFlightCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func baseSearchCacheKey(c domain.SearchCriteria) string {
	return fmt.Sprintf("flight_search:%s", canonicalCriteriaSuffix(c))
}

func providerSearchCacheKey(providerName string, c domain.SearchCriteria) string {
	return fmt.Sprintf("provider:%s:%s", sanitizeKeySegment(providerName), canonicalCriteriaSuffix(c))
}

// canonicalCriteriaSuffix builds the deterministic cache key suffix from outbound search
// dimensions. Unused ReturnDate is intentionally excluded because outbound provider queries
// depend solely on (origin, destination, departure_date, passengers, cabin_class); keeping it
// out prevents distinct return_date requests from polluting the cache with duplicate keys.
func canonicalCriteriaSuffix(c domain.SearchCriteria) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s",
		strings.ToUpper(strings.TrimSpace(c.Origin)),
		strings.ToUpper(strings.TrimSpace(c.Destination)),
		strings.TrimSpace(c.DepartureDate),
		c.Passengers,
		strings.ToLower(strings.TrimSpace(c.CabinClass)),
	)
}

func sanitizeKeySegment(raw string) string {
	cleaned := strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(" ", "_", ":", "_")
	return replacer.Replace(cleaned)
}

func marshalBaseSearch(flights []domain.Flight, meta domain.SearchMetadata) ([]byte, error) {
	meta.ProviderErrors = nil
	return json.Marshal(cachedBaseSearch{
		Flights:  flights,
		Metadata: meta,
	})
}

func unmarshalBaseSearch(raw []byte) (cachedBaseSearch, error) {
	var entry cachedBaseSearch
	if err := json.Unmarshal(raw, &entry); err != nil {
		return cachedBaseSearch{}, err
	}
	if entry.Flights == nil {
		entry.Flights = []domain.Flight{}
	}
	return entry, nil
}

func filterValidFlightsForCriteria(flights []domain.Flight, criteria domain.SearchCriteria) []domain.Flight {
	valid := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		if err := f.Validate(); err != nil {
			continue
		}
		if !matchesSearchCriteria(f, criteria) {
			continue
		}
		if f.Amenities == nil {
			f.Amenities = []string{}
		}
		valid = append(valid, f)
	}
	return valid
}

func matchesSearchCriteria(f domain.Flight, c domain.SearchCriteria) bool {
	if c.Origin != "" && !strings.EqualFold(f.Departure.Airport, c.Origin) {
		return false
	}
	if c.Destination != "" && !strings.EqualFold(f.Arrival.Airport, c.Destination) {
		return false
	}
	if c.DepartureDate != "" && !timeutil.MatchesDepartureDate(f.Departure.Datetime, "", f.Departure.Airport, c.DepartureDate) {
		return false
	}
	if c.CabinClass != "" && !strings.EqualFold(f.CabinClass, c.CabinClass) {
		return false
	}
	if c.Passengers > 0 && f.AvailableSeats < c.Passengers {
		return false
	}
	return true
}

// cloneFlights performs a deep copy of each Flight (including mutable slice and pointer
// fields) so concurrent singleflight callers never alias shared state.
func cloneFlights(flights []domain.Flight) []domain.Flight {
	if flights == nil {
		return []domain.Flight{}
	}
	cloned := make([]domain.Flight, len(flights))
	for i, f := range flights {
		if f.Aircraft != nil {
			v := *f.Aircraft
			f.Aircraft = &v
		}
		if f.Amenities != nil {
			amen := make([]string, len(f.Amenities))
			copy(amen, f.Amenities)
			f.Amenities = amen
		} else {
			f.Amenities = []string{}
		}
		if f.BestValueScore != nil {
			v := *f.BestValueScore
			f.BestValueScore = &v
		}
		if f.PriceComparison != nil {
			pc := *f.PriceComparison
			f.PriceComparison = &pc
		}
		cloned[i] = f
	}
	return cloned
}

func cloneMetadata(meta domain.SearchMetadata) domain.SearchMetadata {
	if meta.ProviderErrors != nil {
		meta.ProviderErrors = maps.Clone(meta.ProviderErrors)
	}
	return meta
}
