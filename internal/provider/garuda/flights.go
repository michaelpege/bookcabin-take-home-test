package garuda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

type apiResponse struct {
	Status  string      `json:"status"`
	Flights []rawFlight `json:"flights"`
}

type rawFlight struct {
	FlightID        string       `json:"flight_id"`
	Airline         string       `json:"airline"`
	AirlineCode     string       `json:"airline_code"`
	Departure       rawEndpoint  `json:"departure"`
	Arrival         rawEndpoint  `json:"arrival"`
	DurationMinutes int          `json:"duration_minutes"`
	Stops           int          `json:"stops"`
	Aircraft        string       `json:"aircraft"`
	Price           rawPrice     `json:"price"`
	Segments        []rawSegment `json:"segments"`
	AvailableSeats  int          `json:"available_seats"`
	FareClass       string       `json:"fare_class"`
	Baggage         rawBaggage   `json:"baggage"`
	Amenities       []string     `json:"amenities"`
}

type rawEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type rawSegment struct {
	FlightNumber    string      `json:"flight_number"`
	Departure       rawEndpoint `json:"departure"`
	Arrival         rawEndpoint `json:"arrival"`
	DurationMinutes int         `json:"duration_minutes"`
	LayoverMinutes  int         `json:"layover_minutes"`
}

type rawPrice struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type rawBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

// SearchFlights simulates querying the Garuda Indonesia API, validating its wire
// contract (Level 1), and normalizing matching flights into canonical domain entities.
func (c *Client) SearchFlights(ctx context.Context, criteria domain.SearchCriteria) ([]domain.Flight, error) {
	headers := make(http.Header)
	if c.auth != nil {
		if err := c.auth.Apply(ctx, headers); err != nil {
			return nil, fmt.Errorf("garuda auth: %w", err)
		}
	}

	// Stage A wire translation: map canonical SearchCriteria to Garuda's supported params.
	_ = buildWireParams(criteria)

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

	results := make([]domain.Flight, 0, len(resp.Flights))
	for _, rf := range resp.Flights {
		flight, err := normalizeFlight(rf)
		if err != nil {
			continue
		}
		if err := flight.Validate(); err != nil {
			continue
		}
		if matchesCriteria(flight, criteria) {
			results = append(results, flight)
		}
	}
	return results, nil
}

func buildWireParams(criteria domain.SearchCriteria) map[string]string {
	return map[string]string{
		"origin":        strings.ToUpper(strings.TrimSpace(criteria.Origin)),
		"destination":   strings.ToUpper(strings.TrimSpace(criteria.Destination)),
		"departureDate": strings.TrimSpace(criteria.DepartureDate),
		"cabinClass":    strings.ToLower(strings.TrimSpace(criteria.CabinClass)),
	}
}

func matchesCriteria(f domain.Flight, c domain.SearchCriteria) bool {
	if !strings.EqualFold(f.Departure.Airport, c.Origin) {
		return false
	}
	if !strings.EqualFold(f.Arrival.Airport, c.Destination) {
		return false
	}
	if c.DepartureDate != "" && !strings.HasPrefix(f.Departure.Datetime, c.DepartureDate) {
		return false
	}
	if c.CabinClass != "" && !strings.EqualFold(f.CabinClass, c.CabinClass) {
		return false
	}
	if c.Passengers > 0 && f.AvailableSeats < c.Passengers {
		return false
	}
	return true
}
