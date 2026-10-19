package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"service-platform/internal/config"
	"service-platform/internal/migration"
	"service-platform/internal/task/infrastructure/postgres"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Printf("migration failed: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
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

	log.Println("database migrations completed")
	return nil
}
