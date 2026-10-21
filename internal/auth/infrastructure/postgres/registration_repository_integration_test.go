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
	taskpostgres "service-platform/internal/task/infrastructure/postgres"
)

func TestRegistrationRepositoryCreateAndReadIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := taskpostgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("creating database pool: %v", err)
	}

	userID := "integration-user-" + time.Now().UTC().Format("20060102150405.000000000")
	email := fmt.Sprintf("%s@example.com", userID)
	user, err := domain.NewUser(userID, email, time.Now())
	if err != nil {
		t.Fatalf("creating domain user: %v", err)
	}

	t.Cleanup(func() {
		cleanupContext, cancelCleanup := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelCleanup()

		_, cleanupErr := pool.Exec(cleanupContext, "DELETE FROM users WHERE id = $1", user.ID)
		if cleanupErr != nil {
			t.Errorf("cleaning up integration user: %v", cleanupErr)
		}

		pool.Close()
	})

	registrationRepository := NewRegistrationRepository(pool)
	if err := registrationRepository.CreateUserWithCredentials(
		ctx,
		user,
		"encoded-password-hash",
		time.Now(),
	); err != nil {
		t.Fatalf("creating registered user: %v", err)
	}

	storedUser, err := NewUserRepository(pool).GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("getting registered user: %v", err)
	}
	if storedUser.ID != user.ID || storedUser.Email != user.Email || storedUser.Disabled != user.Disabled {
		t.Fatalf("expected stored user %#v, got %#v", user, storedUser)
	}
	if !storedUser.CreatedAt.Equal(user.CreatedAt) {
		t.Fatalf("expected creation time %v, got %v", user.CreatedAt, storedUser.CreatedAt)
	}

	passwordHash, err := NewCredentialRepository(pool).GetPasswordHash(ctx, user.ID)
	if err != nil {
		t.Fatalf("getting stored password hash: %v", err)
	}
	if passwordHash != "encoded-password-hash" {
		t.Errorf("expected stored password hash, got %q", passwordHash)
	}

	if err := registrationRepository.CreateUserWithCredentials(
		ctx,
		user,
		"another-hash",
		time.Now(),
	); !errors.Is(err, domain.ErrUserEmailTaken) {
		t.Errorf("expected duplicate email error, got %v", err)
	}
}
