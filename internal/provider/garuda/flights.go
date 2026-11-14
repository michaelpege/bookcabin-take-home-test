package garuda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// SearchFlights translates canonical criteria into Garuda's wire contract (Stage A),
// queries the simulated Garuda API, and normalizes + validates the returned flights.
func (c *Client) SearchFlights(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	headers := make(http.Header)
	if c.auth != nil {
		if err := c.auth.Apply(ctx, headers); err != nil {
			return nil, fmt.Errorf("garuda auth: %w", err)
		}
	}

	wireReq := buildWireParams(criteria)

	rawFlights, err := c.fetchFromMockAPI(ctx, headers, wireReq)
	if err != nil {
		return nil, err
	}

	results := make([]domain.Flight, 0, len(rawFlights))
	for _, rf := range rawFlights {
		flight, err := normalizeFlight(rf)
		if err != nil {
			continue
		}
		if err := flight.Validate(); err != nil {
			continue
		}
		results = append(results, flight)
	}
	return results, nil
}

// buildWireParams performs Stage-A translation from canonical SearchCriteria
// into Garuda's native request format.
func buildWireParams(criteria domain.SearchCriteria) wireSearchRequest {
	return wireSearchRequest{
		Origin:        strings.ToUpper(strings.TrimSpace(criteria.Origin)),
		Destination:   strings.ToUpper(strings.TrimSpace(criteria.Destination)),
		DepartureDate: strings.TrimSpace(criteria.DepartureDate),
		CabinClass:    strings.ToLower(strings.TrimSpace(criteria.CabinClass)),
		Passengers:    criteria.Passengers,
	}
}

// fetchFromMockAPI simulates Garuda's remote endpoint: checks auth headers,
// simulates 50-100ms latency, enforces Level-1 contract status, and applies
// server-side query matching using wireReq.
func (c *Client) fetchFromMockAPI(ctx context.Context, headers http.Header, req wireSearchRequest) ([]rawFlight, error) {
	if c.auth != nil && headers.Get("Authorization") == "" {
		return nil, fmt.Errorf("garuda unauthorized: missing authorization header")
	}

	if err := c.simulateDelay(ctx); err != nil {
		return nil, fmt.Errorf("garuda request cancelled: %w", err)
	}

	var resp apiResponse
	if err := json.Unmarshal(c.payload, &resp); err != nil {
		return nil, fmt.Errorf("garuda decode response: %w", err)
	}
	if !strings.EqualFold(resp.Status, "success") {
		return nil, fmt.Errorf("garuda contract error: unexpected status %q", resp.Status)
	}

	matched := make([]rawFlight, 0, len(resp.Flights))
	for _, rf := range resp.Flights {
		if matchesWireRequest(rf, req) {
			matched = append(matched, rf)
		}
	}
	return matched, nil
}

func matchesWireRequest(rf rawFlight, req wireSearchRequest) bool {
	dep, arr, _ := resolveRouteEndpoints(rf)
	if !strings.EqualFold(dep.Airport, req.Origin) {
		return false
	}
	if !strings.EqualFold(arr.Airport, req.Destination) {
		return false
	}
	if req.DepartureDate != "" && !strings.HasPrefix(dep.Time, req.DepartureDate) {
		return false
	}
	if req.CabinClass != "" && !strings.EqualFold(rf.FareClass, req.CabinClass) {
		return false
	}
	if req.Passengers > 0 && rf.AvailableSeats < req.Passengers {
		return false
	}
	return true
}
