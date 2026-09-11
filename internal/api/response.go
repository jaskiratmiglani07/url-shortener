package api

import (
	"encoding/json"
	"net/http"
)

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// RespondJSON writes a JSON response with the provided status code.
func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, `{"error":{"status":500,"message":"failed to encode response"}}`, http.StatusInternalServerError)
		}
	}
}

// RespondError writes a standardized JSON error response.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorEnvelope{
		Error: ErrorDetail{
			Status:  status,
			Message: message,
		},
	})
}
