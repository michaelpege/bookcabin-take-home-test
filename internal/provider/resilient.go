package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"golang.org/x/time/rate"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// ResilientProvider is a Go struct-wrapping Decorator around domain.FlightProvider
// that enforces token-bucket rate limiting, per-attempt timeouts, and exponential
// backoff retries with jitter.
type ResilientProvider struct {
	next    domain.FlightProvider
	sla     ProviderSLA
	limiter *rate.Limiter
}

// NewResilientProvider wraps a FlightProvider with the supplied SLA policy.
func NewResilientProvider(next domain.FlightProvider, sla ProviderSLA) *ResilientProvider {
	var limiter *rate.Limiter
	if sla.RateLimitRPS > 0 {
		burst := sla.RateBurst
		if burst < 1 {
			burst = 1
		}
		limiter = rate.NewLimiter(rate.Limit(sla.RateLimitRPS), burst)
	}
	return &ResilientProvider{
		next:    next,
		sla:     sla,
		limiter: limiter,
	}
}

// Name delegates to the wrapped provider's display name.
func (r *ResilientProvider) Name() string {
	return r.next.Name()
}

// Search executes the wrapped provider with rate limiting, retries, and timeouts.
func (r *ResilientProvider) Search(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("%s rate limit wait: %w", r.next.Name(), err)
		}
	}

	attempts := r.sla.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		flights, err := r.runAttempt(ctx, criteria)
		if err == nil {
			return flights, nil
		}
		lastErr = err

		// Do not retry if the caller's parent context was cancelled or expired.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt == attempts-1 {
			break
		}

		backoff := r.computeBackoff(attempt)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	if errors.Is(lastErr, domain.ErrProviderUnavailable) {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%w: %s: %v", domain.ErrProviderUnavailable, r.next.Name(), lastErr)
}

func (r *ResilientProvider) runAttempt(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	if r.sla.Timeout <= 0 {
		return r.next.Search(ctx, criteria)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, r.sla.Timeout)
	defer cancel()
	return r.next.Search(attemptCtx, criteria)
}

func (r *ResilientProvider) computeBackoff(attempt int) time.Duration {
	base := r.sla.BaseBackoff
	if base <= 0 {
		base = 20 * time.Millisecond
	}
	delay := base << attempt
	if r.sla.MaxBackoff > 0 && delay > r.sla.MaxBackoff {
		delay = r.sla.MaxBackoff
	}
	// Full jitter in [0.5*delay, 1.0*delay] to prevent synchronized retry storms.
	half := delay / 2
	if half <= 0 {
		return delay
	}
	return half + time.Duration(rand.Int64N(int64(half)+1))
}
