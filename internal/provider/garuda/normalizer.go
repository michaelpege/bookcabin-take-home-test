package garuda

import (
	"fmt"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/currency"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
)

var airportCities = map[string]string{
	"CGK": "Jakarta",
	"DPS": "Denpasar",
	"SUB": "Surabaya",
	"SOC": "Solo",
	"UPG": "Makassar",
	"JOG": "Yogyakarta",
	"YIA": "Yogyakarta",
}

func (rf rawFlight) toDomain(providerName string) (domain.Flight, error) {
	depRaw, arrRaw, stops := resolveRouteEndpoints(rf)

	depTime, err := timeutil.ParseTimestamp(depRaw.Time, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure time for %s: %w", rf.FlightID, err)
	}
	arrTime, err := timeutil.ParseTimestamp(arrRaw.Time, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival time for %s: %w", rf.FlightID, err)
	}

	// Derive authoritative elapsed duration from UTC timestamps when connecting segments exist,
	// avoiding Garuda's truncated top-level duration_minutes (e.g., GA315).
	totalMinutes := rf.DurationMinutes
	if len(rf.Segments) > 1 || totalMinutes <= 0 {
		totalMinutes = int(arrTime.Sub(depTime) / time.Minute)
	}

	amenities := rf.Amenities
	if amenities == nil {
		amenities = []string{}
	}

	var aircraft *string
	if trimmed := strings.TrimSpace(rf.Aircraft); trimmed != "" {
		aircraft = &trimmed
	}

	cabinClass := strings.ToLower(strings.TrimSpace(rf.FareClass))
	if cabinClass == "" {
		cabinClass = "economy"
	}

	return domain.Flight{
		ID:       fmt.Sprintf("%s_%s", rf.FlightID, providerName),
		Provider: providerName,
		Airline: domain.Airline{
			Name: rf.Airline,
			Code: rf.AirlineCode,
		},
		FlightNumber: rf.FlightID,
		Departure: domain.LocationPoint{
			Airport:   depRaw.Airport,
			City:      resolveCity(depRaw),
			Datetime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.LocationPoint{
			Airport:   arrRaw.Airport,
			City:      resolveCity(arrRaw),
			Datetime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration: domain.DurationInfo{
			TotalMinutes: totalMinutes,
			Formatted:    timeutil.FormatDuration(totalMinutes),
		},
		Stops: stops,
		Price: domain.PriceInfo{
			Amount:    rf.Price.Amount,
			Currency:  rf.Price.Currency,
			Formatted: currency.FormatIDR(rf.Price.Amount),
		},
		AvailableSeats: rf.AvailableSeats,
		CabinClass:     cabinClass,
		Aircraft:       aircraft,
		Amenities:      amenities,
		Baggage: domain.BaggageInfo{
			CarryOn: formatPieces(rf.Baggage.CarryOn),
			Checked: formatPieces(rf.Baggage.Checked),
		},
	}, nil
}

func resolveRouteEndpoints(rf rawFlight) (dep, arr rawEndpoint, stops int) {
	if len(rf.Segments) == 0 {
		return rf.Departure, rf.Arrival, rf.Stops
	}
	first := rf.Segments[0].Departure
	last := rf.Segments[len(rf.Segments)-1].Arrival
	if first.City == "" {
		first.City = rf.Departure.City
	}
	return first, last, len(rf.Segments) - 1
}

func resolveCity(ep rawEndpoint) string {
	if strings.TrimSpace(ep.City) != "" {
		return ep.City
	}
	if city, ok := airportCities[strings.ToUpper(strings.TrimSpace(ep.Airport))]; ok {
		return city
	}
	return ep.Airport
}

func formatPieces(count int) string {
	if count <= 0 {
		return "None"
	}
	if count == 1 {
		return "1 piece"
	}
	return fmt.Sprintf("%d pieces", count)
}
