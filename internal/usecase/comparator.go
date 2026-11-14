package usecase

import (
	"fmt"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

type routeGroupStats struct {
	cheapestAmount   int64
	cheapestProvider string
	providers        map[string]struct{}
}

// ComparePrices annotates flights with cross-provider price comparisons per
// (origin, destination, cabin_class) market when two or more providers compete.
func ComparePrices(flights []domain.Flight) {
	if len(flights) == 0 {
		return
	}

	groups := make(map[string]*routeGroupStats)
	for _, f := range flights {
		key := comparisonGroupKey(f)
		stats, exists := groups[key]
		if !exists {
			stats = &routeGroupStats{
				cheapestAmount:   f.Price.Amount,
				cheapestProvider: f.Provider,
				providers:        make(map[string]struct{}),
			}
			groups[key] = stats
		}
		stats.providers[f.Provider] = struct{}{}
		if f.Price.Amount < stats.cheapestAmount ||
			(f.Price.Amount == stats.cheapestAmount && f.Provider < stats.cheapestProvider) {
			stats.cheapestAmount = f.Price.Amount
			stats.cheapestProvider = f.Provider
		}
	}

	for i := range flights {
		stats := groups[comparisonGroupKey(flights[i])]
		providerCount := len(stats.providers)
		if providerCount < 2 {
			flights[i].PriceComparison = nil
			continue
		}
		flights[i].PriceComparison = &domain.PriceComparison{
			CheapestProvider: stats.cheapestProvider,
			CheapestAmount:   stats.cheapestAmount,
			PriceDiff:        flights[i].Price.Amount - stats.cheapestAmount,
			ProviderCount:    providerCount,
		}
	}
}

func comparisonGroupKey(f domain.Flight) string {
	return fmt.Sprintf("%s|%s|%s",
		strings.ToUpper(strings.TrimSpace(f.Departure.Airport)),
		strings.ToUpper(strings.TrimSpace(f.Arrival.Airport)),
		strings.ToLower(strings.TrimSpace(f.CabinClass)),
	)
}
