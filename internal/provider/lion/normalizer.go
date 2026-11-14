package lion

import (
	"fmt"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/currency"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
)

func (rf rawFlight) toDomain(providerName string) (domain.Flight, error) {
	depTime, err := timeutil.ParseTimestamp(rf.Schedule.Departure, rf.Schedule.DepartureTimezone)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure time for %s: %w", rf.ID, err)
	}
	arrTime, err := timeutil.ParseTimestamp(rf.Schedule.Arrival, rf.Schedule.ArrivalTimezone)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival time for %s: %w", rf.ID, err)
	}

	totalMinutes := rf.FlightTime
	if totalMinutes <= 0 {
		totalMinutes = int(arrTime.Sub(depTime) / time.Minute)
	}

	amenities := []string{}
	if rf.Services.WifiAvailable {
		amenities = append(amenities, "wifi")
	}
	if rf.Services.MealsIncluded {
		amenities = append(amenities, "meal")
	}

	var aircraft *string
	if trimmed := strings.TrimSpace(rf.PlaneType); trimmed != "" {
		aircraft = &trimmed
	}

	cabinClass := strings.ToLower(strings.TrimSpace(rf.Pricing.FareType))
	if cabinClass == "" {
		cabinClass = "economy"
	}

	return domain.Flight{
		ID:       fmt.Sprintf("%s_%s", rf.ID, providerName),
		Provider: providerName,
		Airline: domain.Airline{
			Name: rf.Carrier.Name,
			Code: rf.Carrier.IATA,
		},
		FlightNumber: rf.ID,
		Departure: domain.LocationPoint{
			Airport:   rf.Route.From.Code,
			City:      rf.Route.From.City,
			Datetime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.LocationPoint{
			Airport:   rf.Route.To.Code,
			City:      rf.Route.To.City,
			Datetime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration: domain.DurationInfo{
			TotalMinutes: totalMinutes,
			Formatted:    timeutil.FormatDuration(totalMinutes),
		},
		Stops: resolveStops(rf),
		Price: domain.PriceInfo{
			Amount:    rf.Pricing.Total,
			Currency:  rf.Pricing.Currency,
			Formatted: currency.FormatIDR(rf.Pricing.Total),
		},
		AvailableSeats: rf.SeatsLeft,
		CabinClass:     cabinClass,
		Aircraft:       aircraft,
		Amenities:      amenities,
		Baggage: domain.BaggageInfo{
			CarryOn: resolveBaggage(rf.Services.BaggageAllowance.Cabin),
			Checked: resolveBaggage(rf.Services.BaggageAllowance.Hold),
		},
	}, nil
}

func resolveStops(rf rawFlight) int {
	if rf.IsDirect {
		return 0
	}
	if rf.StopCount > 0 {
		return rf.StopCount
	}
	return len(rf.Layovers)
}

func resolveBaggage(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "None"
	}
	return trimmed
}
