package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

var (
	_ application.UserRepository = (*UserRepository)(nil)
)

// UserRepository persists user identities in PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a PostgreSQL user repository using pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a user and maps duplicate emails to a domain error.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const query = `
		INSERT INTO users (id, email, created_at, disabled)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.pool.Exec(
		ctx,
		query,
		user.ID,
		user.Email,
		user.CreatedAt,
		user.Disabled,
	); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrUserEmailTaken
		}

		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// GetByID returns a user by its identifier.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	const query = `
		SELECT id, email, created_at, disabled
		FROM users
		WHERE id = $1
	`

	var user domain.User
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
		&user.Disabled,
	); errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	} else if err != nil {
		return domain.User{}, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

// GetByEmail returns a user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
		SELECT id, email, created_at, disabled
		FROM users
		WHERE lower(email) = lower($1)
	`

	var user domain.User
	if err := r.pool.QueryRow(ctx, query, strings.TrimSpace(email)).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
		&user.Disabled,
	); errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	} else if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}
