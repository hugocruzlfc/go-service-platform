package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oapi-codegen/runtime/types"

	"service-platform/internal/api"
	authapplication "service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

const refreshCookieName = "refresh_token"

// The authentication operations live here so the OpenAPI contract is the only
// description of the routes. Their logic moved from the hand-written handler.

func (s *openAPIServer) RegisterUser(
	ctx context.Context,
	request api.RegisterUserRequestObject,
) (api.RegisterUserResponseObject, error) {
	user, err := s.authenticationService.Register(ctx, authRegisterInput(request.Body))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserEmailTaken):
			return api.RegisterUser409JSONResponse(errorBody("EMAIL_TAKEN", "email is already registered")), nil
		case errors.Is(err, domain.ErrUserEmailRequired),
			errors.Is(err, domain.ErrUserEmailInvalid),
			errors.Is(err, domain.ErrPasswordRequired):
			return api.RegisterUser400JSONResponse(errorBody("VALIDATION_ERROR", "invalid registration data")), nil
		default:
			s.logger.Error("registering user", slog.Any("error", err))
			return api.RegisterUser500JSONResponse(internalErrorResponse()), nil
		}
	}

	return api.RegisterUser201JSONResponse(toAPIUser(user)), nil
}

func (s *openAPIServer) LoginUser(
	ctx context.Context,
	request api.LoginUserRequestObject,
) (api.LoginUserResponseObject, error) {
	result, err := s.authenticationService.Login(ctx, authLoginInput(request.Body))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return api.LoginUser401JSONResponse(errorBody("UNAUTHORIZED", "invalid credentials")), nil
		}
		s.logger.Error("logging in user", slog.Any("error", err))
		return api.LoginUser500JSONResponse(internalErrorResponse()), nil
	}

	cookie := refreshCookie(result.RefreshToken).String()
	return api.LoginUser200JSONResponse{
		Body: api.AuthTokens{
			AccessToken:  result.AccessToken,
			RefreshToken: &result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		},
		Headers: api.LoginUser200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *openAPIServer) RefreshSession(
	ctx context.Context,
	request api.RefreshSessionRequestObject,
) (api.RefreshSessionResponseObject, error) {
	refreshToken, ok := refreshTokenFromRequest(request.Params.RefreshToken, request.Body)
	if !ok {
		return api.RefreshSession401JSONResponse(errorBody("UNAUTHORIZED", "invalid refresh token")), nil
	}

	result, err := s.authenticationService.Refresh(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenInvalid) {
			return api.RefreshSession401JSONResponse(errorBody("UNAUTHORIZED", "invalid refresh token")), nil
		}
		s.logger.Error("refreshing session", slog.Any("error", err))
		return api.RefreshSession500JSONResponse(internalErrorResponse()), nil
	}

	cookie := refreshCookie(result.RefreshToken).String()
	return api.RefreshSession200JSONResponse{
		Body: api.AuthTokens{
			AccessToken:  result.AccessToken,
			RefreshToken: &result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		},
		Headers: api.RefreshSession200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *openAPIServer) LogoutUser(
	ctx context.Context,
	request api.LogoutUserRequestObject,
) (api.LogoutUserResponseObject, error) {
	clearCookie := clearRefreshCookie().String()

	refreshToken, ok := refreshTokenFromRequest(request.Params.RefreshToken, request.Body)
	if !ok {
		return api.LogoutUser204Response{Headers: api.LogoutUser204ResponseHeaders{SetCookie: &clearCookie}}, nil
	}

	if err := s.authenticationService.Logout(ctx, refreshToken); err != nil && !errors.Is(err, domain.ErrRefreshTokenInvalid) {
		s.logger.Error("logging out user", slog.Any("error", err))
		return api.LogoutUser500JSONResponse(internalErrorResponse()), nil
	}

	return api.LogoutUser204Response{Headers: api.LogoutUser204ResponseHeaders{SetCookie: &clearCookie}}, nil
}

// refreshTokenFromRequest prefers the refresh_token cookie and falls back to the JSON body.
func refreshTokenFromRequest(cookie *string, body *api.RefreshRequest) (string, bool) {
	if cookie != nil && strings.TrimSpace(*cookie) != "" {
		return *cookie, true
	}

	if body != nil && body.RefreshToken != nil && strings.TrimSpace(*body.RefreshToken) != "" {
		return *body.RefreshToken, true
	}

	return "", false
}

func refreshCookie(refreshToken string) *http.Cookie {
	return &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/auth",
		MaxAge:   int((7 * 24 * time.Hour) / time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func errorBody(code, message string) api.ErrorResponse {
	return api.ErrorResponse{Error: api.ErrorDetail{Code: code, Message: message}}
}

func toAPIUser(user domain.User) api.User {
	return api.User{
		Id:        user.ID,
		Email:     types.Email(user.Email),
		CreatedAt: user.CreatedAt,
		Disabled:  user.Disabled,
	}
}

func authRegisterInput(body *api.RegisterRequest) authapplication.RegisterInput {
	if body == nil {
		return authapplication.RegisterInput{}
	}
	return authapplication.RegisterInput{Email: string(body.Email), Password: body.Password}
}

func authLoginInput(body *api.LoginRequest) authapplication.LoginInput {
	if body == nil {
		return authapplication.LoginInput{}
	}
	return authapplication.LoginInput{Email: string(body.Email), Password: body.Password}
}
