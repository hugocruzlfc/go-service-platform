package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"service-platform/internal/api"
)

func requestBodyMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/tasks" {
			validateCreateTaskBody(logger, next, w, r)
			return
		}

		if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/tasks/") {
			validateUpdateTaskBody(logger, next, w, r)
			return
		}

		// Auth bodies are checked with the same strictness as the hand-written
		// handler used before: unknown fields and trailing JSON are rejected.
		if r.Method == http.MethodPost {
			switch r.URL.Path {
			case "/auth/register":
				validateJSONBody(logger, next, w, r, &api.RegisterRequest{}, false)
				return
			case "/auth/login":
				validateJSONBody(logger, next, w, r, &api.LoginRequest{}, false)
				return
			case "/auth/refresh", "/auth/logout":
				// The refresh token may come from the cookie, so an empty body is allowed.
				validateJSONBody(logger, next, w, r, &api.RefreshRequest{}, true)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// validateJSONBody rejects bodies that are not exactly one JSON object matching target.
// An empty body is accepted only when allowEmpty is true.
func validateJSONBody(logger *slog.Logger, next http.Handler, w http.ResponseWriter, r *http.Request, target any, allowEmpty bool) {
	body, err := readJSONBody(w, r)
	if err != nil {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if len(bytes.TrimSpace(body)) == 0 && !allowEmpty {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if len(bytes.TrimSpace(body)) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(target); err != nil || !decoderAtEOF(decoder) {
			writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	next.ServeHTTP(w, r)
}

func validateCreateTaskBody(logger *slog.Logger, next http.Handler, w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(w, r)
	if err != nil {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var request api.CreateTaskRequest
	if err := decoder.Decode(&request); err != nil || !decoderAtEOF(decoder) {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	next.ServeHTTP(w, r)
}

func validateUpdateTaskBody(logger *slog.Logger, next http.Handler, w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(w, r)
	if err != nil {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var request api.UpdateTaskRequest
	if err := decoder.Decode(&request); err != nil || !decoderAtEOF(decoder) {
		writeError(logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if request.Title == nil && request.Completed == nil {
		writeError(logger, w, http.StatusBadRequest, "VALIDATION_ERROR", "task update must contain at least one field")
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	next.ServeHTTP(w, r)
}

func readJSONBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	return io.ReadAll(r.Body)
}

func decoderAtEOF(decoder *json.Decoder) bool {
	return decoder.Decode(&struct{}{}) == io.EOF
}
