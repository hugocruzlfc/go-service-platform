package application

import (
	"errors"
	"strings"
	"testing"
	"time"

	"service-platform/internal/auth/domain"
)

func TestSessionServiceRefreshRotatesSessionAndIssuesTokens(t *testing.T) {
	now := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	session := domain.Session{
		ID:          "session-1",
		UserID:      "user-1",
		TokenFamily: "family-1",
		RefreshHash: "refresh-hash",
		CreatedAt:   now.Add(-time.Hour),
		LastUsedAt:  now.Add(-time.Hour),
		ExpiresAt:   now.Add(6 * 24 * time.Hour),
	}
	sessions := &fakeSessionRepository{session: session}
	accessTokens := &fakeAccessTokenService{}
	service := NewSessionService(sessions, testRefreshTokenManager{}, accessTokens)
	service.now = func() time.Time { return now }

	result, err := service.Refresh(t.Context(), "session-1.old-secret")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if result.AccessToken != "access-token" {
		t.Fatalf("access token = %q", result.AccessToken)
	}
	if result.RefreshToken != "session-1.new-secret" {
		t.Fatalf("refresh token = %q", result.RefreshToken)
	}
	if !sessions.session.LastUsedAt.Equal(now) {
		t.Fatalf("last used at = %v, want %v", sessions.session.LastUsedAt, now)
	}
	if !sessions.session.ExpiresAt.Equal(now.Add(refreshTokenInactivity)) {
		t.Fatalf("expires at = %v, want %v", sessions.session.ExpiresAt, now.Add(refreshTokenInactivity))
	}
	if accessTokens.input.UserID != session.UserID || accessTokens.input.SessionID != session.ID {
		t.Fatalf("access token input = %#v", accessTokens.input)
	}
}

func TestSessionServiceMapsRefreshTokenReuseToInvalidToken(t *testing.T) {
	now := time.Now().UTC()
	sessions := &fakeSessionRepository{
		session: domain.Session{
			ID:        "session-1",
			UserID:    "user-1",
			CreatedAt: now.Add(-time.Hour),
			ExpiresAt: now.Add(time.Hour),
		},
		rotateErr: domain.ErrRefreshTokenReused,
	}
	service := NewSessionService(sessions, testRefreshTokenManager{}, &fakeAccessTokenService{})

	_, err := service.Refresh(t.Context(), "session-1.old-secret")
	if !errors.Is(err, domain.ErrRefreshTokenInvalid) {
		t.Fatalf("error = %v, want ErrRefreshTokenInvalid", err)
	}
}

func TestSessionServiceLogoutRevokesSession(t *testing.T) {
	sessions := &fakeSessionRepository{}
	service := NewSessionService(sessions, testRefreshTokenManager{}, &fakeAccessTokenService{})

	if err := service.Logout(t.Context(), "session-1.old-secret"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if sessions.revokedAt.IsZero() {
		t.Fatal("expected session to be revoked")
	}
}

type testRefreshTokenManager struct{}

func (testRefreshTokenManager) Issue(sessionID string) (string, string, error) {
	return sessionID + ".new-secret", "new-refresh-hash", nil
}

func (testRefreshTokenManager) Parse(refreshToken string) (string, string, error) {
	if !strings.HasSuffix(refreshToken, ".old-secret") {
		return "", "", domain.ErrRefreshTokenInvalid
	}
	return strings.TrimSuffix(refreshToken, ".old-secret"), "refresh-hash", nil
}
