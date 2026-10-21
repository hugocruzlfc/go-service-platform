package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

var _ application.RegistrationRepository = (*RegistrationRepository)(nil)

// RegistrationRepository atomically persists users and their credentials.
type RegistrationRepository struct {
	pool *pgxpool.Pool
}

// NewRegistrationRepository creates a PostgreSQL registration repository using pool.
func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{pool: pool}
}

// CreateUserWithCredentials stores a user and password hash in one transaction.
func (r *RegistrationRepository) CreateUserWithCredentials(
	ctx context.Context,
	user domain.User,
	passwordHash string,
	updatedAt time.Time,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user registration: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertUserQuery = `
		INSERT INTO users (id, email, created_at, disabled)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := tx.Exec(
		ctx,
		insertUserQuery,
		user.ID,
		user.Email,
		user.CreatedAt,
		user.Disabled,
	); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrUserEmailTaken
		}

		return fmt.Errorf("insert registered user: %w", err)
	}

	//nolint:gosec // G101: password_hash is a SQL column; the value is parameterized.
	const insertCredentialsQuery = `
		INSERT INTO user_credentials (user_id, password_hash, updated_at)
		VALUES ($1, $2, $3)
	`

	if _, err := tx.Exec(
		ctx,
		insertCredentialsQuery,
		user.ID,
		passwordHash,
		updatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("insert registered user credentials: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user registration: %w", err)
	}

	return nil
}
