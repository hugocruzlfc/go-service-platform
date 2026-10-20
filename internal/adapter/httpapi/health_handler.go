package httpapi

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// DatabasePinger checks whether the database is reachable.
type DatabasePinger interface {
	Ping(context.Context) error
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
