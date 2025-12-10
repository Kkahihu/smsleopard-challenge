package handler

import (
	"encoding/json"
	"net/http"

	"smsleopard/internal/service"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthService *service.HealthChecker
}

// NewHealthHandler creates a new HealthHandler instance
func NewHealthHandler(healthService *service.HealthChecker) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

// HandleHealth handles GET requests to the /health endpoint
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Perform health check
	healthStatus, err := h.healthService.CheckHealth()
	if err != nil {
		// Handle health check error with 500 status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to perform health check",
		})
		return
	}

	// Set Content-Type header
	w.Header().Set("Content-Type", "application/json")

	// Determine HTTP status code based on health status
	switch healthStatus.Status {
	case service.StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case service.StatusDegraded, service.StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	// Encode and send health status response
	if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
		// If encoding fails, log it but response is already sent
		// Nothing we can do at this point
		return
	}
}
