package repository

import (
	"context"
	"database/sql"

	"smsleopard/internal/models"
)

// CustomerRepository defines customer data access operations
type CustomerRepository interface {
	Create(ctx context.Context, customer *models.Customer) error
	GetByID(ctx context.Context, id int) (*models.Customer, error)
	GetByIDs(ctx context.Context, ids []int) ([]*models.Customer, error)
	List(ctx context.Context, limit, offset int) ([]*models.Customer, error)
	Update(ctx context.Context, customer *models.Customer) error
	Delete(ctx context.Context, id int) error
}

// CampaignRepository defines campaign data access operations
type CampaignRepository interface {
	Create(ctx context.Context, campaign *models.Campaign) error
	GetByID(ctx context.Context, id int) (*models.Campaign, error)
	GetWithStats(ctx context.Context, id int) (*models.CampaignWithStats, error)
	List(ctx context.Context, filters CampaignFilters) ([]*models.Campaign, int, error)
	UpdateStatus(ctx context.Context, id int, status models.CampaignStatus) error
	Delete(ctx context.Context, id int) error
}

// CampaignFilters defines filters for listing campaigns
type CampaignFilters struct {
	Page     int
	PageSize int
	Channel  *models.Channel
	Status   *models.CampaignStatus
}

// MessageRepository defines outbound message data access operations
type MessageRepository interface {
	Create(ctx context.Context, message *models.OutboundMessage) error
	CreateBatch(ctx context.Context, messages []*models.OutboundMessage) error
	GetByID(ctx context.Context, id int) (*models.OutboundMessage, error)
	GetWithDetails(ctx context.Context, id int) (*models.OutboundMessageWithDetails, error)
	UpdateStatus(ctx context.Context, id int, status models.MessageStatus, lastError *string) error
	GetPendingMessages(ctx context.Context, limit int) ([]*models.OutboundMessage, error)
	GetByCampaignID(ctx context.Context, campaignID int) ([]*models.OutboundMessage, error)
}

// DB is a wrapper around *sql.DB to allow passing in transaction
type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
