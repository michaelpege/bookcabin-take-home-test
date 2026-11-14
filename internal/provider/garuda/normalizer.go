package garuda

import (
	"fmt"
	"strings"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/currency"
	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/timeutil"
	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

func (rf rawFlight) toDomain(providerName string) (domain.Flight, error) {
	if err := validateSegments(rf.FlightID, rf.Segments); err != nil {
		return domain.Flight{}, err
	}

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
			City:      provider.ResolveCity(depRaw.Airport, depRaw.City),
			Datetime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.LocationPoint{
			Airport:   arrRaw.Airport,
			City:      provider.ResolveCity(arrRaw.Airport, arrRaw.City),
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

func validateSegments(flightID string, segments []rawSegment) error {
	if len(segments) == 0 {
		return nil
	}
	var prevArr time.Time
	var prevAirport string
	for i, seg := range segments {
		if seg.DurationMinutes < 0 || seg.LayoverMinutes < 0 {
			return fmt.Errorf("negative segment duration or layover for %s", flightID)
		}
		dep, err := timeutil.ParseTimestamp(seg.Departure.Time, "")
		if err != nil {
			return fmt.Errorf("parse segment %d departure for %s: %w", i, flightID, err)
		}
		arr, err := timeutil.ParseTimestamp(seg.Arrival.Time, "")
		if err != nil {
			return fmt.Errorf("parse segment %d arrival for %s: %w", i, flightID, err)
		}
		if !arr.After(dep) {
			return fmt.Errorf("segment %d arrival not after departure for %s", i, flightID)
		}
		if i > 0 {
			if !strings.EqualFold(prevAirport, seg.Departure.Airport) {
				return fmt.Errorf("segment discontinuity for %s: %s -> %s", flightID, prevAirport, seg.Departure.Airport)
			}
			if dep.Before(prevArr) {
				return fmt.Errorf("segment %d departs before previous segment arrival for %s", i, flightID)
			}
			if seg.LayoverMinutes > 0 {
				actualLayover := int(dep.Sub(prevArr) / time.Minute)
				if diff := actualLayover - seg.LayoverMinutes; diff < -1 || diff > 1 {
					return fmt.Errorf("segment %d layover mismatch for %s: %dm vs %dm", i, flightID, seg.LayoverMinutes, actualLayover)
				}
			}
		}
		prevArr = arr
		prevAirport = seg.Arrival.Airport
	}
	return nil
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
