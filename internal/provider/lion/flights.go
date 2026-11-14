package lion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

// SearchFlights translates canonical criteria into Lion Air's wire contract (Stage A),
// queries the simulated Lion Air API, and normalizes + validates returned flights.
func (c *Client) SearchFlights(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	headers := make(http.Header)
	if c.auth != nil {
		if err := c.auth.Apply(ctx, headers); err != nil {
			return nil, fmt.Errorf("lion auth: %w", err)
		}
	}

	wireReq := buildWireParams(criteria)

	rawFlights, err := c.fetchFromMockAPI(ctx, headers, wireReq)
	if err != nil {
		return nil, err
	}

	results := make([]domain.Flight, 0, len(rawFlights))
	for _, rf := range rawFlights {
		flight, err := rf.toDomain(c.name)
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

func buildWireParams(criteria domain.SearchCriteria) wireSearchRequest {
	return wireSearchRequest{
		Origin:        strings.ToUpper(strings.TrimSpace(criteria.Origin)),
		Destination:   strings.ToUpper(strings.TrimSpace(criteria.Destination)),
		DepartureDate: strings.TrimSpace(criteria.DepartureDate),
		CabinClass:    strings.ToLower(strings.TrimSpace(criteria.CabinClass)),
		Passengers:    criteria.Passengers,
	}
}

func (c *Client) fetchFromMockAPI(ctx context.Context, headers http.Header, req wireSearchRequest) ([]rawFlight, error) {
	if c.auth != nil && len(headers) == 0 {
		return nil, fmt.Errorf("lion unauthorized: missing auth header")
	}

	if err := provider.SimulateDelay(ctx, c.minDelay, c.maxDelay); err != nil {
		return nil, fmt.Errorf("lion request cancelled: %w", err)
	}

	var resp apiResponse
	if err := json.Unmarshal(c.payload, &resp); err != nil {
		return nil, fmt.Errorf("lion decode response: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("lion contract error: unsuccessful response")
	}

	matched := make([]rawFlight, 0, len(resp.Data.AvailableFlights))
	for _, rf := range resp.Data.AvailableFlights {
		if matchesWireRequest(rf, req) {
			matched = append(matched, rf)
		}
	}
	return matched, nil
}

func matchesWireRequest(rf rawFlight, req wireSearchRequest) bool {
	if !strings.EqualFold(rf.Route.From.Code, req.Origin) {
		return false
	}
	if !strings.EqualFold(rf.Route.To.Code, req.Destination) {
		return false
	}
	if req.DepartureDate != "" && !strings.HasPrefix(rf.Schedule.Departure, req.DepartureDate) {
		return false
	}
	if req.CabinClass != "" && !strings.EqualFold(rf.Pricing.FareType, req.CabinClass) {
		return false
	}
	if req.Passengers > 0 && rf.SeatsLeft < req.Passengers {
		return false
	}
	return true
}
