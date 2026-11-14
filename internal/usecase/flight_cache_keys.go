package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
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

func canonicalCriteriaSuffix(c domain.SearchCriteria) string {
	base := fmt.Sprintf("%s:%s:%s:%d:%s",
		strings.ToUpper(strings.TrimSpace(c.Origin)),
		strings.ToUpper(strings.TrimSpace(c.Destination)),
		strings.TrimSpace(c.DepartureDate),
		c.Passengers,
		strings.ToLower(strings.TrimSpace(c.CabinClass)),
	)
	if c.ReturnDate != nil {
		if ret := strings.TrimSpace(*c.ReturnDate); ret != "" {
			return base + ":" + ret
		}
	}
	return base
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

func filterValidFlights(flights []domain.Flight) []domain.Flight {
	valid := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		if err := f.Validate(); err != nil {
			continue
		}
		if f.Amenities == nil {
			f.Amenities = []string{}
		}
		valid = append(valid, f)
	}
	return valid
}

func cloneFlights(flights []domain.Flight) []domain.Flight {
	cloned := make([]domain.Flight, len(flights))
	copy(cloned, flights)
	return cloned
}
