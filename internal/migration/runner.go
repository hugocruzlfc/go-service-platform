package migration

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

const createMigrationsTable = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)
`

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "sql")
	if err != nil {
		return fmt.Errorf("read migration files: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if err := applyMigration(ctx, pool, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(
	ctx context.Context,
	pool *pgxpool.Pool,
	version string,
) error {
	var applied bool

	if err := pool.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)",
		version,
	).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", version, err)
	}

	if applied {
		return nil
	}

	contents, err := fs.ReadFile(migrationFiles, "sql/"+version)
	if err != nil {
		return fmt.Errorf("read migration %q: %w", version, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", version, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("execute migration %q: %w", version, err)
	}

	if _, err := tx.Exec(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		version,
	); err != nil {
		return fmt.Errorf("record migration %q: %w", version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %q: %w", version, err)
	}

	return nil
}
