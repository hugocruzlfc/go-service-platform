package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
)

func newTestRouter() http.Handler {
	repository := memory.NewTaskRepository()
	service := application.NewTaskService(repository)

	return NewRouter(service)
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
