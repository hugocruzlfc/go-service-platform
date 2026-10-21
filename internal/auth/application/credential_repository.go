package application

import (
	"context"
	"time"
)

// CredentialRepository persists password hashes separately from user identities.
type CredentialRepository interface {
	Create(ctx context.Context, userID, passwordHash string, updatedAt time.Time) error
	GetPasswordHash(ctx context.Context, userID string) (string, error)
}
