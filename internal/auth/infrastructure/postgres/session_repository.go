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

var _ application.SessionRepository = (*SessionRepository)(nil)

// SessionRepository persists authentication sessions in PostgreSQL.
type SessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a PostgreSQL session repository using pool.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Create stores a new refresh-token session.
func (r *SessionRepository) Create(ctx context.Context, session domain.Session) error {
	const query = `
		INSERT INTO auth_sessions (
			id, user_id, token_family, refresh_token_hash,
			created_at, last_used_at, expires_at, revoked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if _, err := r.pool.Exec(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.TokenFamily,
		session.RefreshHash,
		session.CreatedAt,
		session.LastUsedAt,
		session.ExpiresAt,
		session.RevokedAt,
	); err != nil {
		return fmt.Errorf("insert auth session: %w", err)
	}

	return nil
}

// GetByID returns a session by its identifier.
func (r *SessionRepository) GetByID(ctx context.Context, id string) (domain.Session, error) {
	const query = `
		SELECT id, user_id, token_family, refresh_token_hash,
		       created_at, last_used_at, expires_at, revoked_at
		FROM auth_sessions
		WHERE id = $1
	`

	var session domain.Session
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenFamily,
		&session.RefreshHash,
		&session.CreatedAt,
		&session.LastUsedAt,
		&session.ExpiresAt,
		&session.RevokedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrSessionNotFound
	} else if err != nil {
		return domain.Session{}, fmt.Errorf("get auth session: %w", err)
	}

	return session, nil
}

// Rotate replaces the current refresh hash while locking the session row.
// Reusing a previous hash revokes every active session in the same family.
func (r *SessionRepository) Rotate(
	ctx context.Context,
	sessionID string,
	expectedRefreshHash string,
	rotatedSession domain.Session,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin auth session rotation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const selectQuery = `
		SELECT token_family, refresh_token_hash, expires_at, revoked_at
		FROM auth_sessions
		WHERE id = $1
		FOR UPDATE
	`

	var (
		tokenFamily string
		currentHash string
		expiresAt   time.Time
		revokedAt   *time.Time
	)
	if err := tx.QueryRow(ctx, selectQuery, sessionID).Scan(
		&tokenFamily,
		&currentHash,
		&expiresAt,
		&revokedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSessionNotFound
	} else if err != nil {
		return fmt.Errorf("lock auth session for rotation: %w", err)
	}

	now := time.Now().UTC()
	if revokedAt != nil {
		return domain.ErrSessionRevoked
	}
	if !now.Before(expiresAt) {
		return domain.ErrSessionExpired
	}
	if currentHash != expectedRefreshHash {
		const revokeFamilyQuery = `
			UPDATE auth_sessions
			SET revoked_at = $1
			WHERE token_family = $2 AND revoked_at IS NULL
		`
		if _, err := tx.Exec(ctx, revokeFamilyQuery, now, tokenFamily); err != nil {
			return fmt.Errorf("revoke reused auth session family: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit reused auth session revocation: %w", err)
		}
		return domain.ErrRefreshTokenReused
	}

	const updateQuery = `
		UPDATE auth_sessions
		SET refresh_token_hash = $1,
		    last_used_at = $2,
		    expires_at = $3
		WHERE id = $4
	`
	if _, err := tx.Exec(
		ctx,
		updateQuery,
		rotatedSession.RefreshHash,
		rotatedSession.LastUsedAt,
		rotatedSession.ExpiresAt,
		sessionID,
	); err != nil {
		return fmt.Errorf("update rotated auth session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit auth session rotation: %w", err)
	}

	return nil
}

// Revoke marks one session as revoked.
func (r *SessionRepository) Revoke(ctx context.Context, sessionID string, revokedAt time.Time) error {
	const query = `
		UPDATE auth_sessions
		SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, revokedAt.UTC(), sessionID); err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}

	return nil
}

// RevokeFamily marks every active session in a token family as revoked.
func (r *SessionRepository) RevokeFamily(ctx context.Context, tokenFamily string, revokedAt time.Time) error {
	const query = `
		UPDATE auth_sessions
		SET revoked_at = $1
		WHERE token_family = $2 AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, revokedAt.UTC(), tokenFamily); err != nil {
		return fmt.Errorf("revoke auth session family: %w", err)
	}

	return nil
}
