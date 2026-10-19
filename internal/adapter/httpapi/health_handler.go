package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
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

// Readiness manages whether the service can receive traffic.
type Readiness struct {
	pinger   DatabasePinger
	logger   *slog.Logger
	draining atomic.Bool
}

// NewReadiness creates a readiness checker backed by a database pinger.
func NewReadiness(pinger DatabasePinger, logger *slog.Logger) *Readiness {
	return &Readiness{
		pinger: pinger,
		logger: logger,
	}
}

// BeginDraining marks the service as unavailable for new traffic.
func (r *Readiness) BeginDraining() {
	r.draining.Store(true)
}

func healthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(logger, w, http.StatusOK, HealthResponse{
			Status: "ok",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func (readiness *Readiness) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if readiness.draining.Load() {
			readiness.logger.Info("readiness check rejected during shutdown")
			writeError(readiness.logger, w, http.StatusServiceUnavailable, "NOT_READY", "service is shutting down")
			return
		}

		if err := readiness.pinger.Ping(request.Context()); err != nil {
			readiness.logger.Warn("readiness check failed", slog.Any("error", err))
			writeError(readiness.logger, w, http.StatusServiceUnavailable, "NOT_READY", "service is not ready")
			return
		}

		writeJSON(readiness.logger, w, http.StatusOK, HealthResponse{
			Status: "ready",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	}
}
