package httpapi

import (
	"log/slog"
	"net/http"
)

// VersionResponse is the response returned by the version endpoint.
type VersionResponse struct {
	Version string `json:"version"`
}

func versionHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(logger, w, http.StatusOK, VersionResponse{
			Version: "1.0.0",
		})
	}
}
