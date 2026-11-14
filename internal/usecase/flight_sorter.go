package usecase

import (
	"cmp"
	"slices"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// SortFlights orders flights in-place in O(N log N) time according to sortBy,
// defaulting to ascending price with deterministic tie-breakers.
func SortFlights(flights []domain.Flight, sortBy domain.SortBy) {
	slices.SortStableFunc(flights, func(a, b domain.Flight) int {
		var order int
		switch sortBy {
		case domain.SortPriceDesc:
			order = cmp.Compare(b.Price.Amount, a.Price.Amount)
		case domain.SortDurationAsc:
			order = cmp.Compare(a.Duration.TotalMinutes, b.Duration.TotalMinutes)
		case domain.SortDurationDesc:
			order = cmp.Compare(b.Duration.TotalMinutes, a.Duration.TotalMinutes)
		case domain.SortDepartureAsc:
			order = cmp.Compare(a.Departure.Timestamp, b.Departure.Timestamp)
		case domain.SortDepartureDesc:
			order = cmp.Compare(b.Departure.Timestamp, a.Departure.Timestamp)
		case domain.SortArrivalAsc:
			order = cmp.Compare(a.Arrival.Timestamp, b.Arrival.Timestamp)
		case domain.SortArrivalDesc:
			order = cmp.Compare(b.Arrival.Timestamp, a.Arrival.Timestamp)
		case domain.SortBestValue:
			order = cmp.Compare(scoreValue(b), scoreValue(a))
		case domain.SortPriceAsc:
			fallthrough
		default:
			order = cmp.Compare(a.Price.Amount, b.Price.Amount)
		}

		if order != 0 {
			return order
		}
		if c := cmp.Compare(a.Price.Amount, b.Price.Amount); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Duration.TotalMinutes, b.Duration.TotalMinutes); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Departure.Timestamp, b.Departure.Timestamp); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func scoreValue(f domain.Flight) float64 {
	if f.BestValueScore == nil {
		return 0
	}
	return *f.BestValueScore
}
