package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// writeJSON writes v as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// pagination reads limit and offset from the query string, with the same rules
// the list endpoints use: limit 20 by default and capped at 100, offset 0 by
// default. Writes the 400 itself and returns ok=false when they are invalid.
func pagination(w http.ResponseWriter, r *http.Request) (limit int32, offset int32, ok bool) {
	limit = 20
	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil || v <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return 0, 0, false
		}
		if v > 100 {
			v = 100
		}
		limit = int32(v)
	}

	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return 0, 0, false
		}
		offset = int32(v)
	}
	return limit, offset, true
}
