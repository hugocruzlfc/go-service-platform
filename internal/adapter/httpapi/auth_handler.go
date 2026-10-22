package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

const refreshCookieName = "refresh_token"

type authHTTPHandler struct {
	authenticationService application.AuthenticationUseCase
	logger                *slog.Logger
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authTokenResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// NewAuthHandler creates the HTTP routes for authentication operations.
func NewAuthHandler(
	authenticationService application.AuthenticationUseCase,
	logger *slog.Logger,
) http.Handler {
	handler := &authHTTPHandler{
		authenticationService: authenticationService,
		logger:                logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", handler.registerUser)
	mux.HandleFunc("POST /auth/login", handler.loginUser)
	mux.HandleFunc("POST /auth/refresh", handler.refreshSession)
	mux.HandleFunc("POST /auth/logout", handler.logout)
	return mux
}

func (h *authHTTPHandler) registerUser(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if err := decodeAuthJSON(w, r, &request); err != nil {
		writeError(h.logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	user, err := h.authenticationService.Register(r.Context(), application.RegisterInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserEmailTaken):
			writeError(h.logger, w, http.StatusConflict, "EMAIL_TAKEN", "email is already registered")
		case errors.Is(err, domain.ErrUserEmailRequired),
			errors.Is(err, domain.ErrUserEmailInvalid),
			errors.Is(err, domain.ErrPasswordRequired):
			writeError(h.logger, w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid registration data")
		default:
			h.internalError(w, "registering user", err)
		}
		return
	}

	writeJSON(h.logger, w, http.StatusCreated, user)
}

func (h *authHTTPHandler) loginUser(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeAuthJSON(w, r, &request); err != nil {
		writeError(h.logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	result, err := h.authenticationService.Login(r.Context(), application.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(h.logger, w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		h.internalError(w, "logging in user", err)
		return
	}

	setRefreshCookie(w, result.RefreshToken)
	writeJSON(h.logger, w, http.StatusOK, authTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
	})
}

func (h *authHTTPHandler) refreshSession(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshTokenFromRequest(w, r)
	if err != nil {
		writeError(h.logger, w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}

	result, err := h.authenticationService.Refresh(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenInvalid) {
			writeError(h.logger, w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
			return
		}
		h.internalError(w, "refreshing session", err)
		return
	}

	setRefreshCookie(w, result.RefreshToken)
	writeJSON(h.logger, w, http.StatusOK, authTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
	})
}

func (h *authHTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshTokenFromRequest(w, r)
	if err != nil {
		clearRefreshCookie(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.authenticationService.Logout(r.Context(), refreshToken); err != nil {
		if !errors.Is(err, domain.ErrRefreshTokenInvalid) {
			h.internalError(w, "logging out user", err)
			return
		}
	}

	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHTTPHandler) internalError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, slog.Any("error", err))
	writeError(h.logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func decodeAuthJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request contains more than one JSON value")
	}
	return nil
}

func refreshTokenFromRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	var request refreshRequest
	if err := decodeAuthJSON(w, r, &request); err != nil || strings.TrimSpace(request.RefreshToken) == "" {
		return "", errors.New("refresh token is required")
	}
	return request.RefreshToken, nil
}

func setRefreshCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/auth",
		MaxAge:   int((7 * 24 * time.Hour) / time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
