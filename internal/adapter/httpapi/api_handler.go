package httpapi

import (
	"log/slog"
	"net/http"

	"service-platform/internal/api"
	authapplication "service-platform/internal/auth/application"
	"service-platform/internal/task/application"
)

// NewAPIHandler creates the HTTP handler for the OpenAPI endpoints.
func NewAPIHandler(
	taskService *application.TaskService,
	authenticationService authapplication.AuthenticationUseCase,
	readiness *Readiness,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	server := newOpenAPIServer(taskService, authenticationService, readiness, logger)
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			logger.Error("writing API response", slog.Any("error", err))
			writeError(logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		},
	})

	api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		},
	})

	return requestBodyMiddleware(logger, mux)
}
