// Command migrate applies pending database migrations.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"service-platform/internal/config"
	"service-platform/internal/migration"
	"service-platform/internal/task/infrastructure/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(context.Background(), logger); err != nil {
		logger.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(ctx, 30*time.Second)
	defer cancelDatabase()

	databasePool, err := postgres.NewPool(databaseContext, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer databasePool.Close()

	if err := migration.Run(databaseContext, databasePool); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("database migrations completed")
	return nil
}
