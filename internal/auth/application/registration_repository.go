package application

import (
	"context"
	"time"

	"service-platform/internal/auth/domain"
)

// RegistrationRepository atomically persists a user and its password hash.
type RegistrationRepository interface {
	CreateUserWithCredentials(
		ctx context.Context,
		user domain.User,
		passwordHash string,
		updatedAt time.Time,
	) error
}
