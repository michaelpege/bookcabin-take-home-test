package airasia

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/currency"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

func (rf rawFlight) toDomain(providerName string) (domain.Flight, error) {
	depTime, err := timeutil.ParseTimestamp(rf.DepartTime, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure time for %s: %w", rf.FlightCode, err)
	}
	arrTime, err := timeutil.ParseTimestamp(rf.ArriveTime, "")
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival time for %s: %w", rf.FlightCode, err)
	}

	totalMinutes := timeutil.HoursToMinutes(rf.DurationHours)
	if totalMinutes <= 0 {
		totalMinutes = int(arrTime.Sub(depTime) / time.Minute)
	}

	stops := 0
	if !rf.DirectFlight {
		stops = len(rf.Stops)
	}

	cabinClass := strings.ToLower(strings.TrimSpace(rf.CabinClass))
	if cabinClass == "" {
		cabinClass = "economy"
	}

	carryOn, checked := parseBaggageNote(rf.BaggageNote)

	return domain.Flight{
		ID:       fmt.Sprintf("%s_%s", rf.FlightCode, providerName),
		Provider: providerName,
		Airline: domain.Airline{
			Name: rf.Airline,
			Code: extractAirlineCode(rf.FlightCode),
		},
		FlightNumber: rf.FlightCode,
		Departure: domain.LocationPoint{
			Airport:   rf.FromAirport,
			City:      provider.ResolveCity(rf.FromAirport, ""),
			Datetime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.LocationPoint{
			Airport:   rf.ToAirport,
			City:      provider.ResolveCity(rf.ToAirport, ""),
			Datetime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration: domain.DurationInfo{
			TotalMinutes: totalMinutes,
			Formatted:    timeutil.FormatDuration(totalMinutes),
		},
		Stops: stops,
		Price: domain.PriceInfo{
			Amount:    rf.PriceIDR,
			Currency:  "IDR",
			Formatted: currency.FormatIDR(rf.PriceIDR),
		},
		AvailableSeats: rf.Seats,
		CabinClass:     cabinClass,
		Aircraft:       nil,
		Amenities:      []string{},
		Baggage: domain.BaggageInfo{
			CarryOn: carryOn,
			Checked: checked,
		},
	}, nil
}

func extractAirlineCode(flightCode string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(flightCode) {
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			continue
		}
		break
	}
	if b.Len() > 0 {
		return b.String()
	}
	return "QZ"
}

func parseBaggageNote(note string) (carryOn, checked string) {
	lower := strings.ToLower(note)
	carryOn = "Cabin baggage only"
	checked = "None"
	if strings.TrimSpace(note) == "" {
		carryOn = "None"
	}
	if strings.Contains(lower, "additional fee") {
		checked = "Additional fee"
	}
	return carryOn, checked
}
