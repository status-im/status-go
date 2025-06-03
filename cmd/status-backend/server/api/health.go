package api

import (
	"encoding/json"
	"net/http"

	"github.com/status-im/status-go/pkg/version"
)

type HealthResponse struct {
	Version string `json:"version,omitempty"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Version: version.Version(),
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
