//go:build integration

package migration

import (
	"context"
	"os"
	"testing"
	"time"

	"service-platform/internal/task/infrastructure/postgres"
)

func TestRunRecordsAppliedMigrations(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("creating database pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := Run(ctx, pool); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	var appliedCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
		"001_create_tasks.sql",
	).Scan(&appliedCount); err != nil {
		t.Fatalf("checking migration record: %v", err)
	}

	if appliedCount != 1 {
		t.Fatalf("expected migration to be recorded once, got %d records", appliedCount)
	}
}
