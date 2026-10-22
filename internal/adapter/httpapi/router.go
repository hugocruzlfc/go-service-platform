package httpapi

import (
	"log/slog"
	"net/http"
)

// NewRouter composes already-built HTTP handlers into the application router.
// Task routes require a valid bearer access token; health, version, docs, and
// authentication routes stay public.
func NewRouter(
	apiHandler http.Handler,
	tokenVerifier AccessTokenVerifier,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()
	if apiHandler != nil {
		mux.Handle("/", apiHandler)
		protected := RequireAccessToken(tokenVerifier, logger)(apiHandler)
		mux.Handle("/tasks", protected)
		mux.Handle("/tasks/", protected)
	}
	mux.Handle("GET /openapi.json", openAPIHandler(logger))
	mux.Handle("GET /docs", swaggerUIHandler())
	return mux
}
