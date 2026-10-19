package httpapi

import (
	"net/http"

	"service-platform/internal/task/application"
)

// NewRouter creates the HTTP handler tree for the task API.
func NewRouter(
	service *application.TaskService,
	databasePinger DatabasePinger,
) http.Handler {
	mux := http.NewServeMux()

	tasks := newTaskHandler(service)

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.Handle("GET /readyz", readinessHandler(databasePinger))
	mux.HandleFunc("GET /version", versionHandler)
	mux.HandleFunc("POST /tasks", tasks.createTask)
	mux.HandleFunc("GET /tasks", tasks.listTasks)

	return mux
}
