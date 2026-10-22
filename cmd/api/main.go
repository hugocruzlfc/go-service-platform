// Command api starts the task HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/auth"
	"service-platform/internal/config"
	"service-platform/internal/task"
	"service-platform/internal/task/infrastructure/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(context.Background(), logger); err != nil {
		logger.Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	// Local development only: a missing .env file is fine (production sets real env vars).
	// Variables already set in the shell take precedence over the file.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(ctx, 5*time.Second)
	defer cancelDatabase()

	databasePool, err := postgres.NewPool(databaseContext, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer databasePool.Close()

	readiness := httpapi.NewReadiness(databasePool, logger)
	taskHTTPHandler := task.NewHTTPHandler(databasePool, readiness, logger)
	authenticationHTTPHandler, accessTokenVerifier, err := auth.NewHTTPHandler(
		databasePool,
		cfg.AuthJWTPrivateKeyBase64,
		cfg.AuthJWTIssuer,
		cfg.AuthJWTAudience,
		cfg.AuthAccessTokenTTL,
		logger,
	)
	if err != nil {
		return fmt.Errorf("configure authentication: %w", err)
	}
	router := httpapi.NewRouter(taskHTTPHandler, authenticationHTTPHandler, accessTokenVerifier, logger)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("API listening", slog.String("address", cfg.HTTPAddr))
		serverErrors <- server.ListenAndServe()
	}()

	stopSignals, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		return nil
	case <-stopSignals.Done():
		logger.Info("shutdown signal received")
		readiness.BeginDraining()
		logger.Info("service is draining")

		drainTimer := time.NewTimer(time.Second)
		<-drainTimer.C

		shutdownContext, cancel := context.WithTimeout(
			context.WithoutCancel(stopSignals),
			5*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP after shutdown: %w", err)
		}

		logger.Info("API stopped gracefully")
		return nil
	}
}
