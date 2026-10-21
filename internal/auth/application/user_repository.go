package application

import (
	"context"
	"time"

	"service-platform/internal/auth/domain"
)

// UserRepository persists and retrieves user identities.
type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

// CredentialRepository persists password hashes separately from user identities.
type CredentialRepository interface {
	Create(ctx context.Context, userID, passwordHash string, updatedAt time.Time) error
	GetPasswordHash(ctx context.Context, userID string) (string, error)
}
