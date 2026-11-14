package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/platform/logger"
)

// NewRouter constructs the HTTP handler with health/ping endpoints and middleware.
func NewRouter(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("GET /health", handleHealth)

	return recoverMiddleware(log, logger.Middleware(log, mux))
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "pong"})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"service":   "bookcabin-flight-aggregator",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
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
