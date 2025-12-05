package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer consumes messages from RabbitMQ queue
type Consumer struct {
	conn      *Connection
	queueName string
	handler   MessageHandler
	stopChan  chan struct{}
	doneChan  chan struct{}
}

// MessageHandler is a function that processes a message
type MessageHandler func(job *MessageJob) error

// NewConsumer creates a new consumer instance
func NewConsumer(conn *Connection, queueName string, handler MessageHandler) (*Consumer, error) {
	// Validate conn is not nil
	if conn == nil {
		return nil, errors.New("connection cannot be nil")
	}

	// Validate queueName is not empty
	if queueName == "" {
		return nil, errors.New("queue name cannot be empty")
	}

	// Validate handler is not nil
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	// Get channel from connection
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Declare queue (same settings as publisher: durable, non-auto-delete)
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

	// Create stop and done channels
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})

	// Return Consumer instance
	return &Consumer{
		conn:      conn,
		queueName: queueName,
		handler:   handler,
		stopChan:  stopChan,
		doneChan:  doneChan,
	}, nil
}

// Start starts consuming messages from the queue
func (c *Consumer) Start() error {
	// Get channel from connection
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Set QoS (prefetch count: 1, to process one message at a time)
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		c.queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (manual acknowledgement)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	// Process messages in goroutine
	go func() {
		defer close(c.doneChan)

		for {
			select {
			case <-c.stopChan:
				log.Println("Consumer stopping...")
				return
			case d, ok := <-msgs:
				if !ok {
					log.Println("Delivery channel closed")
					return
				}

				// Process message
				err := c.processMessage(d)
				if err != nil {
					log.Printf("Error processing message: %v", err)
					// Requeue for retry
					// In Phase 5.4, we'll add retry count checking
					d.Nack(false, true)
				} else {
					// Acknowledge successful processing
					d.Ack(false)
				}
			}
		}
	}()

	log.Printf("Consumer started, listening on queue: %s", c.queueName)
	return nil
}

// Stop stops consuming messages gracefully
func (c *Consumer) Stop() error {
	// Send signal to stopChan
	close(c.stopChan)

	// Wait for doneChan
	<-c.doneChan

	log.Println("Consumer stopped successfully")
	return nil
}

// processMessage processes a single message
func (c *Consumer) processMessage(d amqp.Delivery) error {
	// Parse JSON body into MessageJob
	var job MessageJob
	err := json.Unmarshal(d.Body, &job)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message job: %w", err)
	}

	// Call handler with MessageJob
	err = c.handler(&job)
	if err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	return nil
}
