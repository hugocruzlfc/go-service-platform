package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"service-platform/internal/auth/domain"
)

const maxSessionLifetime = 30 * 24 * time.Hour

// SessionService refreshes and revokes authenticated sessions.
type SessionService struct {
	sessions      SessionRepository
	refreshTokens RefreshTokenManager
	accessTokens  TokenService
	now           func() time.Time
}

// RefreshResult contains the tokens issued after a successful refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// NewSessionService creates a service for refresh and logout operations.
func NewSessionService(
	sessions SessionRepository,
	refreshTokens RefreshTokenManager,
	accessTokens TokenService,
) *SessionService {
	return &SessionService{
		sessions:      sessions,
		refreshTokens: refreshTokens,
		accessTokens:  accessTokens,
		now:           time.Now,
	}
}

// Refresh rotates the current refresh token and issues a new access token.
func (s *SessionService) Refresh(ctx context.Context, refreshToken string) (RefreshResult, error) {
	sessionID, expectedHash, err := s.refreshTokens.Parse(refreshToken)
	if err != nil {
		return RefreshResult{}, domain.ErrRefreshTokenInvalid
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return RefreshResult{}, domain.ErrRefreshTokenInvalid
		}
		return RefreshResult{}, fmt.Errorf("get session for refresh: %w", err)
	}

	now := s.now().UTC()
	if session.IsRevoked() || session.IsExpired(now) {
		return RefreshResult{}, domain.ErrRefreshTokenInvalid
	}

	newRefreshToken, newRefreshHash, err := s.refreshTokens.Issue(session.ID)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("issue rotated refresh token: %w", err)
	}

	expiresAt := now.Add(refreshTokenInactivity)
	maximumExpiresAt := session.CreatedAt.Add(maxSessionLifetime)
	if expiresAt.After(maximumExpiresAt) {
		expiresAt = maximumExpiresAt
	}
	if !expiresAt.After(now) {
		return RefreshResult{}, domain.ErrRefreshTokenInvalid
	}

	rotatedSession := session
	rotatedSession.RefreshHash = newRefreshHash
	rotatedSession.LastUsedAt = now
	rotatedSession.ExpiresAt = expiresAt

	accessToken, err := s.accessTokens.IssueAccessToken(AccessTokenInput{
		UserID:    session.UserID,
		SessionID: session.ID,
	})
	if err != nil {
		return RefreshResult{}, fmt.Errorf("issue refreshed access token: %w", err)
	}

	if err := s.sessions.Rotate(ctx, session.ID, expectedHash, rotatedSession); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenReused) ||
			errors.Is(err, domain.ErrSessionExpired) ||
			errors.Is(err, domain.ErrSessionRevoked) ||
			errors.Is(err, domain.ErrSessionNotFound) {
			return RefreshResult{}, domain.ErrRefreshTokenInvalid
		}
		return RefreshResult{}, fmt.Errorf("rotate session: %w", err)
	}

	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout revokes the session associated with a refresh token.
func (s *SessionService) Logout(ctx context.Context, refreshToken string) error {
	sessionID, _, err := s.refreshTokens.Parse(refreshToken)
	if err != nil {
		return domain.ErrRefreshTokenInvalid
	}

	if err := s.sessions.Revoke(ctx, sessionID, s.now().UTC()); err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.ErrRefreshTokenInvalid
		}
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
