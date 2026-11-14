package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider/garuda"
	"github.com/michaelpege/bookcabin-take-home-test/internal/usecase"
)

func TestSearchFlightsEndpointE2EWithGaruda(t *testing.T) {
	log := newTestLogger()
	agg := usecase.NewFlightAggregator(
		[]domain.FlightProvider{garuda.NewClient(garuda.WithDelay(0, 0))},
		log,
	)
	router := NewRouter(log, agg)

	body := `{
		"origin": "CGK",
		"destination": "DPS",
		"departureDate": "2025-12-15",
		"returnDate": null,
		"passengers": 1,
		"cabinClass": "economy"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flights/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id response header")
	}

	var res domain.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode SearchResult: %v", err)
	}
	if res.Metadata.ProvidersQueried != 1 || res.Metadata.ProvidersSucceeded != 1 {
		t.Fatalf("unexpected provider metadata: %+v", res.Metadata)
	}
	if res.Metadata.TotalResults != 3 || len(res.Flights) != 3 {
		t.Fatalf("expected 3 Garuda flights, got %d", len(res.Flights))
	}
}

func TestSearchFlightsEndpointValidation(t *testing.T) {
	log := newTestLogger()
	agg := usecase.NewFlightAggregator(nil, log)
	router := NewRouter(log, agg)

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed json", body: `{`},
		{name: "invalid criteria", body: `{"origin":"CGK","destination":"CGK","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/flights/search", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

type failingSearchService struct{}

func (failingSearchService) SearchFlights(context.Context, domain.SearchCriteria, usecase.SearchOptions) (domain.SearchResult, error) {
	return domain.SearchResult{
		Metadata: domain.SearchMetadata{
			ProvidersQueried:   2,
			ProvidersSucceeded: 0,
			ProvidersFailed:    2,
			ProviderErrors: map[string]string{
				"Garuda Indonesia": domain.ErrProviderUnavailable.Error(),
				"AirAsia":          domain.ErrProviderUnavailable.Error(),
			},
		},
	}, domain.ErrAllProvidersFailed
}

func TestSearchFlightsEndpointAllProvidersFailReturns503(t *testing.T) {
	router := NewRouter(newTestLogger(), failingSearchService{})
	body := `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flights/search", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Error    string                `json:"error"`
		Metadata domain.SearchMetadata `json:"metadata"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode 503 payload: %v", err)
	}
	if resp.Error == "" || resp.Metadata.ProvidersFailed != 2 || resp.Metadata.ProvidersSucceeded != 0 {
		t.Fatalf("unexpected 503 payload: %+v", resp)
	}
}

func TestSearchFlightsEndpointFiltersAndBestValueSort(t *testing.T) {
	log := newTestLogger()
	agg := usecase.NewFlightAggregator(
		[]domain.FlightProvider{garuda.NewClient(garuda.WithDelay(0, 0))},
		log,
	)
	router := NewRouter(log, agg)

	body := `{
		"origin": "CGK",
		"destination": "DPS",
		"departureDate": "2025-12-15",
		"passengers": 1,
		"cabinClass": "economy",
		"sortBy": "best_value",
		"filters": {
			"max_stops": 0,
			"max_price": 1500000
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flights/search", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res domain.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode SearchResult: %v", err)
	}
	if res.Metadata.TotalResults != 2 || len(res.Flights) != 2 {
		t.Fatalf("expected 2 filtered non-stop Garuda flights, got %d", len(res.Flights))
	}
	for _, f := range res.Flights {
		if f.Stops != 0 {
			t.Fatalf("expected non-stop flight only, got stops=%d", f.Stops)
		}
		if f.BestValueScore == nil {
			t.Fatalf("expected BestValueScore populated on %s", f.FlightNumber)
		}
	}
}

func TestSearchFlightsEndpointRejectsInvalidSortAndFilters(t *testing.T) {
	log := newTestLogger()
	agg := usecase.NewFlightAggregator(nil, log)
	router := NewRouter(log, agg)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid sortBy",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","sortBy":"invalid_mode"}`,
		},
		{
			name: "malformed departure_window",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","filters":{"departure_window":{"start":"99:99","end":"bad"}}}`,
		},
		{
			name: "inverted min_price > max_price",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","filters":{"min_price":1000000,"max_price":500000}}`,
		},
		{
			name: "returnDate before departureDate",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","returnDate":"2025-12-10","passengers":1,"cabinClass":"economy"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/flights/search", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}
