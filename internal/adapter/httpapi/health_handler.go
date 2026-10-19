package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// DatabasePinger checks whether the database is reachable.
type DatabasePinger interface {
	Ping(context.Context) error
}

// HealthResponse is the response returned by the health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encoding health response: %v", err)
	}
}

func readinessHandler(pinger DatabasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pinger.Ping(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "NOT_READY", "service is not ready")
			return
		}

		writeJSON(w, http.StatusOK, HealthResponse{
			Status: "ready",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	}
}
