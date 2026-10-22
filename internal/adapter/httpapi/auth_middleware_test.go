package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaskRoutesRequireAccessToken(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing header"},
		{name: "wrong scheme", authorization: "Basic " + testAccessToken},
		{name: "empty token", authorization: "Bearer   "},
		{name: "invalid token", authorization: "Bearer not-a-valid-token"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			if test.authorization != "" {
				req.Header.Set("Authorization", test.authorization)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assertErrorResponse(t, rec, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		})
	}
}

func TestTaskRoutesAcceptValidAccessToken(t *testing.T) {
	router := newTestRouter()
	req := newTaskRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestPublicRoutesDoNotRequireAccessToken(t *testing.T) {
	router := newTestRouter()

	for _, path := range []string{"/health", "/healthz", "/version", "/openapi.json"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusUnauthorized {
				t.Fatalf("expected %s to be public, got %d", path, rec.Code)
			}
		})
	}
}

func TestRequireAccessTokenPassesRequestsWithValidToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAccessToken(fakeTokenVerifier{}, logger)(next)
	req := newTaskRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected the next handler to be called")
	}
}
