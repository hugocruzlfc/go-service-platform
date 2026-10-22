package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "postgres://example/database")

	config, err := Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.HTTPAddr != ":8080" {
		t.Errorf("expected address %q, got %q", ":8080", config.HTTPAddr)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is not set")
	}
}

func TestLoadUsesConfiguredPort(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://example/database")

	config, err := Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.HTTPAddr != ":9090" {
		t.Errorf("expected address %q, got %q", ":9090", config.HTTPAddr)
	}

	if config.DatabaseURL != "postgres://example/database" {
		t.Errorf("expected database URL %q, got %q", "postgres://example/database", config.DatabaseURL)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "not a number", port: "http"},
		{name: "too low", port: "0"},
		{name: "too high", port: "65536"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PORT", test.port)

			if _, err := Load(); err == nil {
				t.Fatalf("expected an error for PORT=%q", test.port)
			}
		})
	}
}

func TestLoadRejectsInvalidDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "missing host", url: "postgres://"},
		{name: "unsupported scheme", url: "mysql://user:password@localhost/database"},
		{name: "malformed URL", url: "postgres://%zz"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", test.url)

			if _, err := Load(); err == nil {
				t.Fatalf("expected an error for DATABASE_URL=%q", test.url)
			}
		})
	}
}

func TestLoadUsesConfiguredAuthAccessTokenTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example/database")
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "10m")

	config, err := Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.AuthAccessTokenTTL != 10*time.Minute {
		t.Errorf("expected access token TTL %v, got %v", 10*time.Minute, config.AuthAccessTokenTTL)
	}
	if config.AuthJWTIssuer != defaultAuthJWTIssuer {
		t.Errorf("expected default issuer %q, got %q", defaultAuthJWTIssuer, config.AuthJWTIssuer)
	}
	if config.AuthJWTAudience != defaultAuthJWTAudience {
		t.Errorf("expected default audience %q, got %q", defaultAuthJWTAudience, config.AuthJWTAudience)
	}
}

func TestLoadRejectsInvalidAuthAccessTokenTTL(t *testing.T) {
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a non-positive access token TTL")
	}
}
