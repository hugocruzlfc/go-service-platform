package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

const refreshSecretSize = 32

var _ application.RefreshTokenManager = (*RefreshTokenManager)(nil)

// RefreshTokenManager creates opaque refresh tokens and their hashes.
type RefreshTokenManager struct{}

// Issue creates a refresh token associated with sessionID and its SHA-256 hash.
func (RefreshTokenManager) Issue(sessionID string) (string, string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", "", domain.ErrSessionIDRequired
	}

	var secret [refreshSecretSize]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	token := sessionID + "." + base64.RawURLEncoding.EncodeToString(secret[:])
	return token, hashRefreshToken(token), nil
}

// Parse validates the token shape and returns its session ID and hash.
func (RefreshTokenManager) Parse(token string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] == "" {
		return "", "", domain.ErrRefreshTokenInvalid
	}

	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != refreshSecretSize {
		return "", "", domain.ErrRefreshTokenInvalid
	}

	return parts[0], hashRefreshToken(token), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
