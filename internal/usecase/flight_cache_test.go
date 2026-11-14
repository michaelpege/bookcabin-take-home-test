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

func TestFlightAggregatorSingleflightLeaderTimeoutDoesNotPoisonFollower(t *testing.T) {
	cache := newMemoryTestCache()
	p := &countingProvider{
		name:  "Garuda Indonesia",
		delay: 70 * time.Millisecond,
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000)}, nil
		},
	}

	agg := NewFlightAggregator([]domain.FlightProvider{p}, nil, WithCache(cache))

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer leaderCancel()
	followerCtx, followerCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer followerCancel()

	type outcome struct {
		res domain.SearchResult
		err error
	}
	leaderCh := make(chan outcome, 1)
	followerCh := make(chan outcome, 1)

	go func() {
		res, err := agg.SearchFlights(leaderCtx, validCriteria(), SearchOptions{})
		leaderCh <- outcome{res: res, err: err}
	}()
	time.Sleep(5 * time.Millisecond)
	go func() {
		res, err := agg.SearchFlights(followerCtx, validCriteria(), SearchOptions{})
		followerCh <- outcome{res: res, err: err}
	}()

	lOut := <-leaderCh
	if !errors.Is(lOut.err, context.DeadlineExceeded) {
		t.Fatalf("expected leader to abort with DeadlineExceeded, got %v", lOut.err)
	}
	fOut := <-followerCh
	if fOut.err != nil {
		t.Fatalf("expected healthy follower to succeed despite leader timeout, got %v", fOut.err)
	}
	if len(fOut.res.Flights) != 1 || fOut.res.Flights[0].FlightNumber != "GA400" {
		t.Fatalf("unexpected follower flights: %+v", fOut.res.Flights)
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("expected shared fetch (1 provider call), got %d", got)
	}
}

func TestFlightAggregatorSingleflightFollowerTimeoutExitsEarly(t *testing.T) {
	p := &countingProvider{
		name:  "Garuda Indonesia",
		delay: 120 * time.Millisecond,
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000)}, nil
		},
	}

	agg := NewFlightAggregator([]domain.FlightProvider{p}, nil)

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer leaderCancel()
	followerCtx, followerCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer followerCancel()

	type outcome struct {
		elapsed time.Duration
		res     domain.SearchResult
		err     error
	}
	leaderCh := make(chan outcome, 1)
	followerCh := make(chan outcome, 1)

	go func() {
		start := time.Now()
		res, err := agg.SearchFlights(leaderCtx, validCriteria(), SearchOptions{})
		leaderCh <- outcome{elapsed: time.Since(start), res: res, err: err}
	}()
	time.Sleep(5 * time.Millisecond)
	go func() {
		start := time.Now()
		res, err := agg.SearchFlights(followerCtx, validCriteria(), SearchOptions{})
		followerCh <- outcome{elapsed: time.Since(start), res: res, err: err}
	}()

	fOut := <-followerCh
	if !errors.Is(fOut.err, context.DeadlineExceeded) {
		t.Fatalf("expected follower DeadlineExceeded, got %v", fOut.err)
	}
	if fOut.elapsed >= 80*time.Millisecond {
		t.Fatalf("expected follower to exit early without waiting for 120ms leader, took %v", fOut.elapsed)
	}

	lOut := <-leaderCh
	if lOut.err != nil || len(lOut.res.Flights) != 1 {
		t.Fatalf("expected leader to complete cleanly, got err=%v flights=%d", lOut.err, len(lOut.res.Flights))
	}
}

func TestFlightAggregatorCanonicalizesCriteriaAndDedupsReturnDateCacheKeys(t *testing.T) {
	cache := newMemoryTestCache()
	p := &countingProvider{
		name: "Garuda Indonesia",
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000)}, nil
		},
	}
	agg := NewFlightAggregator([]domain.FlightProvider{p}, nil, WithCache(cache))

	ret1 := "2025-12-20"
	c1 := domain.SearchCriteria{
		Origin:        " cgk ",
		Destination:   " dps ",
		DepartureDate: "2025-12-15",
		ReturnDate:    &ret1,
		Passengers:    1,
		CabinClass:    " ECONOMY ",
	}
	res1, err := agg.SearchFlights(context.Background(), c1, SearchOptions{})
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if res1.SearchCriteria.Origin != "CGK" || res1.SearchCriteria.Destination != "DPS" || res1.SearchCriteria.CabinClass != "economy" {
		t.Fatalf("expected canonicalized SearchCriteria echoed, got %+v", res1.SearchCriteria)
	}

	ret2 := "2025-12-25"
	c2 := validCriteria()
	c2.ReturnDate = &ret2
	res2, err := agg.SearchFlights(context.Background(), c2, SearchOptions{})
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if !res2.Metadata.CacheHit {
		t.Fatalf("expected L1 cache hit across distinct return_date values")
	}
	if res2.SearchCriteria.ReturnDate == nil || *res2.SearchCriteria.ReturnDate != "2025-12-25" {
		t.Fatalf("expected caller's return_date preserved on cache hit, got %v", res2.SearchCriteria.ReturnDate)
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("expected 1 provider call total, got %d", got)
	}
}

func TestFlightAggregatorDeepClonesCoalescedResults(t *testing.T) {
	aircraft := "Boeing 737"
	f := sampleFlight("GA1", "Garuda Indonesia", "GA400", 1250000)
	f.Aircraft = &aircraft
	f.Amenities = []string{"wifi", "meal"}

	p1 := &countingProvider{
		name:  "Garuda Indonesia",
		delay: 30 * time.Millisecond,
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{f}, nil
		},
	}
	p2 := &countingProvider{
		name:  "Lion Air",
		delay: 30 * time.Millisecond,
		fn: func(_ int) ([]domain.Flight, error) {
			return []domain.Flight{sampleFlight("JT1", "Lion Air", "JT740", 950000)}, nil
		},
	}

	agg := NewFlightAggregator([]domain.FlightProvider{p1, p2}, nil)

	const callers = 4
	results := make([]domain.SearchResult, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := agg.SearchFlights(context.Background(), validCriteria(), SearchOptions{})
			if err != nil {
				t.Errorf("caller %d failed: %v", idx, err)
				return
			}
			// Deliberately mutate caller-owned slices/pointers under -race.
			for j := range res.Flights {
				if len(res.Flights[j].Amenities) > 0 {
					res.Flights[j].Amenities[0] = "mutated"
				}
				if res.Flights[j].Aircraft != nil {
					*res.Flights[j].Aircraft = "mutated"
				}
				if res.Flights[j].BestValueScore != nil {
					*res.Flights[j].BestValueScore = -1
				}
				if res.Flights[j].PriceComparison != nil {
					res.Flights[j].PriceComparison.CheapestAmount = -1
				}
			}
			results[idx] = res
		}(i)
	}
	wg.Wait()
	for idx, res := range results {
		if len(res.Flights) != 2 {
			t.Fatalf("caller %d expected 2 flights, got %d", idx, len(res.Flights))
		}
	}
}
