package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
)

// SearchOptions carries optional post-aggregation filter and sort settings.
type SearchOptions struct {
	Filters domain.FilterOptions
	SortBy  domain.SortBy
}

// FlightAggregator orchestrates concurrent scatter-gather queries across
// registered FlightProvider adapters and collects partial-failure metadata.
type FlightAggregator struct {
	providers []domain.FlightProvider
	log       *slog.Logger
}

// NewFlightAggregator constructs an aggregator over the given FlightProvider ports.
func NewFlightAggregator(providers []domain.FlightProvider, log *slog.Logger) *FlightAggregator {
	return &FlightAggregator{
		providers: providers,
		log:       log,
	}
}

type providerOutcome struct {
	provider   string
	flights    []domain.Flight
	err        error
	durationMS int64
}

// SearchFlights validates criteria, fans out concurrently to all providers using
// isolated contexts, logs per-provider latency, and returns unified results.
func (a *FlightAggregator) SearchFlights(ctx context.Context, criteria domain.SearchCriteria, _ SearchOptions) (domain.SearchResult, error) {
	if err := criteria.Validate(); err != nil {
		return domain.SearchResult{}, err
	}

	start := time.Now()
	traceID := logger.TraceIDFromContext(ctx)

	outcomes := make(chan providerOutcome, len(a.providers))
	var wg sync.WaitGroup

	for _, p := range a.providers {
		wg.Add(1)
		go func(prov domain.FlightProvider) {
			defer wg.Done()
			pStart := time.Now()

			// Isolate cancellation per provider so one failure never aborts peers.
			pCtx := context.WithoutCancel(ctx)
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				pCtx, cancel = context.WithDeadline(pCtx, deadline)
				defer cancel()
			}

			flights, err := prov.SearchFlights(pCtx, criteria)
			outcomes <- providerOutcome{
				provider:   prov.Name(),
				flights:    flights,
				err:        err,
				durationMS: time.Since(pStart).Milliseconds(),
			}
		}(p)
	}

	wg.Wait()
	close(outcomes)

	aggregated := make([]domain.Flight, 0)
	providerErrors := make(map[string]string)
	succeeded := 0
	failed := 0

	for out := range outcomes {
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
		validCount := 0
		for _, f := range out.flights {
			if err := f.Validate(); err != nil {
				continue
			}
			if f.Amenities == nil {
				f.Amenities = []string{}
			}
			aggregated = append(aggregated, f)
			validCount++
		}

		if a.log != nil {
			a.log.Info("provider_query",
				slog.String("trace_id", traceID),
				slog.String("provider", out.provider),
				slog.Int64("duration_ms", out.durationMS),
				slog.Int("flights", validCount),
				slog.String("status", "ok"),
			)
		}
	}

	if len(providerErrors) == 0 {
		providerErrors = nil
	}

	return domain.SearchResult{
		SearchCriteria: criteria,
		Metadata: domain.SearchMetadata{
			TotalResults:       len(aggregated),
			ProvidersQueried:   len(a.providers),
			ProvidersSucceeded: succeeded,
			ProvidersFailed:    failed,
			SearchTimeMS:       time.Since(start).Milliseconds(),
			CacheHit:           false,
			ProviderErrors:     providerErrors,
		},
		Flights: aggregated,
	}, nil
}
