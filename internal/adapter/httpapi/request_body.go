package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"service-platform/internal/api"
)

func requestBodyMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks" {
			next.ServeHTTP(w, r)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()

		var request api.CreateTaskRequest
		if err := decoder.Decode(&request); err != nil {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}
