package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSON escreve v como JSON com o status informado.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
