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

// ComparePrices annotates flights with cross-provider price comparisons using a
// two-level matching strategy:
//  1. Exact-flight / identical-schedule matching: if two or more providers offer the
//     same normalized (origin, destination, cabin_class, flight_number) or identical
//     (origin, destination, cabin_class, departure_ts, arrival_ts) schedule (e.g., codeshares),
//     comparison is scoped directly to those matching offerings.
//  2. Route-market fallback: when providers operate disjoint flight numbers and schedules
//     (such as distinct carriers on the same route), comparison falls back to the
//     (origin, destination, cabin_class) market baseline across competing providers.
func ComparePrices(flights []domain.Flight) {
	if len(flights) == 0 {
		return
	}

	exactGroups := make(map[string]*routeGroupStats)
	scheduleGroups := make(map[string]*routeGroupStats)
	routeGroups := make(map[string]*routeGroupStats)

	for _, f := range flights {
		if key := exactFlightGroupKey(f); key != "" {
			recordGroupStat(exactGroups, key, f)
		}
		recordGroupStat(scheduleGroups, scheduleGroupKey(f), f)
		recordGroupStat(routeGroups, comparisonGroupKey(f), f)
	}

	for i := range flights {
		stats := selectComparisonGroup(flights[i], exactGroups, scheduleGroups, routeGroups)
		if stats == nil {
			flights[i].PriceComparison = nil
			continue
		}
		flights[i].PriceComparison = &domain.PriceComparison{
			CheapestProvider: stats.cheapestProvider,
			CheapestAmount:   stats.cheapestAmount,
			PriceDiff:        flights[i].Price.Amount - stats.cheapestAmount,
			ProviderCount:    len(stats.providers),
		}
	}
}

func recordGroupStat(groups map[string]*routeGroupStats, key string, f domain.Flight) {
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

func selectComparisonGroup(
	f domain.Flight,
	exactGroups, scheduleGroups, routeGroups map[string]*routeGroupStats,
) *routeGroupStats {
	if key := exactFlightGroupKey(f); key != "" {
		if stats := exactGroups[key]; stats != nil && len(stats.providers) >= 2 {
			return stats
		}
	}
	if stats := scheduleGroups[scheduleGroupKey(f)]; stats != nil && len(stats.providers) >= 2 {
		return stats
	}
	if stats := routeGroups[comparisonGroupKey(f)]; stats != nil && len(stats.providers) >= 2 {
		return stats
	}
	return nil
}

func exactFlightGroupKey(f domain.Flight) string {
	fn := normalizeFlightNumber(f.FlightNumber)
	if fn == "" {
		return ""
	}
	return fmt.Sprintf("%s|%s", comparisonGroupKey(f), fn)
}

func scheduleGroupKey(f domain.Flight) string {
	return fmt.Sprintf("%s|%d|%d", comparisonGroupKey(f), f.Departure.Timestamp, f.Arrival.Timestamp)
}

func normalizeFlightNumber(raw string) string {
	cleaned := strings.ToUpper(strings.TrimSpace(raw))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	return cleaned
}

func comparisonGroupKey(f domain.Flight) string {
	return fmt.Sprintf("%s|%s|%s",
		strings.ToUpper(strings.TrimSpace(f.Departure.Airport)),
		strings.ToUpper(strings.TrimSpace(f.Arrival.Airport)),
		strings.ToLower(strings.TrimSpace(f.CabinClass)),
	)
}
