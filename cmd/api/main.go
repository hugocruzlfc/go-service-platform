package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type VersionResponse struct {
    Version string `json:"version"`
}


func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    response := HealthResponse{
        Status: "ok",
        Time:   time.Now().UTC().Format(time.RFC3339),
    }

    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("encoding health response: %v", err)
    }
}





func main() {
	mux := http.NewServeMux()

	//health
	mux.HandleFunc("GET /health", healthHandler);

    //version
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    response := VersionResponse{
        Version: "1.0.0",
    }

    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("encoding version response: %v", err)
    }
})


	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("API listening on :8080")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
