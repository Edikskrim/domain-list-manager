package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a unified API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a unified API success response.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// WriteError writes a JSON error response with the given HTTP status code.
func WriteError(w http.ResponseWriter, message string, statusCode int, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Details: details,
		Code:    statusCode,
	})
}

// WriteSuccess writes a JSON success response.
func WriteSuccess(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// WriteCreated writes a JSON success response with HTTP 201 status.
func WriteCreated(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// WriteBadRequest writes a 400 error response.
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, message, http.StatusBadRequest, "")
}

// WriteNotFound writes a 404 error response.
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, message, http.StatusNotFound, "")
}

// WriteInternalServerError writes a 500 error response.
func WriteInternalServerError(w http.ResponseWriter, message string) {
	WriteError(w, message, http.StatusInternalServerError, "")
}

// WriteMethodNotAllowed writes a 405 error response.
func WriteMethodNotAllowed(w http.ResponseWriter, method string) {
	WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "Expected "+method)
}

// RequireMethod returns a middleware that checks the HTTP method.
func RequireMethod(method string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error:  "Method not allowed",
					Details: "Expected " + method,
					Code:   http.StatusMethodNotAllowed,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
