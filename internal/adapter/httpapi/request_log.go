package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

const requestIDHeader = "X-Request-ID"

// validRequestID accepts client-supplied IDs that are safe to echo back and log.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// healthPaths are polled frequently by probes, so they are logged at DEBUG.
var healthPaths = map[string]bool{
	"/health":  true,
	"/healthz": true,
	"/readyz":  true,
}

// RequestLogger assigns a request ID to every request and logs one line per
// request with the method, route, status, and duration. It never logs query
// strings, headers, or bodies, because those can contain tokens and passwords.
func RequestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFrom(r)
		w.Header().Set(requestIDHeader, requestID)

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}

		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		}

		switch {
		case healthPaths[r.URL.Path]:
			logger.Debug("request completed", attrs...)
		case status >= http.StatusInternalServerError:
			logger.Error("request completed", attrs...)
		default:
			logger.Info("request completed", attrs...)
		}
	})
}

// requestIDFrom reuses a valid X-Request-ID sent by the client, or generates one.
func requestIDFrom(r *http.Request) string {
	if id := r.Header.Get(requestIDHeader); validRequestID.MatchString(id) {
		return id
	}

	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(bytes[:])
}

// statusRecorder remembers the status code written by the handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
