package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
)

// SearchOptions carries optional post-aggregation filter and sort settings.
type SearchOptions struct {
	Filters domain.FilterOptions
	SortBy  domain.SortBy
}

// AggregatorOption configures optional dependencies or TTL policies on FlightAggregator.
type AggregatorOption func(*FlightAggregator)

// WithCache injects a domain.FlightCache adapter (Redis, in-memory, or noop).
func WithCache(cache domain.FlightCache) AggregatorOption {
	return func(a *FlightAggregator) {
		if cache != nil {
			a.cache = cache
		}
	}
}

// WithCacheTTLs overrides the L1 base search TTL and Tier-2 per-provider TTL.
func WithCacheTTLs(baseTTL, providerTTL time.Duration) AggregatorOption {
	return func(a *FlightAggregator) {
		if baseTTL > 0 {
			a.baseCacheTTL = baseTTL
		}
		if providerTTL > 0 {
			a.providerCacheTTL = providerTTL
		}
	}
}

// WithUpstreamTimeout overrides the bounded singleflight upstream timeout used to
// shield coalesced searches from any individual caller's short deadline.
func WithUpstreamTimeout(d time.Duration) AggregatorOption {
	return func(a *FlightAggregator) {
		if d > 0 {
			a.upstreamTimeout = d
		}
	}
}

const defaultUpstreamTimeout = 10 * time.Second

// FlightAggregator orchestrates concurrent scatter-gather queries across
// registered FlightProvider adapters with two-tier write-through caching.
type FlightAggregator struct {
	providers        []domain.FlightProvider
	cache            domain.FlightCache
	baseCacheTTL     time.Duration
	providerCacheTTL time.Duration
	upstreamTimeout  time.Duration
	sfGroup          singleflight.Group
	log              *slog.Logger
}

// NewFlightAggregator constructs an aggregator over the given FlightProvider ports.
func NewFlightAggregator(providers []domain.FlightProvider, log *slog.Logger, opts ...AggregatorOption) *FlightAggregator {
	agg := &FlightAggregator{
		providers:        providers,
		cache:            noopFlightCache{},
		baseCacheTTL:     defaultBaseCacheTTL,
		providerCacheTTL: defaultProviderCacheTTL,
		upstreamTimeout:  defaultUpstreamTimeout,
		log:              log,
	}
	for _, opt := range opts {
		opt(agg)
	}
	return agg
}

type providerOutcome struct {
	provider   string
	flights    []domain.Flight
	err        error
	durationMS int64
	cacheHit   bool
}

type baseSearchSnapshot struct {
	flights []domain.Flight
	meta    domain.SearchMetadata
}

// SearchFlights validates criteria and search options, checks L1 cache or coalesces
// live provider fan-out, then runs price comparison, ranking, filtering, and sorting.
func (a *FlightAggregator) SearchFlights(ctx context.Context, criteria domain.SearchCriteria, opts SearchOptions) (domain.SearchResult, error) {
	if err := criteria.Validate(); err != nil {
		return domain.SearchResult{}, err
	}
	if err := opts.Filters.Validate(); err != nil {
		return domain.SearchResult{}, err
	}
	sortBy := domain.SortBy(strings.TrimSpace(string(opts.SortBy)))
	if err := sortBy.Validate(); err != nil {
		return domain.SearchResult{}, err
	}
	criteria = criteria.Normalize()

	start := time.Now()
	traceID := logger.TraceIDFromContext(ctx)

	aggregated, meta, cacheHit, err := a.loadAggregatedFlights(ctx, traceID, criteria)
	if err != nil {
		meta.CacheHit = cacheHit
		meta.SearchTimeMS = time.Since(start).Milliseconds()
		return domain.SearchResult{
			SearchCriteria: criteria,
			Metadata:       meta,
			Flights:        aggregated,
		}, err
	}

	// Run ComparePrices and ScoreBestValue on the full aggregated cohort BEFORE FilterFlights.
	// Keeping ComparePrices pre-filter ensures price_comparison baselines reflect the true
	// route-wide market even when a user filters down to a single airline or narrow price band.
	// Keeping ScoreBestValue pre-filter ensures a lone or sparse surviving flight retains its
	// market-relative value score instead of inflating to 1.0 when competitors are filtered out.
	ComparePrices(aggregated)
	ScoreBestValue(aggregated)
	filtered := FilterFlights(aggregated, opts.Filters)
	SortFlights(filtered, sortBy)

	meta.CacheHit = cacheHit
	meta.TotalResults = len(filtered)
	meta.SearchTimeMS = time.Since(start).Milliseconds()

	return domain.SearchResult{
		SearchCriteria: criteria,
		Metadata:       meta,
		Flights:        filtered,
	}, nil
}

func (a *FlightAggregator) loadAggregatedFlights(
	ctx context.Context,
	traceID string,
	criteria domain.SearchCriteria,
) ([]domain.Flight, domain.SearchMetadata, bool, error) {
	baseKey := baseSearchCacheKey(criteria)
	if cached, found := a.getCachedBaseSearch(ctx, traceID, baseKey); found {
		return filterValidFlightsForCriteria(cached.Flights, criteria), cached.Metadata, true, nil
	}

	ch := a.sfGroup.DoChan(baseKey, func() (any, error) {
		// Detach the shared upstream fetch from the first caller's deadline/cancellation
		// using an independent bounded timeout so a fast-failing leader never aborts
		// healthy coalesced followers or prevents cache warming.
		sfCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.upstreamTimeout)
		defer cancel()

		outcomes := a.fetchProviderOutcomes(sfCtx, traceID, criteria)
		aggregated, meta := a.collectOutcomes(traceID, criteria, outcomes)

		if len(a.providers) > 0 && meta.ProvidersSucceeded == 0 {
			return baseSearchSnapshot{flights: aggregated, meta: meta}, domain.ErrAllProvidersFailed
		}

		// Skip L1 caching on degraded responses so transient provider failures are never cached.
		if meta.ProvidersFailed == 0 {
			a.setCachedBaseSearch(sfCtx, traceID, baseKey, aggregated, meta)
		}

		return baseSearchSnapshot{flights: aggregated, meta: meta}, nil
	})

	select {
	case <-ctx.Done():
		return nil, domain.SearchMetadata{}, false, ctx.Err()
	case res := <-ch:
		snap, _ := res.Val.(baseSearchSnapshot)
		if res.Err != nil {
			return cloneFlights(snap.flights), cloneMetadata(snap.meta), false, res.Err
		}
		return cloneFlights(snap.flights), cloneMetadata(snap.meta), false, nil
	}
}

func (a *FlightAggregator) fetchProviderOutcomes(ctx context.Context, traceID string, criteria domain.SearchCriteria) []providerOutcome {
	outcomes := make([]providerOutcome, len(a.providers))
	var wg sync.WaitGroup

	for i, p := range a.providers {
		wg.Add(1)
		go func(idx int, prov domain.FlightProvider) {
			defer wg.Done()
			pStart := time.Now()

			// Isolate cancellation per provider so one failure never aborts peers.
			pCtx := context.WithoutCancel(ctx)
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				pCtx, cancel = context.WithDeadline(pCtx, deadline)
				defer cancel()
			}

			if cachedFlights, found := a.getCachedProviderFlights(pCtx, traceID, prov.Name(), criteria); found {
				outcomes[idx] = providerOutcome{
					provider:   prov.Name(),
					flights:    cachedFlights,
					durationMS: time.Since(pStart).Milliseconds(),
					cacheHit:   true,
				}
				return
			}

			flights, err := prov.SearchFlights(pCtx, criteria)
			if err == nil {
				validFlights := filterValidFlightsForCriteria(flights, criteria)
				a.setCachedProviderFlights(pCtx, traceID, prov.Name(), criteria, validFlights)
				flights = validFlights
			}

			outcomes[idx] = providerOutcome{
				provider:   prov.Name(),
				flights:    flights,
				err:        err,
				durationMS: time.Since(pStart).Milliseconds(),
			}
		}(i, p)
	}

	wg.Wait()
	return outcomes
}

func (a *FlightAggregator) collectOutcomes(traceID string, criteria domain.SearchCriteria, outcomes []providerOutcome) ([]domain.Flight, domain.SearchMetadata) {
	aggregated := make([]domain.Flight, 0)
	providerErrors := make(map[string]string)
	succeeded := 0
	failed := 0

	for _, out := range outcomes {
		if out.err != nil {
			failed++
			providerErrors[out.provider] = out.err.Error()
			if a.log != nil {
				a.log.Warn("provider_query",
					slog.String("trace_id", traceID),
					slog.String("provider", out.provider),
					slog.Int64("duration_ms", out.durationMS),
					slog.Int("flights", 0),
					slog.String("status", "error"),
					slog.String("error", out.err.Error()),
				)
			}
			continue
		}

		succeeded++
		validFlights := filterValidFlightsForCriteria(out.flights, criteria)
		aggregated = append(aggregated, validFlights...)

		if a.log != nil {
			a.log.Info("provider_query",
				slog.String("trace_id", traceID),
				slog.String("provider", out.provider),
				slog.Int64("duration_ms", out.durationMS),
				slog.Int("flights", len(validFlights)),
				slog.Bool("cache_hit", out.cacheHit),
				slog.String("status", "ok"),
			)
		}
	}

	if len(providerErrors) == 0 {
		providerErrors = nil
	}

	meta := domain.SearchMetadata{
		TotalResults:       len(aggregated),
		ProvidersQueried:   len(a.providers),
		ProvidersSucceeded: succeeded,
		ProvidersFailed:    failed,
		ProviderErrors:     providerErrors,
	}
	return aggregated, meta
}

func (a *FlightAggregator) getCachedBaseSearch(ctx context.Context, traceID, key string) (cachedBaseSearch, bool) {
	raw, err := a.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, domain.ErrCacheMiss) && a.log != nil {
			a.log.Warn("cache_get_failed",
				slog.String("trace_id", traceID),
				slog.String("tier", "base"),
				slog.String("key", key),
				slog.String("error", err.Error()),
			)
		}
		return cachedBaseSearch{}, false
	}

	entry, err := unmarshalBaseSearch(raw)
	if err != nil {
		if a.log != nil {
			a.log.Warn("cache_decode_failed",
				slog.String("trace_id", traceID),
				slog.String("tier", "base"),
				slog.String("key", key),
				slog.String("error", err.Error()),
			)
		}
		return cachedBaseSearch{}, false
	}
	return entry, true
}

func (a *FlightAggregator) setCachedBaseSearch(ctx context.Context, traceID, key string, flights []domain.Flight, meta domain.SearchMetadata) {
	raw, err := marshalBaseSearch(flights, meta)
	if err != nil {
		return
	}
	if err := a.cache.Set(ctx, key, raw, a.baseCacheTTL); err != nil && a.log != nil {
		a.log.Warn("cache_set_failed",
			slog.String("trace_id", traceID),
			slog.String("tier", "base"),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
	}
}

func (a *FlightAggregator) getCachedProviderFlights(ctx context.Context, traceID, providerName string, criteria domain.SearchCriteria) ([]domain.Flight, bool) {
	key := providerSearchCacheKey(providerName, criteria)
	raw, err := a.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, domain.ErrCacheMiss) && a.log != nil {
			a.log.Warn("cache_get_failed",
				slog.String("trace_id", traceID),
				slog.String("tier", "provider"),
				slog.String("provider", providerName),
				slog.String("key", key),
				slog.String("error", err.Error()),
			)
		}
		return nil, false
	}

	var flights []domain.Flight
	if err := json.Unmarshal(raw, &flights); err != nil {
		return nil, false
	}
	return filterValidFlightsForCriteria(flights, criteria), true
}

func (a *FlightAggregator) setCachedProviderFlights(ctx context.Context, traceID, providerName string, criteria domain.SearchCriteria, flights []domain.Flight) {
	key := providerSearchCacheKey(providerName, criteria)
	raw, err := json.Marshal(flights)
	if err != nil {
		return
	}
	if err := a.cache.Set(ctx, key, raw, a.providerCacheTTL); err != nil && a.log != nil {
		a.log.Warn("cache_set_failed",
			slog.String("trace_id", traceID),
			slog.String("tier", "provider"),
			slog.String("provider", providerName),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
	}
}
