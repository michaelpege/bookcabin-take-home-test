package usecase

import (
	"strconv"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// FilterFlights applies post-aggregation predicates in a single O(N) pass and
// always returns a non-nil slice.
func FilterFlights(flights []domain.Flight, opts domain.FilterOptions) []domain.Flight {
	stopsSet := make(map[int]struct{}, len(opts.Stops))
	for _, s := range opts.Stops {
		stopsSet[s] = struct{}{}
	}

	airlineSet := make(map[string]struct{}, len(opts.Airlines))
	for _, a := range opts.Airlines {
		if key := strings.ToUpper(strings.TrimSpace(a)); key != "" {
			airlineSet[key] = struct{}{}
		}
	}

	depWindow, hasDepWindow := parseTimeWindow(opts.DepartureWindow)
	arrWindow, hasArrWindow := parseTimeWindow(opts.ArrivalWindow)

	filtered := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		if !matchesFilter(f, opts, stopsSet, airlineSet, depWindow, hasDepWindow, arrWindow, hasArrWindow) {
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

func matchesFilter(
	f domain.Flight,
	opts domain.FilterOptions,
	stopsSet map[int]struct{},
	airlineSet map[string]struct{},
	depWindow timeWindowMinutes,
	hasDepWindow bool,
	arrWindow timeWindowMinutes,
	hasArrWindow bool,
) bool {
	belowMinPrice := opts.MinPrice != nil && f.Price.Amount < *opts.MinPrice
	if belowMinPrice {
		return false
	}

	aboveMaxPrice := opts.MaxPrice != nil && f.Price.Amount > *opts.MaxPrice
	if aboveMaxPrice {
		return false
	}

	if len(stopsSet) > 0 {
		_, stopsIncluded := stopsSet[f.Stops]
		if !stopsIncluded {
			return false
		}
	}

	exceedsMaxStops := opts.MaxStops != nil && f.Stops > *opts.MaxStops
	if exceedsMaxStops {
		return false
	}

	exceedsMaxDuration := opts.MaxDurationMinutes != nil && f.Duration.TotalMinutes > *opts.MaxDurationMinutes
	if exceedsMaxDuration {
		return false
	}

	airlineExcluded := len(airlineSet) > 0 && !matchesAirline(f, airlineSet)
	if airlineExcluded {
		return false
	}

	outsideDepartureWindow := hasDepWindow && !matchesTimeWindow(f.Departure.Datetime, depWindow)
	if outsideDepartureWindow {
		return false
	}

	outsideArrivalWindow := hasArrWindow && !matchesTimeWindow(f.Arrival.Datetime, arrWindow)
	if outsideArrivalWindow {
		return false
	}

	return true
}

func matchesAirline(f domain.Flight, allowed map[string]struct{}) bool {
	for _, candidate := range []string{f.Airline.Code, f.Airline.Name, f.Provider} {
		if _, ok := allowed[strings.ToUpper(strings.TrimSpace(candidate))]; ok {
			return true
		}
	}
	return false
}

type timeWindowMinutes struct {
	start int
	end   int
}

func parseTimeWindow(tr *domain.TimeRange) (timeWindowMinutes, bool) {
	if tr == nil {
		return timeWindowMinutes{}, false
	}
	start, okStart := parseHHMM(tr.Start)
	end, okEnd := parseHHMM(tr.End)
	if !okStart || !okEnd {
		return timeWindowMinutes{}, false
	}
	return timeWindowMinutes{start: start, end: end}, true
}

func parseHHMM(raw string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, errH := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func matchesTimeWindow(rfc3339 string, window timeWindowMinutes) bool {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	if window.start <= window.end {
		return mins >= window.start && mins <= window.end
	}
	// Overnight wrap (e.g., 22:00 to 05:00).
	return mins >= window.start || mins <= window.end
}
