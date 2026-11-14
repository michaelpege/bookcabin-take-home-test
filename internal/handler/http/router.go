package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
)

// NewRouter constructs the HTTP handler with health/ping endpoints, flight search,
// and recovery + request-logging middleware.
func NewRouter(log *slog.Logger, flightSearch SearchService) http.Handler {
	if flightSearch == nil {
		panic("http.NewRouter: flightSearch service must not be nil")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /api/v1/flights/search", handleSearchFlights(flightSearch))

	return recoverMiddleware(log, logger.Middleware(log, mux))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func recoverMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic_recovered", slog.Any("panic", rec), slog.String("path", r.URL.Path))
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
