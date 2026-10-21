package application

import (
	"context"
	"time"

	"service-platform/internal/auth/domain"
)

// SessionRepository persists refresh-token sessions and their lifecycle.
type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	GetByID(ctx context.Context, id string) (domain.Session, error)
	Rotate(
		ctx context.Context,
		sessionID string,
		expectedRefreshHash string,
		rotatedSession domain.Session,
	) error
	Revoke(ctx context.Context, sessionID string, revokedAt time.Time) error
	RevokeFamily(ctx context.Context, tokenFamily string, revokedAt time.Time) error
}
