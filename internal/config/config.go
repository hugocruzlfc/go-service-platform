package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	defaultPort               = 8080
	defaultAuthJWTIssuer      = "service-platform"
	defaultAuthJWTAudience    = "service-platform-api"
	defaultAuthAccessTokenTTL = 15 * time.Minute
	//nolint:gosec // G101: development-only credentials for the local Docker database.
	defaultDatabaseURL = "postgres://service_platform:service_platform_dev@localhost:5432/service_platform?sslmode=disable"
)

// Config contains validated application configuration.
type Config struct {
	HTTPAddr                string
	DatabaseURL             string
	AuthJWTPrivateKeyBase64 string
	AuthJWTIssuer           string
	AuthJWTAudience         string
	AuthAccessTokenTTL      time.Duration
}

// Load reads application configuration from environment variables.
func Load() (Config, error) {
	port := defaultPort

	if rawPort := os.Getenv("PORT"); rawPort != "" {
		parsedPort, err := strconv.Atoi(rawPort)
		if err != nil {
			return Config{}, fmt.Errorf("parse PORT: %w", err)
		}

		if parsedPort < 1 || parsedPort > 65535 {
			return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
		}

		port = parsedPort
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	if err := validateDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	accessTokenTTL := defaultAuthAccessTokenTTL
	if rawTTL := os.Getenv("AUTH_ACCESS_TOKEN_TTL"); rawTTL != "" {
		parsedTTL, err := time.ParseDuration(rawTTL)
		if err != nil || parsedTTL <= 0 {
			return Config{}, errors.New("AUTH_ACCESS_TOKEN_TTL must be a positive duration")
		}
		accessTokenTTL = parsedTTL
	}

	issuer := os.Getenv("AUTH_JWT_ISSUER")
	if issuer == "" {
		issuer = defaultAuthJWTIssuer
	}
	audience := os.Getenv("AUTH_JWT_AUDIENCE")
	if audience == "" {
		audience = defaultAuthJWTAudience
	}

	return Config{
		HTTPAddr:                fmt.Sprintf(":%d", port),
		DatabaseURL:             databaseURL,
		AuthJWTPrivateKeyBase64: os.Getenv("AUTH_JWT_PRIVATE_KEY_BASE64"),
		AuthJWTIssuer:           issuer,
		AuthJWTAudience:         audience,
		AuthAccessTokenTTL:      accessTokenTTL,
	}, nil
}

func validateDatabaseURL(databaseURL string) error {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil || parsedURL.Host == "" {
		return errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return errors.New("DATABASE_URL must use the postgres or postgresql scheme")
	}

	return nil
}
