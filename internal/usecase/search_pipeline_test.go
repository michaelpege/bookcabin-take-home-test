package usecase

import (
	"testing"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

func buildTestFlights() []domain.Flight {
	f1 := sampleFlight("GA400_Garuda", "Garuda Indonesia", "GA400", 1250000)
	f1.Duration = domain.DurationInfo{TotalMinutes: 110, Formatted: "1h 50m"}
	f1.Departure = domain.LocationPoint{Airport: "CGK", City: "Jakarta", Datetime: "2025-12-15T06:00:00+07:00", Timestamp: 1765753200}
	f1.Arrival = domain.LocationPoint{Airport: "DPS", City: "Denpasar", Datetime: "2025-12-15T08:50:00+08:00", Timestamp: 1765759800}
	f1.Amenities = []string{"wifi", "meal", "entertainment"}
	f1.CabinClass = "economy"

	f2 := sampleFlight("QZ532_AirAsia", "AirAsia", "QZ532", 595000)
	f2.Duration = domain.DurationInfo{TotalMinutes: 100, Formatted: "1h 40m"}
	f2.Departure = domain.LocationPoint{Airport: "CGK", City: "Jakarta", Datetime: "2025-12-15T19:30:00+07:00", Timestamp: 1765801800}
	f2.Arrival = domain.LocationPoint{Airport: "DPS", City: "Denpasar", Datetime: "2025-12-15T22:10:00+08:00", Timestamp: 1765807800}
	f2.CabinClass = "economy"

	f3 := sampleFlight("JT650_Lion", "Lion Air", "JT650", 780000)
	f3.Stops = 1
	f3.Duration = domain.DurationInfo{TotalMinutes: 230, Formatted: "3h 50m"}
	f3.Departure = domain.LocationPoint{Airport: "CGK", City: "Jakarta", Datetime: "2025-12-15T16:20:00+07:00", Timestamp: 1765790400}
	f3.Arrival = domain.LocationPoint{Airport: "DPS", City: "Denpasar", Datetime: "2025-12-15T21:10:00+08:00", Timestamp: 1765804200}
	f3.CabinClass = "economy"

	return []domain.Flight{f1, f2, f3}
}

func TestFilterFlightsAllPredicates(t *testing.T) {
	flights := buildTestFlights()

	minP := int64(600000)
	maxP := int64(1300000)
	maxStops := 0
	maxDur := 120
	out := FilterFlights(flights, domain.FilterOptions{
		MinPrice:           &minP,
		MaxPrice:           &maxP,
		MaxStops:           &maxStops,
		MaxDurationMinutes: &maxDur,
		Airlines:           []string{"ga", "qz"},
		DepartureWindow:    &domain.TimeRange{Start: "05:30", End: "09:00"},
	})
	if len(out) != 1 || out[0].FlightNumber != "GA400" {
		t.Fatalf("expected only GA400 to pass all predicates, got %+v", out)
	}

	// Non-matching filter must return empty non-nil slice.
	none := FilterFlights(flights, domain.FilterOptions{Stops: []int{2}})
	if none == nil || len(none) != 0 {
		t.Fatalf("expected non-nil empty slice, got %#v", none)
	}
}

func TestComparePricesAndScoreBestValue(t *testing.T) {
	flights := buildTestFlights()
	ComparePrices(flights)
	ScoreBestValue(flights)

	for _, f := range flights {
		if f.PriceComparison == nil {
			t.Fatalf("expected PriceComparison populated for %s", f.FlightNumber)
		}
		if f.PriceComparison.CheapestAmount != 595000 || f.PriceComparison.CheapestProvider != "AirAsia" {
			t.Fatalf("unexpected cheapest baseline: %+v", f.PriceComparison)
		}
		if f.PriceComparison.ProviderCount != 3 {
			t.Fatalf("expected ProviderCount=3, got %d", f.PriceComparison.ProviderCount)
		}
		if f.BestValueScore == nil || *f.BestValueScore < 0 || *f.BestValueScore > 1 {
			t.Fatalf("expected BestValueScore in [0,1], got %v", f.BestValueScore)
		}
	}

	if flights[0].PriceComparison.PriceDiff != 1250000-595000 {
		t.Fatalf("unexpected PriceDiff for GA400: %d", flights[0].PriceComparison.PriceDiff)
	}

	// Single provider group omits cross-provider comparison.
	single := []domain.Flight{flights[0]}
	ComparePrices(single)
	if single[0].PriceComparison != nil {
		t.Fatalf("expected nil PriceComparison for single provider, got %+v", single[0].PriceComparison)
	}
}

func TestSortFlightsAllModes(t *testing.T) {
	tests := []struct {
		mode      domain.SortBy
		wantFirst string
	}{
		{mode: "", wantFirst: "QZ532"},
		{mode: domain.SortPriceAsc, wantFirst: "QZ532"},
		{mode: domain.SortPriceDesc, wantFirst: "GA400"},
		{mode: domain.SortDurationAsc, wantFirst: "QZ532"},
		{mode: domain.SortDurationDesc, wantFirst: "JT650"},
		{mode: domain.SortDepartureAsc, wantFirst: "GA400"},
		{mode: domain.SortDepartureDesc, wantFirst: "QZ532"},
		{mode: domain.SortArrivalAsc, wantFirst: "GA400"},
		{mode: domain.SortArrivalDesc, wantFirst: "QZ532"},
		{mode: domain.SortBestValue, wantFirst: "QZ532"},
	}

	for _, tc := range tests {
		t.Run(string(tc.mode), func(t *testing.T) {
			flights := buildTestFlights()
			ScoreBestValue(flights)
			SortFlights(flights, tc.mode)
			if flights[0].FlightNumber != tc.wantFirst {
				t.Fatalf("mode %q: expected %s first, got %s", tc.mode, tc.wantFirst, flights[0].FlightNumber)
			}
		})
	}
}

func TestScoreBestValueSingleFlightAndZeroSpread(t *testing.T) {
	single := []domain.Flight{sampleFlight("GA400_Garuda", "Garuda Indonesia", "GA400", 1250000)}
	ScoreBestValue(single)
	if single[0].BestValueScore == nil || *single[0].BestValueScore != 0.5 {
		t.Fatalf("expected neutral 0.5 score for single flight, got %v", single[0].BestValueScore)
	}

	identical := []domain.Flight{
		sampleFlight("F1", "Garuda Indonesia", "GA400", 1000000),
		sampleFlight("F2", "Lion Air", "JT740", 1000000),
	}
	ScoreBestValue(identical)
	for i, f := range identical {
		if f.BestValueScore == nil || *f.BestValueScore != 0.5 {
			t.Fatalf("flight %d: expected neutral 0.5 score on zero-spread cohort, got %v", i, f.BestValueScore)
		}
	}
}

func TestComparePricesExactFlightAndScheduleBeforeMarketFallback(t *testing.T) {
	gaA := sampleFlight("GA400_A", "OTA Alpha", "GA-400", 1250000)
	gaB := sampleFlight("GA400_B", "OTA Beta", "GA 400", 1180000)
	qz := sampleFlight("QZ520_C", "AirAsia", "QZ520", 650000)
	qz.Departure.Timestamp = 1765748700
	qz.Arrival.Timestamp = 1765754700

	flights := []domain.Flight{gaA, gaB, qz}
	ComparePrices(flights)

	// GA400 entries match exact flight number across OTA Alpha & OTA Beta (not AirAsia QZ520).
	if flights[0].PriceComparison == nil || flights[0].PriceComparison.CheapestProvider != "OTA Beta" ||
		flights[0].PriceComparison.CheapestAmount != 1180000 || flights[0].PriceComparison.PriceDiff != 70000 ||
		flights[0].PriceComparison.ProviderCount != 2 {
		t.Fatalf("expected exact-flight comparison for GA400, got %+v", flights[0].PriceComparison)
	}
	// QZ520 falls back gracefully to route-market baseline across all 3 providers.
	if flights[2].PriceComparison == nil || flights[2].PriceComparison.CheapestProvider != "AirAsia" ||
		flights[2].PriceComparison.ProviderCount != 3 {
		t.Fatalf("expected route-market fallback for QZ520, got %+v", flights[2].PriceComparison)
	}

	// Codeshares sharing exact schedule timestamps also group at Level 1.
	cs1 := sampleFlight("CS1", "Garuda Indonesia", "GA400", 1300000)
	cs2 := sampleFlight("CS2", "Partner Air", "PA9400", 1150000)
	other := sampleFlight("QZ99", "AirAsia", "QZ524", 700000)
	other.Departure.Timestamp = 1765767600
	other.Arrival.Timestamp = 1765773900

	csFlights := []domain.Flight{cs1, cs2, other}
	ComparePrices(csFlights)
	if csFlights[0].PriceComparison == nil || csFlights[0].PriceComparison.CheapestProvider != "Partner Air" ||
		csFlights[0].PriceComparison.ProviderCount != 2 {
		t.Fatalf("expected identical-schedule codeshare comparison, got %+v", csFlights[0].PriceComparison)
	}
}
