package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/config"
	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancelDatabase()

	databasePool, err := pgxpool.New(databaseContext, config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer databasePool.Close()

	if err := databasePool.Ping(databaseContext); err != nil {
		log.Fatal(err)
	}

	repository := postgres.NewTaskRepository(databasePool)
	service := application.NewTaskService(repository)
	router := httpapi.NewRouter(service)

	server := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("API listening on %s", config.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()

	stopSignals, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-stopSignals.Done():
		log.Println("shutdown signal received")

		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownContext); err != nil {
			log.Fatal(err)
		}

		log.Println("API stopped gracefully")
	}
}
