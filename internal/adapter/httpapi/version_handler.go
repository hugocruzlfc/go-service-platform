package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

// VersionResponse is the response returned by the version endpoint.
type VersionResponse struct {
	Version string `json:"version"`
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := VersionResponse{
		Version: "1.0.0",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encoding version response: %v", err)
	}
}
