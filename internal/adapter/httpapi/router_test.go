package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"service-platform/internal/api"
	authapplication "service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
)

type fakeTokenVerifier struct{}

func (fakeTokenVerifier) ParseAccessToken(encodedToken string) (authapplication.AccessTokenClaims, error) {
	if encodedToken != testAccessToken {
		return authapplication.AccessTokenClaims{}, domain.ErrAccessTokenInvalid
	}

	return authapplication.AccessTokenClaims{UserID: "user-1", SessionID: "session-1", TokenID: "token-1"}, nil
}

type fakeDatabasePinger struct {
	err error
}

func (f fakeDatabasePinger) Ping(context.Context) error {
	return f.err
}

func newTestRouter() http.Handler {
	repository := memory.NewTaskRepository()
	taskService := application.NewTaskService(repository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	readiness := NewReadiness(fakeDatabasePinger{}, logger)
	apiHandler := NewAPIHandler(taskService, readiness, logger)

	return NewRouter(apiHandler, nil, fakeTokenVerifier{}, logger)
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

	var response api.VersionResponse

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

func TestOpenAPIRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected content type %q, got %q", "application/json", contentType)
	}

	if !strings.Contains(rec.Body.String(), `"openapi":"3.0.3"`) {
		t.Error("expected the embedded OpenAPI document")
	}
}

func TestDocsRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("expected content type %q, got %q", "text/html; charset=utf-8", contentType)
	}

	if !strings.Contains(rec.Body.String(), "SwaggerUIBundle") {
		t.Error("expected Swagger UI page")
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
			taskService := application.NewTaskService(repository)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			readiness := NewReadiness(fakeDatabasePinger{err: test.databaseErr}, logger)
			apiHandler := NewAPIHandler(taskService, readiness, logger)
			router := NewRouter(apiHandler, nil, fakeTokenVerifier{}, logger)

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

func TestReadinessRouteDuringShutdown(t *testing.T) {
	repository := memory.NewTaskRepository()
	taskService := application.NewTaskService(repository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	readiness := NewReadiness(fakeDatabasePinger{}, logger)
	readiness.BeginDraining()
	apiHandler := NewAPIHandler(taskService, readiness, logger)
	router := NewRouter(apiHandler, nil, fakeTokenVerifier{}, logger)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	livenessRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	livenessResponse := httptest.NewRecorder()

	router.ServeHTTP(livenessResponse, livenessRequest)

	if livenessResponse.Code != http.StatusOK {
		t.Fatalf(
			"expected liveness status %d, got %d",
			http.StatusOK,
			livenessResponse.Code,
		)
	}
}
