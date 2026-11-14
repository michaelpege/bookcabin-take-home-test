package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

func benchmarkFlights(n int) []domain.Flight {
	providers := []string{"Garuda Indonesia", "Lion Air", "Batik Air", "AirAsia"}
	codes := []string{"GA", "JT", "ID", "QZ"}
	out := make([]domain.Flight, n)
	for i := range n {
		pIdx := i % len(providers)
		f := sampleFlight(
			fmt.Sprintf("%s%d_%s", codes[pIdx], 400+i, providers[pIdx]),
			providers[pIdx],
			fmt.Sprintf("%s%d", codes[pIdx], 400+i),
			int64(485000+(i%20)*65000),
		)
		f.Stops = i % 2
		f.Duration.TotalMinutes = 95 + (i%12)*15
		f.Departure.Timestamp += int64((i % 10) * 1800)
		f.Arrival.Timestamp = f.Departure.Timestamp + int64(f.Duration.TotalMinutes*60)
		if i%3 == 0 {
			f.Amenities = []string{"baggage", "meal"}
		}
		out[i] = f
	}
	return out
}

func BenchmarkFilterFlights_100(b *testing.B) {
	benchmarkFilterFlights(b, 100)
}

func BenchmarkFilterFlights_1000(b *testing.B) {
	benchmarkFilterFlights(b, 1000)
}

func benchmarkFilterFlights(b *testing.B, n int) {
	flights := benchmarkFlights(n)
	maxPrice := int64(1300000)
	maxStops := 0
	opts := domain.FilterOptions{
		MaxPrice: &maxPrice,
		MaxStops: &maxStops,
		Airlines: []string{"GA", "AirAsia"},
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = FilterFlights(flights, opts)
	}
}

func BenchmarkComparePrices_1000(b *testing.B) {
	base := benchmarkFlights(1000)
	work := make([]domain.Flight, len(base))

	b.ReportAllocs()
	for b.Loop() {
		copy(work, base)
		ComparePrices(work)
	}
}

func BenchmarkScoreBestValue_1000(b *testing.B) {
	base := benchmarkFlights(1000)
	work := make([]domain.Flight, len(base))

	b.ReportAllocs()
	for b.Loop() {
		copy(work, base)
		ScoreBestValue(work)
	}
}

func BenchmarkSortFlights_1000(b *testing.B) {
	base := benchmarkFlights(1000)
	ScoreBestValue(base)
	work := make([]domain.Flight, len(base))

	b.ReportAllocs()
	for b.Loop() {
		copy(work, base)
		SortFlights(work, domain.SortBestValue)
	}
}

func BenchmarkSearchFlights_ColdScatterGather(b *testing.B) {
	pFlights := benchmarkFlights(16)
	providers := []domain.FlightProvider{
		fakeProvider{name: "Garuda Indonesia", flights: pFlights[:4]},
		fakeProvider{name: "Lion Air", flights: pFlights[4:8]},
		fakeProvider{name: "Batik Air", flights: pFlights[8:12]},
		fakeProvider{name: "AirAsia", flights: pFlights[12:16]},
	}
	agg := NewFlightAggregator(providers, nil)
	criteria := validCriteria()
	opts := SearchOptions{SortBy: domain.SortBestValue}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := agg.SearchFlights(context.Background(), criteria, opts); err != nil {
			b.Fatalf("SearchFlights failed: %v", err)
		}
	}
}

func BenchmarkSearchFlights_WarmL1Cache(b *testing.B) {
	cache := newMemoryTestCache()
	pFlights := benchmarkFlights(16)
	providers := []domain.FlightProvider{
		fakeProvider{name: "Garuda Indonesia", flights: pFlights[:4]},
		fakeProvider{name: "Lion Air", flights: pFlights[4:8]},
		fakeProvider{name: "Batik Air", flights: pFlights[8:12]},
		fakeProvider{name: "AirAsia", flights: pFlights[12:16]},
	}
	agg := NewFlightAggregator(providers, nil, WithCache(cache))
	criteria := validCriteria()
	maxStops := 0
	opts := SearchOptions{
		Filters: domain.FilterOptions{MaxStops: &maxStops},
		SortBy:  domain.SortBestValue,
	}

	// Warm L1 cache before entering the timed loop.
	res, err := agg.SearchFlights(context.Background(), criteria, opts)
	if err != nil {
		b.Fatalf("warm-up SearchFlights failed: %v", err)
	}
	if res.Metadata.CacheHit {
		b.Fatalf("expected warm-up call to miss cache")
	}

	b.ReportAllocs()
	for b.Loop() {
		out, err := agg.SearchFlights(context.Background(), criteria, opts)
		if err != nil {
			b.Fatalf("cached SearchFlights failed: %v", err)
		}
		if !out.Metadata.CacheHit {
			b.Fatalf("expected CacheHit=true in warm benchmark")
		}
	}
}
