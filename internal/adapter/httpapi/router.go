package httpapi

import (
	"log/slog"
	"net/http"
)

// NewRouter composes already-built HTTP handlers into the application router.
func NewRouter(
	apiHandler http.Handler,
	authenticationHandler http.Handler,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()
	if apiHandler != nil {
		mux.Handle("/", apiHandler)
	}
	if authenticationHandler != nil {
		mux.Handle("/auth/", authenticationHandler)
	}

	mux.Handle("GET /openapi.json", openAPIHandler(logger))
	mux.Handle("GET /docs", swaggerUIHandler())
	return mux
}
