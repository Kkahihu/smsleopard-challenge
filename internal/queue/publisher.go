package queue

import (
	"encoding/json"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher publishes message jobs to RabbitMQ
type Publisher struct {
	conn      *Connection
	queueName string
}

// MessageJob represents a message job to be processed
type MessageJob struct {
	MessageID  int `json:"message_id"`
	CampaignID int `json:"campaign_id"`
	CustomerID int `json:"customer_id"`
}

// NewPublisher creates a new publisher instance
func NewPublisher(conn *Connection, queueName string) (*Publisher, error) {
	// Validate conn is not nil
	if conn == nil {
		return nil, errors.New("connection cannot be nil")
	}

	// Validate queueName is not empty
	if queueName == "" {
		return nil, errors.New("queue name cannot be empty")
	}

	// Get channel from connection
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Declare queue (durable, non-auto-delete, non-exclusive)
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Return Publisher instance
	return &Publisher{
		conn:      conn,
		queueName: queueName,
	}, nil
}

// PublishMessage publishes a message job to the queue
func (p *Publisher) PublishMessage(messageID, campaignID, customerID int) error {
	// Create MessageJob struct with provided IDs
	job := MessageJob{
		MessageID:  messageID,
		CampaignID: campaignID,
		CustomerID: customerID,
	}

	// Marshal to JSON
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal message job: %w", err)
	}

	// Get channel from connection
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Publish message
	err = ch.Publish(
		"",          // exchange (default)
		p.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // 2 - persistent
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Close closes the publisher (no-op, connection managed externally)
func (p *Publisher) Close() error {
	// Connection is closed separately
	return nil
}
