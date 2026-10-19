package config

import "testing"

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")

	config, err := Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.HTTPAddr != ":8080" {
		t.Errorf("expected address %q, got %q", ":8080", config.HTTPAddr)
	}
}

func TestLoadUsesConfiguredPort(t *testing.T) {
	t.Setenv("PORT", "9090")

	config, err := Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.HTTPAddr != ":9090" {
		t.Errorf("expected address %q, got %q", ":9090", config.HTTPAddr)
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
