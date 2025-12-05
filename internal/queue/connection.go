package queue

import (
	"errors"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection represents a RabbitMQ connection with automatic reconnection support
type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
	mu      sync.Mutex
}

// NewConnection creates a new RabbitMQ connection
func NewConnection(url string) (*Connection, error) {
	// Validate URL is not empty
	if url == "" {
		return nil, errors.New("rabbitmq url cannot be empty")
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Create and return Connection instance
	c := &Connection{
		conn:    conn,
		channel: channel,
		url:     url,
	}

	log.Println("Successfully connected to RabbitMQ")
	return c, nil
}

// Channel returns the channel, reconnecting if necessary
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if channel is nil or closed
	if c.channel == nil || c.conn == nil || c.conn.IsClosed() {
		log.Println("Channel is closed, attempting to reconnect...")
		if err := c.reconnect(); err != nil {
			return nil, fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	return c.channel, nil
}

// reconnect is an internal method to reconnect to RabbitMQ
func (c *Connection) reconnect() error {
	// Close existing connection/channel if any
	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Dial RabbitMQ with stored URL
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to reconnect to rabbitmq: %w", err)
	}

	// Create new channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create channel on reconnect: %w", err)
	}

	// Update conn and channel fields
	c.conn = conn
	c.channel = channel

	// Log reconnection event
	log.Println("Successfully reconnected to RabbitMQ")
	return nil
}

// Close closes the connection gracefully
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	// Close channel if not nil
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
		c.channel = nil
	}

	// Close connection if not nil
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
		c.conn = nil
	}

	// Return any errors
	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	log.Println("RabbitMQ connection closed successfully")
	return nil
}

// IsConnected checks if the connection is active
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if conn is not nil
	if c.conn == nil {
		return false
	}

	// Check if conn.IsClosed() is false
	if c.conn.IsClosed() {
		return false
	}

	// Check if channel is not nil
	if c.channel == nil {
		return false
	}

	// Return true only if all checks pass
	return true
}
