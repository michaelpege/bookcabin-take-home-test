package batik

import (
	"fmt"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/currency"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

var iataCabinClasses = map[string]string{
	"Y":               "economy",
	"W":               "premium_economy",
	"C":               "business",
	"J":               "business",
	"F":               "first",
	"ECONOMY":         "economy",
	"PREMIUM_ECONOMY": "premium_economy",
	"BUSINESS":        "business",
	"FIRST":           "first",
}

func (rf rawFlight) toDomain(providerName string) (domain.Flight, error) {
	depTime, err := timeutil.ParseTimestamp(rf.DepartureDateTime, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure time for %s: %w", rf.FlightNumber, err)
	}
	arrTime, err := timeutil.ParseTimestamp(rf.ArrivalDateTime, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival time for %s: %w", rf.FlightNumber, err)
	}

	// Derive authoritative elapsed minutes from UTC timestamps when connections exist
	// (e.g., ID7042's travelTime omits the +0700 -> +0800 timezone shift).
	totalMinutes, durErr := timeutil.ParseDurationMinutes(rf.TravelTime)
	if len(rf.Connections) > 0 || durErr != nil || totalMinutes <= 0 {
		totalMinutes = int(arrTime.Sub(depTime) / time.Minute)
	}

	amenities := make([]string, 0, len(rf.OnboardServices))
	for _, svc := range rf.OnboardServices {
		if trimmed := strings.ToLower(strings.TrimSpace(svc)); trimmed != "" {
			amenities = append(amenities, trimmed)
		}
	}

	var aircraft *string
	if trimmed := strings.TrimSpace(rf.AircraftModel); trimmed != "" {
		aircraft = &trimmed
	}

	stops := rf.NumberOfStops
	if stops == 0 && len(rf.Connections) > 0 {
		stops = len(rf.Connections)
	}

	carryOn, checked := parseBaggageInfo(rf.BaggageInfo)

	return domain.Flight{
		ID:       fmt.Sprintf("%s_%s", rf.FlightNumber, providerName),
		Provider: providerName,
		Airline: domain.Airline{
			Name: rf.AirlineName,
			Code: rf.AirlineIATA,
		},
		FlightNumber: rf.FlightNumber,
		Departure: domain.LocationPoint{
			Airport:   rf.Origin,
			City:      provider.ResolveCity(rf.Origin, ""),
			Datetime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.LocationPoint{
			Airport:   rf.Destination,
			City:      provider.ResolveCity(rf.Destination, ""),
			Datetime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration: domain.DurationInfo{
			TotalMinutes: totalMinutes,
			Formatted:    timeutil.FormatDuration(totalMinutes),
		},
		Stops: stops,
		Price: domain.PriceInfo{
			Amount:    rf.Fare.TotalPrice,
			Currency:  rf.Fare.CurrencyCode,
			Formatted: currency.FormatIDR(rf.Fare.TotalPrice),
		},
		AvailableSeats: rf.SeatsAvailable,
		CabinClass:     normalizeCabinClass(rf.Fare.Class),
		Aircraft:       aircraft,
		Amenities:      amenities,
		Baggage: domain.BaggageInfo{
			CarryOn: carryOn,
			Checked: checked,
		},
	}, nil
}

func normalizeCabinClass(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if mapped, ok := iataCabinClasses[code]; ok {
		return mapped
	}
	if code == "" {
		return "economy"
	}
	return strings.ToLower(code)
}

func parseBaggageInfo(raw string) (carryOn, checked string) {
	carryOn, checked = "None", "None"
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		lower := strings.ToLower(p)
		switch {
		case strings.Contains(lower, "cabin"):
			carryOn = strings.TrimSpace(strings.ReplaceAll(lower, "cabin", ""))
		case strings.Contains(lower, "checked"):
			checked = strings.TrimSpace(strings.ReplaceAll(lower, "checked", ""))
		}
	}
	if carryOn == "" {
		carryOn = "None"
	}
	if checked == "" {
		checked = "None"
	}
	return carryOn, checked
}
