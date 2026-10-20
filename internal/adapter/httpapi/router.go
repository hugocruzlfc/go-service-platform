package httpapi

import (
	"log/slog"
	"net/http"

	"service-platform/internal/api"
	"service-platform/internal/task/application"
)

// NewRouter creates the HTTP handler tree for the task API.
func NewRouter(
	service *application.TaskService,
	readiness *Readiness,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /openapi.json", openAPIHandler(logger))
	mux.Handle("GET /docs", swaggerUIHandler())

	server := newStrictServer(service, readiness, logger)
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("writing API response", slog.Any("error", err))
			writeError(logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		},
	})

	api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		},
	})

	return requestBodyMiddleware(logger, mux)
}
