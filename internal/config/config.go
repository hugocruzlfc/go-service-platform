package config

import (
	"fmt"
	"os"
	"strconv"
)

const defaultPort = 8080

type Config struct {
	HTTPAddr string
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

	return Config{
		HTTPAddr: fmt.Sprintf(":%d", port),
	}, nil
}
