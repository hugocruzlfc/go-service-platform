package httpapi

import (
	"context"
	"log/slog"
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

func healthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(logger, w, http.StatusOK, HealthResponse{
			Status: "ok",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func readinessHandler(
	pinger DatabasePinger,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pinger.Ping(r.Context()); err != nil {
			logger.Warn("readiness check failed", slog.Any("error", err))
			writeError(logger, w, http.StatusServiceUnavailable, "NOT_READY", "service is not ready")
			return
		}

		writeJSON(logger, w, http.StatusOK, HealthResponse{
			Status: "ready",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	}
}
