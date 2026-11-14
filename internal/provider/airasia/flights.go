package airasia

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

// SearchFlights translates canonical criteria into AirAsia's wire contract (Stage A),
// queries the simulated AirAsia API (including 90% SLA simulation), and normalizes results.
func (c *Client) SearchFlights(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	headers := make(http.Header)
	if c.auth != nil {
		if err := c.auth.Apply(ctx, headers); err != nil {
			return nil, fmt.Errorf("%w: airasia auth: %w", domain.ErrProviderPermanent, err)
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
		return nil, fmt.Errorf("%w: airasia unauthorized: missing auth header", domain.ErrProviderPermanent)
	}

	if err := provider.SimulateDelay(ctx, c.minDelay, c.maxDelay); err != nil {
		return nil, fmt.Errorf("airasia request cancelled: %w", err)
	}

	if c.failureRate > 0 && rand.Float64() < c.failureRate {
		return nil, fmt.Errorf("%w: airasia simulated transient failure", domain.ErrProviderUnavailable)
	}

	var resp apiResponse
	if err := json.Unmarshal(c.payload, &resp); err != nil {
		return nil, fmt.Errorf("%w: airasia decode response: %w", domain.ErrProviderPermanent, err)
	}
	if !strings.EqualFold(resp.Status, "ok") {
		return nil, fmt.Errorf("%w: airasia contract error: unexpected status %q", domain.ErrProviderPermanent, resp.Status)
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
	if !strings.EqualFold(rf.FromAirport, req.Origin) {
		return false
	}
	if !strings.EqualFold(rf.ToAirport, req.Destination) {
		return false
	}
	if req.DepartureDate != "" && !timeutil.MatchesDepartureDate(rf.DepartTime, "", rf.FromAirport, req.DepartureDate) {
		return false
	}
	if req.CabinClass != "" && !strings.EqualFold(rf.CabinClass, req.CabinClass) {
		return false
	}
	if req.Passengers > 0 && rf.Seats < req.Passengers {
		return false
	}
	return true
}
