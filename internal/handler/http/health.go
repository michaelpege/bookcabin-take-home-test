package http

import (
	"net/http"
	"time"
)

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
