package service

import (
	"context"
	"database/sql"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Health status constants
const (
	StatusHealthy      = "healthy"
	StatusDegraded     = "degraded"
	StatusUnhealthy    = "unhealthy"
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
)

// HealthStatus represents the overall health status of the application
type HealthStatus struct {
	Status    string            `json:"status"`
	Services  map[string]string `json:"services"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
}

// HealthChecker handles health check operations
type HealthChecker struct {
	db       *sql.DB
	queueURL string
	version  string
}

// NewHealthService creates a new HealthChecker instance
func NewHealthService(db *sql.DB, queueURL, version string) *HealthChecker {
	return &HealthChecker{
		db:       db,
		queueURL: queueURL,
		version:  version,
	}
}

// checkDatabase verifies PostgreSQL connectivity with a timeout
func (h *HealthChecker) checkDatabase() string {
	// Create context with 2-second timeout for database ping
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Attempt to ping the database
	if err := h.db.PingContext(ctx); err != nil {
		return StatusDisconnected
	}

	return StatusConnected
}

// checkQueue verifies RabbitMQ connectivity
func (h *HealthChecker) checkQueue() string {
	// Attempt to establish connection to RabbitMQ
	conn, err := amqp.Dial(h.queueURL)
	if err != nil {
		return StatusDisconnected
	}

	// Close connection immediately after successful connection test
	defer conn.Close()

	return StatusConnected
}

// determineOverallStatus calculates the overall health status based on service statuses
func (h *HealthChecker) determineOverallStatus(services map[string]string) string {
	databaseStatus := services["database"]
	queueStatus := services["queue"]

	// If database is disconnected, system is unhealthy
	if databaseStatus == StatusDisconnected {
		return StatusUnhealthy
	}

	// If queue is disconnected but database is connected, system is degraded
	if queueStatus == StatusDisconnected {
		return StatusDegraded
	}

	// All services connected, system is healthy
	return StatusHealthy
}

// CheckHealth performs health checks on all dependencies and returns the overall status
func (h *HealthChecker) CheckHealth() (*HealthStatus, error) {
	// Check individual services
	services := map[string]string{
		"database": h.checkDatabase(),
		"queue":    h.checkQueue(),
	}

	// Determine overall system health
	overallStatus := h.determineOverallStatus(services)

	// Build and return health status
	healthStatus := &HealthStatus{
		Status:    overallStatus,
		Services:  services,
		Timestamp: time.Now().UTC(),
		Version:   h.version,
	}

	return healthStatus, nil
}
