// Package task composes the task module.
package task

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/postgres"
)

// NewHTTPHandler creates the task HTTP handler and its internal dependencies.
func NewHTTPHandler(
	databasePool *pgxpool.Pool,
	readiness *httpapi.Readiness,
	logger *slog.Logger,
) http.Handler {
	taskRepository := postgres.NewTaskRepository(databasePool)
	taskService := application.NewTaskService(taskRepository)

	return httpapi.NewAPIHandler(taskService, readiness, logger)
}
