package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

type stubProvider struct {
	name    string
	handler func(ctx context.Context, attempt int32) ([]domain.Flight, error)
	calls   atomic.Int32
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) SearchFlights(ctx context.Context, _ domain.SearchCriteria) ([]domain.Flight, error) {
	n := s.calls.Add(1)
	return s.handler(ctx, n)
}

func sampleCriteria() domain.SearchCriteria {
	return domain.SearchCriteria{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func TestResilientProviderRetriesAndSucceeds(t *testing.T) {
	stub := &stubProvider{
		name: "AirAsia",
		handler: func(_ context.Context, attempt int32) ([]domain.Flight, error) {
			if attempt < 2 {
				return nil, errors.New("simulated 503")
			}
			return []domain.Flight{{ID: "QZ520_AirAsia"}}, nil
		},
	}

	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:     200 * time.Millisecond,
		MaxRetries:  2,
		BaseBackoff: 5 * time.Millisecond,
		MaxBackoff:  15 * time.Millisecond,
	})

	flights, err := p.SearchFlights(context.Background(), sampleCriteria())
	if err != nil {
		t.Fatalf("expected retry recovery, got error: %v", err)
	}
	if len(flights) != 1 || flights[0].ID != "QZ520_AirAsia" {
		t.Fatalf("unexpected flights: %+v", flights)
	}
	if got := stub.calls.Load(); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestResilientProviderExhaustsRetries(t *testing.T) {
	stub := &stubProvider{
		name: "AirAsia",
		handler: func(_ context.Context, _ int32) ([]domain.Flight, error) {
			return nil, errors.New("upstream down")
		},
	}

	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:     100 * time.Millisecond,
		MaxRetries:  2,
		BaseBackoff: 2 * time.Millisecond,
	})

	_, err := p.SearchFlights(context.Background(), sampleCriteria())
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if got := stub.calls.Load(); got != 3 {
		t.Fatalf("expected 3 total attempts (1 + 2 retries), got %d", got)
	}
}

func TestResilientProviderEnforcesAttemptTimeout(t *testing.T) {
	stub := &stubProvider{
		name: "Batik Air",
		handler: func(ctx context.Context, _ int32) ([]domain.Flight, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return []domain.Flight{}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:    20 * time.Millisecond,
		MaxRetries: 0,
	})

	_, err := p.SearchFlights(context.Background(), sampleCriteria())
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable on timeout, got %v", err)
	}
}

func TestResilientProviderRespectsRateLimitAndParentCancel(t *testing.T) {
	stub := &stubProvider{
		name: "Garuda Indonesia",
		handler: func(_ context.Context, _ int32) ([]domain.Flight, error) {
			return []domain.Flight{}, nil
		},
	}

	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:      100 * time.Millisecond,
		RateLimitRPS: 1,
		RateBurst:    1,
	})

	if _, err := p.SearchFlights(context.Background(), sampleCriteria()); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	if _, err := p.SearchFlights(ctx, sampleCriteria()); err == nil {
		t.Fatal("expected rate-limit context deadline error, got nil")
	}
}

func TestAuthenticators(t *testing.T) {
	ctx := context.Background()

	h1 := make(http.Header)
	if err := (BearerTokenAuth{Token: "mock-jwt"}).Apply(ctx, h1); err != nil {
		t.Fatalf("bearer apply failed: %v", err)
	}
	if got := h1.Get("Authorization"); got != "Bearer mock-jwt" {
		t.Fatalf("unexpected bearer header: %q", got)
	}

	h2 := make(http.Header)
	if err := (APIKeyAuth{HeaderName: "X-API-Key", Key: "secret"}).Apply(ctx, h2); err != nil {
		t.Fatalf("api key apply failed: %v", err)
	}
	if got := h2.Get("X-API-Key"); got != "secret" {
		t.Fatalf("unexpected api key header: %q", got)
	}

	h3 := make(http.Header)
	if err := (BasicAuth{Username: "batik", Password: "pw"}).Apply(ctx, h3); err != nil {
		t.Fatalf("basic auth apply failed: %v", err)
	}
	if got := h3.Get("Authorization"); got != "Basic YmF0aWs6cHc=" {
		t.Fatalf("unexpected basic auth header: %q", got)
	}
}

func TestResilientProviderConsumesRateLimitPerRetryAttempt(t *testing.T) {
	stub := &stubProvider{
		name: "AirAsia",
		handler: func(_ context.Context, attempt int32) ([]domain.Flight, error) {
			if attempt < 3 {
				return nil, errors.New("transient 503")
			}
			return []domain.Flight{{ID: "QZ520_AirAsia"}}, nil
		},
	}

	// RateBurst=1 allows attempt 1 immediately; retries 2 and 3 each require a 50ms token refill.
	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:      100 * time.Millisecond,
		MaxRetries:   2,
		BaseBackoff:  1 * time.Millisecond,
		RateLimitRPS: 20, // 50ms per token
		RateBurst:    1,
	})

	start := time.Now()
	flights, err := p.SearchFlights(context.Background(), sampleCriteria())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("expected eventual retry success, got %v", err)
	}
	if len(flights) != 1 || stub.calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got calls=%d flights=%d", stub.calls.Load(), len(flights))
	}
	if elapsed < 80*time.Millisecond {
		t.Fatalf("expected per-attempt rate limiting across retries (>=80ms for 2 refills), finished in %v", elapsed)
	}

	// Drain burst and verify retry blocked by rate limit wraps both ErrProviderUnavailable & context deadline.
	stubFail := &stubProvider{
		name: "AirAsia",
		handler: func(_ context.Context, _ int32) ([]domain.Flight, error) {
			return nil, errors.New("transient failure")
		},
	}
	pStrict := NewResilientProvider(stubFail, ResiliencePolicy{
		Timeout:      50 * time.Millisecond,
		MaxRetries:   2,
		BaseBackoff:  1 * time.Millisecond,
		RateLimitRPS: 1, // 1s refill
		RateBurst:    1,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err = pStrict.SearchFlights(ctx, sampleCriteria())
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable when retry rate limit times out, got %v", err)
	}
	if !strings.Contains(err.Error(), "context deadline") {
		t.Fatalf("expected rate limiter context deadline message, got %v", err)
	}
	if got := stubFail.calls.Load(); got != 1 {
		t.Fatalf("expected attempt 2 blocked at limiter before provider call, got %d calls", got)
	}
}

func TestResilientProviderWrapsUnavailableOnCancellation(t *testing.T) {
	stub := &stubProvider{
		name: "Garuda Indonesia",
		handler: func(_ context.Context, _ int32) ([]domain.Flight, error) {
			return nil, errors.New("temporary failure")
		},
	}

	p := NewResilientProvider(stub, ResiliencePolicy{
		Timeout:     50 * time.Millisecond,
		MaxRetries:  2,
		BaseBackoff: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := p.SearchFlights(ctx, sampleCriteria())
	if !errors.Is(err, domain.ErrProviderUnavailable) || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected both ErrProviderUnavailable and context.Canceled, got %v", err)
	}
}

func TestResilientProviderFailsFastOnPermanentErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "domain permanent sentinel",
			err:  fmt.Errorf("%w: bad credentials", domain.ErrProviderPermanent),
		},
		{
			name: "json syntax decode error",
			err:  json.Unmarshal([]byte("{bad"), new(map[string]any)),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubProvider{
				name: "Batik Air",
				handler: func(_ context.Context, _ int32) ([]domain.Flight, error) {
					return nil, tc.err
				},
			}
			p := NewResilientProvider(stub, ResiliencePolicy{
				Timeout:     100 * time.Millisecond,
				MaxRetries:  3,
				BaseBackoff: 5 * time.Millisecond,
			})

			_, err := p.SearchFlights(context.Background(), sampleCriteria())
			if !errors.Is(err, domain.ErrProviderUnavailable) {
				t.Fatalf("expected ErrProviderUnavailable wrapper, got %v", err)
			}
			if got := stub.calls.Load(); got != 1 {
				t.Fatalf("expected fail-fast with 1 attempt (no retries), got %d", got)
			}
		})
	}
}
