package models

import "time"

// MessageStatus represents valid message statuses
type MessageStatus string

const (
	MessageStatusPending MessageStatus = "pending"
	MessageStatusSent    MessageStatus = "sent"
	MessageStatusFailed  MessageStatus = "failed"
)

// OutboundMessage represents an outbound message
type OutboundMessage struct {
	ID              int           `json:"id" db:"id"`
	CampaignID      int           `json:"campaign_id" db:"campaign_id"`
	CustomerID      int           `json:"customer_id" db:"customer_id"`
	Status          MessageStatus `json:"status" db:"status"`
	RenderedContent *string       `json:"rendered_content,omitempty" db:"rendered_content"`
	LastError       *string       `json:"last_error,omitempty" db:"last_error"`
	RetryCount      int           `json:"retry_count" db:"retry_count"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// OutboundMessageWithDetails includes campaign and customer info
type OutboundMessageWithDetails struct {
	OutboundMessage
	Campaign Campaign `json:"campaign"`
	Customer Customer `json:"customer"`
}

// CanRetry checks if message can be retried
func (m *OutboundMessage) CanRetry() bool {
	return m.Status == MessageStatusFailed && m.RetryCount < 3
}

// IncrementRetry increments the retry count
func (m *OutboundMessage) IncrementRetry() {
	m.RetryCount++
}
