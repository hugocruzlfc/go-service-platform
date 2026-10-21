package token

import (
	"errors"
	"strings"
	"testing"

	"service-platform/internal/auth/domain"
)

func TestRefreshTokenManagerIssuesAndParsesToken(t *testing.T) {
	manager := RefreshTokenManager{}

	token, expectedHash, err := manager.Issue("session-1")
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	sessionID, actualHash, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if sessionID != "session-1" {
		t.Fatalf("session ID = %q, want %q", sessionID, "session-1")
	}
	if actualHash != expectedHash {
		t.Fatalf("hash = %q, want %q", actualHash, expectedHash)
	}
	if len(expectedHash) != 64 {
		t.Fatalf("hash length = %d, want 64 hexadecimal characters", len(expectedHash))
	}
}

func TestRefreshTokenManagerIssuesUniqueTokens(t *testing.T) {
	manager := RefreshTokenManager{}

	first, _, err := manager.Issue("session-1")
	if err != nil {
		t.Fatalf("issue first token: %v", err)
	}
	second, _, err := manager.Issue("session-1")
	if err != nil {
		t.Fatalf("issue second token: %v", err)
	}
	if first == second {
		t.Fatal("expected refresh tokens to be unique")
	}
}

func TestRefreshTokenManagerRejectsInvalidTokens(t *testing.T) {
	manager := RefreshTokenManager{}
	token, _, err := manager.Issue("session-1")
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	tests := []string{"", "session-1", "session-1.invalid", token + ".extra", strings.Replace(token, "session-1", "", 1)}
	for _, testToken := range tests {
		t.Run(testToken, func(t *testing.T) {
			_, _, err := manager.Parse(testToken)
			if !errors.Is(err, domain.ErrRefreshTokenInvalid) {
				t.Fatalf("error = %v, want ErrRefreshTokenInvalid", err)
			}
		})
	}
}

func TestRefreshTokenManagerRequiresSessionID(t *testing.T) {
	_, _, err := (RefreshTokenManager{}).Issue(" ")
	if !errors.Is(err, domain.ErrSessionIDRequired) {
		t.Fatalf("error = %v, want ErrSessionIDRequired", err)
	}
}
