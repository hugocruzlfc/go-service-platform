package main

import (
	"log"
	"net/http"
	"time"

	"go-service-platform/internal/adapter/httpapi"
	"go-service-platform/internal/task/application"
	"go-service-platform/internal/task/infrastructure/memory"
)

func main() {
	repository := memory.NewTaskRepository()
	service := application.NewTaskService(repository)
	router := httpapi.NewRouter(service)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("API listening on :8080")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
