package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

type fakeAuthenticationService struct {
	registerUser  domain.User
	registerErr   error
	loginResult   application.LoginResult
	loginErr      error
	refreshResult application.RefreshResult
	refreshErr    error
	logoutErr     error
	logoutToken   string
}

func (s *fakeAuthenticationService) Register(context.Context, application.RegisterInput) (domain.User, error) {
	return s.registerUser, s.registerErr
}

func (s *fakeAuthenticationService) Login(context.Context, application.LoginInput) (application.LoginResult, error) {
	return s.loginResult, s.loginErr
}

func (s *fakeAuthenticationService) Refresh(context.Context, string) (application.RefreshResult, error) {
	return s.refreshResult, s.refreshErr
}

func (s *fakeAuthenticationService) Logout(_ context.Context, refreshToken string) error {
	s.logoutToken = refreshToken
	return s.logoutErr
}

func newAuthTestHandler(
	authenticationService application.AuthenticationUseCase,
) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewAuthHandler(authenticationService, logger)
}

func TestAuthHandlerRegisterReturnsCreatedUser(t *testing.T) {
	handler := newAuthTestHandler(
		&fakeAuthenticationService{registerUser: domain.User{ID: "user-1", Email: "alice@example.com"}},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/register",
		strings.NewReader(`{"email":"alice@example.com","password":"secret"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if !strings.Contains(response.Body.String(), `"email":"alice@example.com"`) {
		t.Fatalf("response = %s", response.Body.String())
	}
}

func TestAuthHandlerLoginSetsSecureRefreshCookie(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	handler := newAuthTestHandler(
		&fakeAuthenticationService{loginResult: application.LoginResult{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    expiresAt,
		}},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(`{"email":"alice@example.com","password":"secret"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	cookie := response.Result().Cookies()[0]
	if cookie.Name != refreshCookieName || cookie.Value != "refresh-token" {
		t.Fatalf("cookie = %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security attributes = %#v", cookie)
	}
}

func TestAuthHandlerRefreshReadsCookie(t *testing.T) {
	session := &fakeAuthenticationService{refreshResult: application.RefreshResult{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}}
	handler := newAuthTestHandler(session)
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "old-refresh-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if session.logoutToken != "" {
		t.Fatal("refresh must not call logout")
	}
}

func TestAuthHandlerMapsInvalidLoginToUnauthorized(t *testing.T) {
	handler := newAuthTestHandler(
		&fakeAuthenticationService{loginErr: domain.ErrInvalidCredentials},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(`{"email":"alice@example.com","password":"wrong"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlerLogoutClearsCookie(t *testing.T) {
	session := &fakeAuthenticationService{}
	handler := newAuthTestHandler(session)
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if session.logoutToken != "refresh-token" {
		t.Fatalf("logout token = %q", session.logoutToken)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("clear cookie = %#v", cookies)
	}
}

func TestAuthHandlerMapsRegistrationConflict(t *testing.T) {
	handler := newAuthTestHandler(
		&fakeAuthenticationService{registerErr: domain.ErrUserEmailTaken},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/register",
		strings.NewReader(`{"email":"alice@example.com","password":"secret"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestAuthHandlerUsesGenericInternalError(t *testing.T) {
	handler := newAuthTestHandler(
		&fakeAuthenticationService{loginErr: errors.New("database unavailable")},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(`{"email":"alice@example.com","password":"secret"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
