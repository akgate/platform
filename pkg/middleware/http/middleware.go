package http

import (
	"encoding/json"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode json", http.StatusInternalServerError)
	}
}
