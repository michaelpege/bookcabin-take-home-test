package provider

import (
	"context"
	"errors"
	"net/http"
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
