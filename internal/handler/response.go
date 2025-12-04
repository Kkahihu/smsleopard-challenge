package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"smsleopard/internal/service"
)

// ErrorResponse represents the standard error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error code and message
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes a JSON response with the given status code
// It sets the Content-Type header, writes the status code, and encodes the data to JSON
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return nil
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: Failed to encode JSON response: %v", err)
		return err
	}

	return nil
}

// WriteError writes a structured JSON error response
// It creates an ErrorResponse with the given code and message
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		log.Printf("ERROR: Failed to write error response: %v", err)
	}
}

// WriteCreated writes a 201 Created response with the given data
func WriteCreated(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusCreated, data)
}

// WriteOK writes a 200 OK response with the given data
func WriteOK(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusOK, data)
}

// WriteNoContent writes a 204 No Content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteValidationError writes a 400 Bad Request response with VALIDATION_ERROR code
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

// WriteNotFoundError writes a 404 Not Found response with RESOURCE_NOT_FOUND code
func WriteNotFoundError(w http.ResponseWriter, resource string, id int) {
	message := fmt.Sprintf("%s with ID %d not found", resource, id)
	WriteError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", message)
}

// WriteInternalError writes a 500 Internal Server Error response with INTERNAL_ERROR code
// It logs the error but doesn't expose internal details to the client
func WriteInternalError(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
}

// WriteBusinessLogicError writes a 400 Bad Request response with BUSINESS_LOGIC_ERROR code
func WriteBusinessLogicError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "BUSINESS_LOGIC_ERROR", message)
}

// WriteConflictError writes a 409 Conflict response with CONFLICT code
func WriteConflictError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, "CONFLICT", message)
}

// HandleServiceError maps service layer errors to appropriate HTTP responses
// It uses type assertions to determine the error type and calls the appropriate write function
func HandleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *service.NotFoundError:
		WriteNotFoundError(w, e.Resource, e.ID)
	case *service.ValidationError:
		WriteValidationError(w, e.Message)
	case *service.BusinessLogicError:
		WriteBusinessLogicError(w, e.Message)
	case *service.ConflictError:
		WriteConflictError(w, e.Message)
	default:
		// Log the actual error for debugging
		log.Printf("ERROR: Unhandled service error: %v", err)
		WriteInternalError(w)
	}
}
