package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
)

// newLogCapture returns a logger that writes JSON lines to the returned buffer.
func newLogCapture(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: level})
	return slog.New(handler), &buffer
}

func lastLogLine(t *testing.T, buffer *bytes.Buffer) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	return entry
}

func TestRequestLoggerGeneratesRequestID(t *testing.T) {
	logger, _ := newLogCapture(slog.LevelInfo)
	handler := RequestLogger(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))

	if id := rec.Header().Get(requestIDHeader); len(id) != 32 {
		t.Fatalf("generated request ID = %q, want 32 hex characters", id)
	}
}

func TestRequestLoggerReusesValidClientRequestID(t *testing.T) {
	logger, _ := newLogCapture(slog.LevelInfo)
	handler := RequestLogger(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req.Header.Set(requestIDHeader, "client-abc-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if id := rec.Header().Get(requestIDHeader); id != "client-abc-123" {
		t.Fatalf("request ID = %q, want the client value", id)
	}
}

func TestRequestLoggerReplacesUnsafeClientRequestID(t *testing.T) {
	logger, _ := newLogCapture(slog.LevelInfo)
	handler := RequestLogger(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req.Header.Set(requestIDHeader, "bad id\nwith newline")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if id := rec.Header().Get(requestIDHeader); id == "bad id\nwith newline" || len(id) != 32 {
		t.Fatalf("unsafe request ID was not replaced: %q", id)
	}
}

func TestRequestLoggerLogsRouteStatusAndDuration(t *testing.T) {
	logger, buffer := newLogCapture(slog.LevelInfo)
	router := NewRouter(
		NewAPIHandler(application.NewTaskService(memory.NewTaskRepository()), nil, NewReadiness(fakeDatabasePinger{}, logger), logger),
		fakeTokenVerifier{},
		logger,
	)
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-1?token=secret", nil)
	req.Header.Set("Authorization", "Bearer "+testAccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	entry := lastLogLine(t, buffer)
	if entry["route"] != "GET /tasks/{id}" {
		t.Fatalf("route = %v, want the mux pattern", entry["route"])
	}
	if entry["status"] != float64(http.StatusNotFound) {
		t.Fatalf("status = %v, want 404 for a missing task", entry["status"])
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Fatal("duration_ms is missing")
	}
	if strings.Contains(buffer.String(), "secret") {
		t.Fatal("query string must not be logged")
	}
}

func TestRequestLoggerUsesErrorLevelForServerErrors(t *testing.T) {
	logger, buffer := newLogCapture(slog.LevelInfo)
	handler := RequestLogger(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	if entry := lastLogLine(t, buffer); entry["level"] != "ERROR" {
		t.Fatalf("level = %v, want ERROR", entry["level"])
	}
}

func TestRequestLoggerLogsHealthChecksAtDebugOnly(t *testing.T) {
	logger, buffer := newLogCapture(slog.LevelInfo)
	handler := RequestLogger(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if buffer.Len() != 0 {
		t.Fatalf("health check was logged at INFO: %s", buffer.String())
	}

	debugLogger, debugBuffer := newLogCapture(slog.LevelDebug)
	RequestLogger(debugLogger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if entry := lastLogLine(t, debugBuffer); entry["level"] != "DEBUG" {
		t.Fatalf("level = %v, want DEBUG", entry["level"])
	}
}
