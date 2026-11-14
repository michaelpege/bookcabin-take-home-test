package http

import (
	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/usecase"
)

// SearchFlightsRequest matches the PDF take-home Search Request JSON contract
// (camelCase inputs) while allowing optional filter/sort extensions.
type SearchFlightsRequest struct {
	Origin        string  `json:"origin"`
	Destination   string  `json:"destination"`
	DepartureDate string  `json:"departureDate"`
	ReturnDate    *string `json:"returnDate"`
	Passengers    int     `json:"passengers"`
	CabinClass    string  `json:"cabinClass"`
	SortBy        string  `json:"sortBy,omitempty"`
}

func (r SearchFlightsRequest) toDomain() (domain.SearchCriteria, usecase.SearchOptions) {
	criteria := domain.SearchCriteria{
		Origin:        r.Origin,
		Destination:   r.Destination,
		DepartureDate: r.DepartureDate,
		ReturnDate:    r.ReturnDate,
		Passengers:    r.Passengers,
		CabinClass:    r.CabinClass,
	}
	opts := usecase.SearchOptions{
		SortBy: domain.SortBy(r.SortBy),
	}
	return criteria, opts
}
