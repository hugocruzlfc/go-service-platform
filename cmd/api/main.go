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

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/config"
	"service-platform/internal/task/application"
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
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancelDatabase()

	databasePool, err := postgres.NewPool(databaseContext, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer databasePool.Close()

	repository := postgres.NewTaskRepository(databasePool)
	service := application.NewTaskService(repository)
	router := httpapi.NewRouter(service, databasePool, logger)

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
