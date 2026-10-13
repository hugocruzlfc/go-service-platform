package httpapi

import (
	"net/http"

	"go-service-platform/internal/task/application"
)

func NewRouter(service *application.TaskService) http.Handler {
	mux := http.NewServeMux()

	tasks := newTaskHandler(service)

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /version", versionHandler)
	mux.HandleFunc("POST /tasks", tasks.createTask)
	mux.HandleFunc("GET /tasks", tasks.listTasks)

	return mux
}
