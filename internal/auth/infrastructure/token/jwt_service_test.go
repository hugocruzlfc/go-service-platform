package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

func newTestService(t *testing.T, accessTTL time.Duration) *Service {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating Ed25519 key: %v", err)
	}

	service, err := NewService(privateKey, "service-platform", "service-platform-api", accessTTL)
	if err != nil {
		t.Fatalf("creating token service: %v", err)
	}

	return service
}

func TestServiceIssueAndParseAccessToken(t *testing.T) {
	service := newTestService(t, time.Minute)

	encodedToken, err := service.IssueAccessToken(application.AccessTokenInput{
		UserID:      "user-1",
		SessionID:   "session-1",
		Roles:       []string{"user"},
		Permissions: []string{"tasks:read"},
	})
	if err != nil {
		t.Fatalf("issuing access token: %v", err)
	}

	claims, err := service.ParseAccessToken(encodedToken)
	if err != nil {
		t.Fatalf("parsing access token: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected identity claims: %#v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("unexpected roles: %#v", claims.Roles)
	}
	if len(claims.Permissions) != 1 || claims.Permissions[0] != "tasks:read" {
		t.Fatalf("unexpected permissions: %#v", claims.Permissions)
	}
}

func TestServiceRejectsExpiredAccessToken(t *testing.T) {
	service := newTestService(t, time.Minute)
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "service-platform",
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{"service-platform-api"},
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			ID:        "expired-token",
		},
		SessionID: "session-1",
	})
	encodedToken, err := token.SignedString(service.privateKey)
	if err != nil {
		t.Fatalf("issuing expired access token: %v", err)
	}

	if _, err := service.ParseAccessToken(encodedToken); !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Errorf("expected ErrAccessTokenInvalid, got %v", err)
	}
}

func TestServiceRejectsTokenSignedByDifferentKey(t *testing.T) {
	service := newTestService(t, time.Minute)
	_, otherPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating second Ed25519 key: %v", err)
	}

	token := jwtTokenWithKey(t, otherPrivateKey)
	if _, err := service.ParseAccessToken(token); !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Errorf("expected ErrAccessTokenInvalid, got %v", err)
	}
}

func TestServiceRejectsMissingTokenIdentity(t *testing.T) {
	service := newTestService(t, time.Minute)

	if _, err := service.IssueAccessToken(application.AccessTokenInput{SessionID: "session-1"}); !errors.Is(err, domain.ErrTokenSubjectRequired) {
		t.Errorf("expected ErrTokenSubjectRequired, got %v", err)
	}
	if _, err := service.IssueAccessToken(application.AccessTokenInput{UserID: "user-1"}); !errors.Is(err, domain.ErrTokenSessionRequired) {
		t.Errorf("expected ErrTokenSessionRequired, got %v", err)
	}
}

func jwtTokenWithKey(t *testing.T, privateKey ed25519.PrivateKey) string {
	t.Helper()

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "service-platform",
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{"service-platform-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "token-1",
		},
		SessionID: "session-1",
	})

	encodedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token with second key: %v", err)
	}

	return encodedToken
}
