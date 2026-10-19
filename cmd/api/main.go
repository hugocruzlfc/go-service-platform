package main

import (
	"log"
	"net/http"
	"time"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/config"
	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	repository := memory.NewTaskRepository()
	service := application.NewTaskService(repository)
	router := httpapi.NewRouter(service)

	server := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on %s", config.HTTPAddr)

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
