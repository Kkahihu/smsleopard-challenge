package repository

import (
	"context"
	"database/sql"
	"fmt"

	"smsleopard/internal/models"
)

type messageRepository struct {
	db *sql.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sql.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Create creates a new outbound message
func (r *messageRepository) Create(ctx context.Context, message *models.OutboundMessage) error {
	query := `
		INSERT INTO outbound_messages (campaign_id, customer_id, status, rendered_content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		message.CampaignID,
		message.CustomerID,
		message.Status,
		message.RenderedContent,
	).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// CreateBatch creates multiple outbound messages
func (r *messageRepository) CreateBatch(ctx context.Context, messages []*models.OutboundMessage) error {
	if len(messages) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO outbound_messages (campaign_id, customer_id, status, rendered_content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, message := range messages {
		err := stmt.QueryRowContext(
			ctx,
			message.CampaignID,
			message.CustomerID,
			message.Status,
			message.RenderedContent,
		).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to create message: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a message by ID
func (r *messageRepository) GetByID(ctx context.Context, id int) (*models.OutboundMessage, error) {
	query := `
		SELECT id, campaign_id, customer_id, status, rendered_content, last_error, retry_count, created_at, updated_at
		FROM outbound_messages
		WHERE id = $1
	`

	message := &models.OutboundMessage{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&message.ID,
		&message.CampaignID,
		&message.CustomerID,
		&message.Status,
		&message.RenderedContent,
		&message.LastError,
		&message.RetryCount,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

// GetWithDetails retrieves a message with campaign and customer details
func (r *messageRepository) GetWithDetails(ctx context.Context, id int) (*models.OutboundMessageWithDetails, error) {
	query := `
		SELECT 
			m.id, m.campaign_id, m.customer_id, m.status, m.rendered_content, m.last_error, m.retry_count, m.created_at, m.updated_at,
			c.id, c.name, c.channel, c.status, c.base_template, c.scheduled_at, c.created_at, c.updated_at,
			cu.id, cu.phone, cu.first_name, cu.last_name, cu.location, cu.preferred_product, cu.created_at
		FROM outbound_messages m
		JOIN campaigns c ON m.campaign_id = c.id
		JOIN customers cu ON m.customer_id = cu.id
		WHERE m.id = $1
	`

	result := &models.OutboundMessageWithDetails{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&result.ID,
		&result.CampaignID,
		&result.CustomerID,
		&result.Status,
		&result.RenderedContent,
		&result.LastError,
		&result.RetryCount,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.Campaign.ID,
		&result.Campaign.Name,
		&result.Campaign.Channel,
		&result.Campaign.Status,
		&result.Campaign.BaseTemplate,
		&result.Campaign.ScheduledAt,
		&result.Campaign.CreatedAt,
		&result.Campaign.UpdatedAt,
		&result.Customer.ID,
		&result.Customer.Phone,
		&result.Customer.FirstName,
		&result.Customer.LastName,
		&result.Customer.Location,
		&result.Customer.PreferredProduct,
		&result.Customer.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message with details: %w", err)
	}

	return result, nil
}

// UpdateStatus updates message status and error
func (r *messageRepository) UpdateStatus(ctx context.Context, id int, status models.MessageStatus, lastError *string) error {
	query := `
		UPDATE outbound_messages
		SET status = $1, last_error = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, status, lastError, id)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// GetPendingMessages retrieves pending messages for processing
func (r *messageRepository) GetPendingMessages(ctx context.Context, limit int) ([]*models.OutboundMessage, error) {
	query := `
		SELECT id, campaign_id, customer_id, status, rendered_content, last_error, retry_count, created_at, updated_at
		FROM outbound_messages
		WHERE status = 'pending' AND retry_count < 3
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	defer rows.Close()

	messages := []*models.OutboundMessage{}
	for rows.Next() {
		message := &models.OutboundMessage{}
		err := rows.Scan(
			&message.ID,
			&message.CampaignID,
			&message.CustomerID,
			&message.Status,
			&message.RenderedContent,
			&message.LastError,
			&message.RetryCount,
			&message.CreatedAt,
			&message.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// GetByCampaignID retrieves all messages for a campaign
func (r *messageRepository) GetByCampaignID(ctx context.Context, campaignID int) ([]*models.OutboundMessage, error) {
	query := `
		SELECT id, campaign_id, customer_id, status, rendered_content, last_error, retry_count, created_at, updated_at
		FROM outbound_messages
		WHERE campaign_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by campaign: %w", err)
	}
	defer rows.Close()

	messages := []*models.OutboundMessage{}
	for rows.Next() {
		message := &models.OutboundMessage{}
		err := rows.Scan(
			&message.ID,
			&message.CampaignID,
			&message.CustomerID,
			&message.Status,
			&message.RenderedContent,
			&message.LastError,
			&message.RetryCount,
			&message.CreatedAt,
			&message.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}
