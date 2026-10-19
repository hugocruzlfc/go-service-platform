package main

import (
	"log"
	"net/http"
	"time"

	"service-platform/internal/adapter/httpapi"
	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/memory"
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
