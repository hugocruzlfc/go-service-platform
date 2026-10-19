package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
)

type fakeDatabasePinger struct {
	err error
}

func (f fakeDatabasePinger) Ping(context.Context) error {
	return f.err
}

func newTestRouter() http.Handler {
	repository := memory.NewTaskRepository()
	service := application.NewTaskService(repository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewRouter(service, fakeDatabasePinger{}, logger)
}

func TestVersionRoute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(
		http.MethodGet,
		"/version",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	var response VersionResponse

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if response.Version != "1.0.0" {
		t.Errorf(
			"expected version %q, got %q",
			"1.0.0",
			response.Version,
		)
	}
}

func TestHealthRoute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}
}

func TestLivenessRoute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestReadinessRoute(t *testing.T) {
	tests := []struct {
		name           string
		databaseErr    error
		expectedStatus int
	}{
		{
			name:           "database is available",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "database is unavailable",
			databaseErr:    context.Canceled,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := memory.NewTaskRepository()
			service := application.NewTaskService(repository)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			router := NewRouter(service, fakeDatabasePinger{err: test.databaseErr}, logger)

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != test.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					test.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}
