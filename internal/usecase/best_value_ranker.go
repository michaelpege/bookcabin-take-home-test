package usecase

import (
	"math"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

const (
	weightPrice     = 0.50
	weightDuration  = 0.30
	weightStops     = 0.15
	weightAmenities = 0.05
)

// ScoreBestValue computes a normalized [0.0, 1.0] composite BestValueScore on each
// flight using min-max scaling across Price (50%), Duration (30%), Stops (15%), and
// Amenities richness (5%). Higher scores indicate better overall value.
func ScoreBestValue(flights []domain.Flight) {
	if len(flights) == 0 {
		return
	}

	minPrice, maxPrice := flights[0].Price.Amount, flights[0].Price.Amount
	minDur, maxDur := flights[0].Duration.TotalMinutes, flights[0].Duration.TotalMinutes
	minStops, maxStops := flights[0].Stops, flights[0].Stops
	minAmen, maxAmen := len(flights[0].Amenities), len(flights[0].Amenities)

	for _, f := range flights[1:] {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minDur {
			minDur = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxDur {
			maxDur = f.Duration.TotalMinutes
		}
		if f.Stops < minStops {
			minStops = f.Stops
		}
		if f.Stops > maxStops {
			maxStops = f.Stops
		}
		amenCount := len(f.Amenities)
		if amenCount < minAmen {
			minAmen = amenCount
		}
		if amenCount > maxAmen {
			maxAmen = amenCount
		}
	}

	for i := range flights {
		priceScore := inverseMinMax(float64(flights[i].Price.Amount), float64(minPrice), float64(maxPrice))
		durScore := inverseMinMax(float64(flights[i].Duration.TotalMinutes), float64(minDur), float64(maxDur))
		stopsScore := inverseMinMax(float64(flights[i].Stops), float64(minStops), float64(maxStops))
		amenScore := directMinMax(float64(len(flights[i].Amenities)), float64(minAmen), float64(maxAmen))

		composite := weightPrice*priceScore + weightDuration*durScore + weightStops*stopsScore + weightAmenities*amenScore
		rounded := math.Round(composite*10000) / 10000
		flights[i].BestValueScore = &rounded
	}
}

func inverseMinMax(val, minVal, maxVal float64) float64 {
	if maxVal <= minVal {
		return 1.0
	}
	return (maxVal - val) / (maxVal - minVal)
}

func directMinMax(val, minVal, maxVal float64) float64 {
	if maxVal <= minVal {
		return 1.0
	}
	return (val - minVal) / (maxVal - minVal)
}
