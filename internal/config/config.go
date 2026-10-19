package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
)

const defaultPort = 8080
const defaultDatabaseURL = "postgres://service_platform:service_platform_dev@localhost:5432/service_platform?sslmode=disable"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

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

	return Config{
		HTTPAddr:    fmt.Sprintf(":%d", port),
		DatabaseURL: databaseURL,
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
