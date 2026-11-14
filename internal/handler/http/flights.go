package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
	"github.com/michaelpege/bookcabin-take-home-test/internal/usecase"
)

// SearchService defines the inbound usecase port required by the flight HTTP handler.
type SearchService interface {
	SearchFlights(ctx context.Context, criteria domain.SearchCriteria, opts usecase.SearchOptions) (domain.SearchResult, error)
}

func handleSearchFlights(svc SearchService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SearchFlightsRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid request payload: " + err.Error(),
			})
			return
		}

		criteria, opts := req.toDomain()
		result, err := svc.SearchFlights(r.Context(), criteria, opts)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrSearchInvalidCriteria):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			case errors.Is(err, domain.ErrAllProvidersFailed):
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"error":    err.Error(),
					"metadata": result.Metadata,
				})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "flight search failed"})
			}
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}
