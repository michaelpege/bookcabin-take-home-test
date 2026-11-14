package provider

import (
	"context"
	"encoding/json"
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
	policy  ResiliencePolicy
	limiter *rate.Limiter
}

// NewResilientProvider wraps a FlightProvider with the supplied ResiliencePolicy.
func NewResilientProvider(next domain.FlightProvider, policy ResiliencePolicy) *ResilientProvider {
	var limiter *rate.Limiter
	if policy.RateLimitRPS > 0 {
		burst := policy.RateBurst
		if burst < 1 {
			burst = 1
		}
		limiter = rate.NewLimiter(rate.Limit(policy.RateLimitRPS), burst)
	}
	return &ResilientProvider{
		next:    next,
		policy:  policy,
		limiter: limiter,
	}
}

// Name delegates to the wrapped provider's display name.
func (r *ResilientProvider) Name() string {
	return r.next.Name()
}

// SearchFlights executes the wrapped provider with per-attempt rate limiting,
// retries on transient errors, and timeouts.
func (r *ResilientProvider) SearchFlights(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	attempts := r.policy.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, r.wrapUnavailable("cancelled", err)
		}

		// Acquire a token before each physical attempt so retries cannot amplify upstream RPS.
		if r.limiter != nil {
			if err := r.limiter.Wait(ctx); err != nil {
				return nil, r.wrapUnavailable("rate limit wait", err)
			}
		}

		flights, err := r.runAttempt(ctx, criteria)
		if err == nil {
			return flights, nil
		}
		lastErr = err

		// Do not retry if the caller's parent context was cancelled or expired.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, r.wrapUnavailable("cancelled", ctxErr)
		}
		// Fail fast on permanent contract, decode, or auth errors without burning retries.
		if isPermanentProviderError(err) {
			return nil, r.wrapUnavailable("permanent error", err)
		}
		if attempt == attempts-1 {
			break
		}

		backoff := r.computeBackoff(attempt)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, r.wrapUnavailable("cancelled during backoff", ctx.Err())
		case <-timer.C:
		}
	}

	return nil, r.wrapUnavailable(fmt.Sprintf("failed after %d attempt(s)", attempts), lastErr)
}

func (r *ResilientProvider) wrapUnavailable(reason string, err error) error {
	if errors.Is(err, domain.ErrProviderUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %s %s: %w", domain.ErrProviderUnavailable, r.next.Name(), reason, err)
}

func isPermanentProviderError(err error) bool {
	if errors.Is(err, domain.ErrProviderPermanent) {
		return true
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr)
}

func (r *ResilientProvider) runAttempt(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	if r.policy.Timeout <= 0 {
		return r.next.SearchFlights(ctx, criteria)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, r.policy.Timeout)
	defer cancel()
	return r.next.SearchFlights(attemptCtx, criteria)
}

func (r *ResilientProvider) computeBackoff(attempt int) time.Duration {
	base := r.policy.BaseBackoff
	if base <= 0 {
		base = 20 * time.Millisecond
	}
	delay := base << attempt
	if r.policy.MaxBackoff > 0 && delay > r.policy.MaxBackoff {
		delay = r.policy.MaxBackoff
	}
	// Full jitter in [0.5*delay, 1.0*delay] to prevent synchronized retry storms.
	half := delay / 2
	if half <= 0 {
		return delay
	}
	return half + time.Duration(rand.Int64N(int64(half)+1))
}
