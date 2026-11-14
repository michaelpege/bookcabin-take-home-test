package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

type memoryTestCache struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newMemoryTestCache() *memoryTestCache {
	return &memoryTestCache{data: make(map[string][]byte)}
}

func (c *memoryTestCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	if !ok {
		return nil, domain.ErrCacheMiss
	}
	out := make([]byte, len(val))
	copy(out, val)
	return out, nil
}

func (c *memoryTestCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	c.data[key] = cp
	return nil
}

type failingTestCache struct {
	err error
}

func (f failingTestCache) Get(_ context.Context, _ string) ([]byte, error) { return nil, f.err }
func (f failingTestCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return f.err
}

type countingProvider struct {
	name  string
	delay time.Duration
	fn    func(call int) ([]domain.Flight, error)
	calls atomic.Int32
}

func (p *countingProvider) Name() string { return p.name }

func (p *countingProvider) SearchFlights(ctx context.Context, _ domain.SearchCriteria) ([]domain.Flight, error) {
	if p.delay > 0 {
		timer := time.NewTimer(p.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	n := int(p.calls.Add(1))
	return p.fn(n)
}

func TestFlightAggregatorL1CacheHitAcrossFilterSort(t *testing.T) {
	cache := newMemoryTestCache()
	p := &countingProvider{
		name: "Garuda Indonesia",
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{
				sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000),
				sampleFlight("GA2", "Garuda Indonesia", "GA410", 850000),
			}, nil
		},
	}

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	agg := NewFlightAggregator([]domain.FlightProvider{p}, log, WithCache(cache))

	// First request: cache miss, writes L1 + Tier-2.
	res1, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{SortBy: domain.SortPriceAsc})
	if err != nil {
		t.Fatalf("first SearchFlights error: %v", err)
	}
	if res1.Metadata.CacheHit {
		t.Fatalf("expected CacheHit=false on cold cache")
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("expected 1 provider call, got %d", got)
	}

	// Second request with a filter and reverse sort hits L1 without calling provider.
	maxPrice := int64(900000)
	res2, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{
		Filters: domain.FilterOptions{MaxPrice: &maxPrice},
		SortBy:  domain.SortPriceDesc,
	})
	if err != nil {
		t.Fatalf("second SearchFlights error: %v", err)
	}
	if !res2.Metadata.CacheHit {
		t.Fatalf("expected CacheHit=true on warm L1 cache")
	}
	if res2.Metadata.TotalResults != 1 || res2.Flights[0].FlightNumber != "GA410" {
		t.Fatalf("unexpected filtered L1 result: %+v", res2.Flights)
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("expected provider still called once on L1 hit, got %d", got)
	}
}

func TestFlightAggregatorDegradedSkipsL1ButCachesTier2(t *testing.T) {
	cache := newMemoryTestCache()
	healthy := &countingProvider{
		name: "Garuda Indonesia",
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000)}, nil
		},
	}
	flaky := &countingProvider{
		name: "AirAsia",
		fn: func(call int) ([]domain.Flight, error) {
			if call == 1 {
				return nil, errors.New("transient failure")
			}
			return []domain.Flight{sampleFlight("QZ1", "AirAsia", "QZ532", 595000)}, nil
		},
	}

	agg := NewFlightAggregator([]domain.FlightProvider{healthy, flaky}, nil, WithCache(cache))

	// Search 1: degraded (flaky fails). Healthy cached in Tier-2; L1 skipped.
	res1, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if err != nil {
		t.Fatalf("degraded SearchFlights unexpected err: %v", err)
	}
	if res1.Metadata.ProvidersSucceeded != 1 || res1.Metadata.ProvidersFailed != 1 {
		t.Fatalf("unexpected degraded counts: %+v", res1.Metadata)
	}
	if _, l1Err := cache.Get(context.Background(), baseSearchCacheKey(validCriteria())); !errors.Is(l1Err, domain.ErrCacheMiss) {
		t.Fatalf("expected L1 cache to be skipped on degraded response, got %v", l1Err)
	}

	// Search 2: healthy served from Tier-2 (calls stays 1); flaky recovered; L1 populated.
	res2, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if err != nil {
		t.Fatalf("recovery SearchFlights unexpected err: %v", err)
	}
	if res2.Metadata.CacheHit || res2.Metadata.ProvidersSucceeded != 2 || res2.Metadata.TotalResults != 2 {
		t.Fatalf("unexpected recovery metadata: %+v", res2.Metadata)
	}
	if healthy.calls.Load() != 1 {
		t.Fatalf("expected healthy provider served from Tier-2 cache (1 call), got %d", healthy.calls.Load())
	}
	if flaky.calls.Load() != 2 {
		t.Fatalf("expected flaky provider retried once (2 calls), got %d", flaky.calls.Load())
	}

	// Search 3: hits L1 with zero additional provider calls.
	res3, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if err != nil {
		t.Fatalf("warm SearchFlights unexpected err: %v", err)
	}
	if !res3.Metadata.CacheHit || healthy.calls.Load() != 1 || flaky.calls.Load() != 2 {
		t.Fatalf("expected L1 hit with no new provider calls, got cacheHit=%v calls=(%d,%d)",
			res3.Metadata.CacheHit, healthy.calls.Load(), flaky.calls.Load())
	}
}

func TestFlightAggregatorCacheFailOpenWhenCacheErrors(t *testing.T) {
	p := &countingProvider{
		name: "Lion Air",
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("JT1", "Lion Air", "JT742", 890000)}, nil
		},
	}

	agg := NewFlightAggregator(
		[]domain.FlightProvider{p},
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		WithCache(failingTestCache{err: errors.New("redis connection refused")}),
	)

	res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
	if err != nil {
		t.Fatalf("expected fail-open live search on cache error, got %v", err)
	}
	if res.Metadata.CacheHit || res.Metadata.TotalResults != 1 {
		t.Fatalf("unexpected fail-open result: %+v", res.Metadata)
	}
}

func TestFlightAggregatorSingleflightCoalescesMisses(t *testing.T) {
	cache := newMemoryTestCache()
	p := &countingProvider{
		name:  "Batik Air",
		delay: 40 * time.Millisecond,
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{
				sampleFlight("ID1", "Batik Air", "ID6514", 1100000),
				sampleFlight("ID2", "Batik Air", "ID6520", 1180000),
			}, nil
		},
	}

	agg := NewFlightAggregator([]domain.FlightProvider{p}, nil, WithCache(cache))

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{SortBy: domain.SortBestValue})
			if err != nil {
				errs <- err
				return
			}
			if res.Metadata.TotalResults != 2 {
				errs <- errors.New("unexpected total results")
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent SearchFlights failed: %v", err)
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("expected singleflight to coalesce 8 misses into 1 provider call, got %d", got)
	}
}
