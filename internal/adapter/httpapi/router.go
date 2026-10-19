package httpapi

import (
	"log/slog"
	"net/http"

	"service-platform/internal/task/application"
)

// NewRouter creates the HTTP handler tree for the task API.
func NewRouter(
	service *application.TaskService,
	databasePinger DatabasePinger,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	tasks := newTaskHandler(service, logger)

	mux.Handle("GET /health", healthHandler(logger))
	mux.Handle("GET /healthz", healthHandler(logger))
	mux.Handle("GET /readyz", readinessHandler(databasePinger, logger))
	mux.Handle("GET /version", versionHandler(logger))
	mux.HandleFunc("POST /tasks", tasks.createTask)
	mux.HandleFunc("GET /tasks", tasks.listTasks)

	return mux
}
