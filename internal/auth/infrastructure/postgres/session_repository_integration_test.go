//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"service-platform/internal/auth/domain"
	"service-platform/internal/auth/infrastructure/token"
	taskpostgres "service-platform/internal/task/infrastructure/postgres"
)

func TestSessionRepositoryRotatesAndRevokesReusedTokenFamily(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := taskpostgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	userID := "session-test-user-" + suffix
	tokenFamily := "session-test-family-" + suffix
	sessionID := "session-test-" + suffix
	email := "session-test-" + suffix + "@example.com"

	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, created_at, disabled)
		VALUES ($1, $2, $3, FALSE)
	`, userID, email, time.Now().UTC())
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}()

	manager := token.RefreshTokenManager{}
	firstToken, firstHash, err := manager.Issue(sessionID)
	if err != nil {
		t.Fatalf("issue first refresh token: %v", err)
	}
	firstCreatedAt := time.Now().UTC().Add(-time.Minute)
	firstSession, err := domain.NewSession(
		sessionID,
		userID,
		tokenFamily,
		firstHash,
		firstCreatedAt,
		firstCreatedAt.Add(7*24*time.Hour),
	)
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}

	repository := NewSessionRepository(pool)
	if err := repository.Create(ctx, firstSession); err != nil {
		t.Fatalf("create session: %v", err)
	}

	stored, err := repository.GetByID(ctx, sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if stored.RefreshHash != firstHash {
		t.Fatalf("stored refresh hash = %q, want %q", stored.RefreshHash, firstHash)
	}

	secondToken, secondHash, err := manager.Issue(sessionID)
	if err != nil {
		t.Fatalf("issue second refresh token: %v", err)
	}
	if secondToken == firstToken {
		t.Fatal("expected rotated refresh token to be different")
	}
	rotated := firstSession
	rotated.RefreshHash = secondHash
	rotated.LastUsedAt = time.Now().UTC()
	if err := repository.Rotate(ctx, sessionID, firstHash, rotated); err != nil {
		t.Fatalf("rotate session: %v", err)
	}

	stored, err = repository.GetByID(ctx, sessionID)
	if err != nil {
		t.Fatalf("get rotated session: %v", err)
	}
	if stored.RefreshHash != secondHash {
		t.Fatalf("rotated refresh hash = %q, want %q", stored.RefreshHash, secondHash)
	}

	if err := repository.Rotate(ctx, sessionID, firstHash, rotated); !errors.Is(err, domain.ErrRefreshTokenReused) {
		t.Fatalf("reuse error = %v, want ErrRefreshTokenReused", err)
	}

	stored, err = repository.GetByID(ctx, sessionID)
	if err != nil {
		t.Fatalf("get revoked session: %v", err)
	}
	if !stored.IsRevoked() {
		t.Fatal("expected reused token to revoke the session family")
	}
}
