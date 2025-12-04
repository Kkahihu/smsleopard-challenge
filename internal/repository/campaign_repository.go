package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"smsleopard/internal/models"
)

type campaignRepository struct {
	db *sql.DB
}

// NewCampaignRepository creates a new campaign repository
func NewCampaignRepository(db *sql.DB) CampaignRepository {
	return &campaignRepository{db: db}
}

// Create creates a new campaign
func (r *campaignRepository) Create(ctx context.Context, campaign *models.Campaign) error {
	query := `
		INSERT INTO campaigns (name, channel, status, base_template, scheduled_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		campaign.Name,
		campaign.Channel,
		campaign.Status,
		campaign.BaseTemplate,
		campaign.ScheduledAt,
	).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}

	return nil
}

// GetByID retrieves a campaign by ID
func (r *campaignRepository) GetByID(ctx context.Context, id int) (*models.Campaign, error) {
	query := `
		SELECT id, name, channel, status, base_template, scheduled_at, created_at, updated_at
		FROM campaigns
		WHERE id = $1
	`

	campaign := &models.Campaign{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.Channel,
		&campaign.Status,
		&campaign.BaseTemplate,
		&campaign.ScheduledAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("campaign not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}

	return campaign, nil
}

// GetWithStats retrieves a campaign with statistics
func (r *campaignRepository) GetWithStats(ctx context.Context, id int) (*models.CampaignWithStats, error) {
	campaign, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	statsQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'sent') as sent,
			COUNT(*) FILTER (WHERE status = 'failed') as failed
		FROM outbound_messages
		WHERE campaign_id = $1
	`

	stats := models.CampaignStats{}
	err = r.db.QueryRowContext(ctx, statsQuery, id).Scan(
		&stats.Total,
		&stats.Pending,
		&stats.Sent,
		&stats.Failed,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get campaign stats: %w", err)
	}

	return &models.CampaignWithStats{
		Campaign: *campaign,
		Stats:    stats,
	}, nil
}

// List retrieves campaigns with filters and pagination
func (r *campaignRepository) List(ctx context.Context, filters CampaignFilters) ([]*models.Campaign, int, error) {
	// Build query with filters
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, name, channel, status, base_template, scheduled_at, created_at, updated_at
		FROM campaigns
		WHERE 1=1
	`)

	args := []interface{}{}
	argPos := 1

	if filters.Channel != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND channel = $%d", argPos))
		args = append(args, *filters.Channel)
		argPos++
	}

	if filters.Status != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	// Order by ID DESC for stable pagination
	queryBuilder.WriteString(" ORDER BY id DESC")

	// Add pagination
	limit := filters.PageSize
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (filters.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, limit, offset)

	// Execute query
	rows, err := r.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := []*models.Campaign{}
	for rows.Next() {
		campaign := &models.Campaign{}
		err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.Channel,
			&campaign.Status,
			&campaign.BaseTemplate,
			&campaign.ScheduledAt,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM campaigns WHERE 1=1"
	countArgs := []interface{}{}

	if filters.Channel != nil {
		countQuery += " AND channel = $1"
		countArgs = append(countArgs, *filters.Channel)
	}

	if filters.Status != nil {
		pos := len(countArgs) + 1
		countQuery += fmt.Sprintf(" AND status = $%d", pos)
		countArgs = append(countArgs, *filters.Status)
	}

	var totalCount int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	return campaigns, totalCount, nil
}

// UpdateStatus updates campaign status
func (r *campaignRepository) UpdateStatus(ctx context.Context, id int, status models.CampaignStatus) error {
	query := `
		UPDATE campaigns
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update campaign status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("campaign not found")
	}

	return nil
}

// Delete deletes a campaign
func (r *campaignRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM campaigns WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("campaign not found")
	}

	return nil
}
