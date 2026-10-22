// Package auth composes the authentication module for the application.
package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/auth/application"
	"service-platform/internal/auth/infrastructure/argon2"
	"service-platform/internal/auth/infrastructure/postgres"
	"service-platform/internal/auth/infrastructure/token"
)

// NewHTTPHandler creates the complete authentication module and its HTTP handler.
func NewHTTPHandler(
	databasePool *pgxpool.Pool,
	encodedJWTPrivateKey string,
	jwtIssuer string,
	jwtAudience string,
	accessTokenTTL time.Duration,
	logger *slog.Logger,
) (http.Handler, error) {
	jwtPrivateKey, err := decodeJWTPrivateKey(encodedJWTPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("load JWT private key: %w", err)
	}

	accessTokenService, err := token.NewService(
		jwtPrivateKey,
		jwtIssuer,
		jwtAudience,
		accessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("configure access tokens: %w", err)
	}

	userRepository := postgres.NewUserRepository(databasePool)
	credentialRepository := postgres.NewCredentialRepository(databasePool)
	registrationRepository := postgres.NewRegistrationRepository(databasePool)
	sessionRepository := postgres.NewSessionRepository(databasePool)
	passwordHasher := argon2.NewPasswordHasher()
	refreshTokenManager := token.RefreshTokenManager{}

	registrationService := application.NewAuthService(
		registrationRepository,
		passwordHasher,
	)
	loginService := application.NewLoginService(
		userRepository,
		credentialRepository,
		passwordHasher,
		sessionRepository,
		refreshTokenManager,
		accessTokenService,
	)
	sessionService := application.NewSessionService(
		sessionRepository,
		refreshTokenManager,
		accessTokenService,
	)
	authenticationService := application.NewAuthenticationService(
		registrationService,
		loginService,
		sessionService,
	)

	return httpapi.NewAuthHandler(authenticationService, logger), nil
}

func decodeJWTPrivateKey(encodedKey string) (ed25519.PrivateKey, error) {
	if encodedKey == "" {
		return nil, errors.New("AUTH_JWT_PRIVATE_KEY_BASE64 is required")
	}

	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(encodedKey)
	}
	if err != nil {
		return nil, fmt.Errorf("decode AUTH_JWT_PRIVATE_KEY_BASE64: %w", err)
	}

	switch len(key) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(key), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(key), nil
	default:
		return nil, fmt.Errorf(
			"AUTH_JWT_PRIVATE_KEY_BASE64 must decode to %d-byte seed or %d-byte private key",
			ed25519.SeedSize,
			ed25519.PrivateKeySize,
		)
	}
}
