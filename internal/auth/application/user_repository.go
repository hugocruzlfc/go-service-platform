package application

import (
	"context"

	"service-platform/internal/auth/domain"
)

// UserRepository persists and retrieves user identities.
type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}
