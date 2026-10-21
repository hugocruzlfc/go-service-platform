package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

var _ application.CredentialRepository = (*CredentialRepository)(nil)

// CredentialRepository persists password hashes in PostgreSQL.
type CredentialRepository struct {
	pool *pgxpool.Pool
}

// NewCredentialRepository creates a PostgreSQL credential repository using pool.
func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

// Create stores a password hash for a user.
func (r *CredentialRepository) Create(
	ctx context.Context,
	userID string,
	passwordHash string,
	updatedAt time.Time,
) error {
	const query = `
		INSERT INTO user_credentials (user_id, password_hash, updated_at)
		VALUES ($1, $2, $3)
	`

	if _, err := r.pool.Exec(
		ctx,
		query,
		userID,
		passwordHash,
		updatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("insert user credentials: %w", err)
	}

	return nil
}

// GetPasswordHash returns the stored password hash for a user.
func (r *CredentialRepository) GetPasswordHash(
	ctx context.Context,
	userID string,
) (string, error) {
	const query = `
		SELECT password_hash
		FROM user_credentials
		WHERE user_id = $1
	`

	var passwordHash string
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&passwordHash); errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrCredentialsNotFound
	} else if err != nil {
		return "", fmt.Errorf("get password hash: %w", err)
	}

	return passwordHash, nil
}
