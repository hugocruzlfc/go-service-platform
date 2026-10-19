package config

import (
	"fmt"
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

	return Config{
		HTTPAddr:    fmt.Sprintf(":%d", port),
		DatabaseURL: databaseURL,
	}, nil
}
