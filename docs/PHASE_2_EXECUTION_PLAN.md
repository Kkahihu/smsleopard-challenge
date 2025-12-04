# Phase 2 Execution Plan - Data Models & Repository Layer

## 🎯 Objective
Implement data models and repository layer for database operations.

**Estimated Time:** 1.5-2 hours  
**Status:** Ready to Execute  
**Prerequisites:** ✅ Phase 1 Complete

---

## 📋 What Phase 2 Will Deliver

1. **Data Models** - Go structs representing database tables
2. **Repository Interfaces** - Contracts for data access
3. **Repository Implementations** - Database CRUD operations
4. **Connection Pooling** - Efficient database connections
5. **Null Handling** - Proper handling of optional fields

---

## 📊 Phase 2.1: Create Data Model Structs

### File: internal/models/customer.go

```go
package models

import "time"

// Customer represents a customer in the system
type Customer struct {
	ID               int       `json:"id" db:"id"`
	Phone            string    `json:"phone" db:"phone"`
	FirstName        *string   `json:"first_name,omitempty" db:"first_name"`
	LastName         *string   `json:"last_name,omitempty" db:"last_name"`
	Location         *string   `json:"location,omitempty" db:"location"`
	PreferredProduct *string   `json:"preferred_product,omitempty" db:"preferred_product"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// FullName returns the customer's full name
func (c *Customer) FullName() string {
	var firstName, lastName string
	
	if c.FirstName != nil {
		firstName = *c.FirstName
	}
	if c.LastName != nil {
		lastName = *c.LastName
	}
	
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	return "Customer"
}
```

**Key Features:**
- ✅ Pointers for nullable fields (FirstName, LastName, Location, PreferredProduct)
- ✅ JSON tags for API responses
- ✅ DB tags for scanning from database
- ✅ Helper method for full name
- ✅ Matches database schema

---

### File: internal/models/campaign.go

```go
package models

import (
	"fmt"
	"time"
)

// CampaignStatus represents valid campaign statuses
type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusSending   CampaignStatus = "sending"
	CampaignStatusSent      CampaignStatus = "sent"
	CampaignStatusFailed    CampaignStatus = "failed"
)

// Channel represents valid messaging channels
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// Campaign represents a campaign in the system
type Campaign struct {
	ID           int             `json:"id" db:"id"`
	Name         string          `json:"name" db:"name"`
	Channel      Channel         `json:"channel" db:"channel"`
	Status       CampaignStatus  `json:"status" db:"status"`
	BaseTemplate string          `json:"base_template" db:"base_template"`
	ScheduledAt  *time.Time      `json:"scheduled_at,omitempty" db:"scheduled_at"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// CampaignStats represents campaign statistics
type CampaignStats struct {
	Total   int `json:"total"`
	Pending int `json:"pending"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
}

// CampaignWithStats represents a campaign with its statistics
type CampaignWithStats struct {
	Campaign
	Stats CampaignStats `json:"stats"`
}

// Validate checks if the campaign fields are valid
func (c *Campaign) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("campaign name is required")
	}
	if c.Channel != ChannelSMS && c.Channel != ChannelWhatsApp {
		return fmt.Errorf("invalid channel: must be 'sms' or 'whatsapp'")
	}
	if c.BaseTemplate == "" {
		return fmt.Errorf("base template is required")
	}
	return nil
}

// IsScheduled checks if campaign is scheduled for future
func (c *Campaign) IsScheduled() bool {
	return c.ScheduledAt != nil && c.ScheduledAt.After(time.Now())
}

// CanSend checks if campaign can be sent
func (c *Campaign) CanSend() bool {
	return c.Status == CampaignStatusDraft || c.Status == CampaignStatusScheduled
}
```

**Key Features:**
- ✅ Type-safe status and channel enums
- ✅ Validation method
- ✅ Helper methods (IsScheduled, CanSend)
- ✅ Stats struct for aggregated data
- ✅ Matches database schema

---

### File: internal/models/message.go

```go
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
```

**Key Features:**
- ✅ Type-safe status enum
- ✅ Helper methods for retry logic
- ✅ Composite struct with campaign and customer
- ✅ Matches database schema
- ✅ Max retry count enforcement (3)

---

## 🔌 Phase 2.2: Create Repository Interfaces

### File: internal/repository/repository.go

```go
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
```

**Key Features:**
- ✅ Context-aware operations (for timeouts and cancellation)
- ✅ Interface segregation (separate repositories)
- ✅ Batch operations support
- ✅ Filtering and pagination
- ✅ DB interface for transaction support

---

## 👥 Phase 2.3: Implement Customer Repository

### File: internal/repository/customer_repository.go

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"smsleopard/internal/models"
)

type customerRepository struct {
	db *sql.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepository{db: db}
}

// Create creates a new customer
func (r *customerRepository) Create(ctx context.Context, customer *models.Customer) error {
	query := `
		INSERT INTO customers (phone, first_name, last_name, location, preferred_product)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
	).Scan(&customer.ID, &customer.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	return nil
}

// GetByID retrieves a customer by ID
func (r *customerRepository) GetByID(ctx context.Context, id int) (*models.Customer, error) {
	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		WHERE id = $1
	`

	customer := &models.Customer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&customer.ID,
		&customer.Phone,
		&customer.FirstName,
		&customer.LastName,
		&customer.Location,
		&customer.PreferredProduct,
		&customer.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("customer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return customer, nil
}

// GetByIDs retrieves multiple customers by IDs
func (r *customerRepository) GetByIDs(ctx context.Context, ids []int) ([]*models.Customer, error) {
	if len(ids) == 0 {
		return []*models.Customer{}, nil
	}

	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		WHERE id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}
	defer rows.Close()

	customers := []*models.Customer{}
	for rows.Next() {
		customer := &models.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.Phone,
			&customer.FirstName,
			&customer.LastName,
			&customer.Location,
			&customer.PreferredProduct,
			&customer.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// List retrieves customers with pagination
func (r *customerRepository) List(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	customers := []*models.Customer{}
	for rows.Next() {
		customer := &models.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.Phone,
			&customer.FirstName,
			&customer.LastName,
			&customer.Location,
			&customer.PreferredProduct,
			&customer.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// Update updates a customer
func (r *customerRepository) Update(ctx context.Context, customer *models.Customer) error {
	query := `
		UPDATE customers
		SET phone = $1, first_name = $2, last_name = $3, location = $4, preferred_product = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// Delete deletes a customer
func (r *customerRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM customers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}
```

**Note:** Add import for pq.Array:
```go
import (
	"github.com/lib/pq"
)
```

---

## 📢 Phase 2.4: Implement Campaign Repository

### File: internal/repository/campaign_repository.go

```go
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
```

---

## 💬 Phase 2.5: Implement Message Repository

### File: internal/repository/message_repository.go

```go
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
```

---

## ✅ Phase 2.6: Testing & Validation

### Test Checklist

- [ ] All model files compile without errors
- [ ] All repository files compile without errors
- [ ] No import errors
- [ ] Proper null handling for pointer fields
- [ ] Repository interfaces match implementations

### Compilation Test

```bash
# Test models
go build ./internal/models/...

# Test repositories
go build ./internal/repository/...

# Build entire project
go build ./...
```

Expected: No errors

---

## 📊 Phase 2 Deliverables

| Component | File | Status |
|-----------|------|--------|
| Customer Model | `internal/models/customer.go` | ⏳ |
| Campaign Model | `internal/models/campaign.go` | ⏳ |
| Message Model | `internal/models/message.go` | ⏳ |
| Repository Interfaces | `internal/repository/repository.go` | ⏳ |
| Customer Repository | `internal/repository/customer_repository.go` | ⏳ |
| Campaign Repository | `internal/repository/campaign_repository.go` | ⏳ |
| Message Repository | `internal/repository/message_repository.go` | ⏳ |

---

## 🎯 Success Criteria

Phase 2 is complete when:
- ✅ All 7 files created
- ✅ All code compiles without errors
- ✅ Models match database schema
- ✅ Repositories implement all interface methods
- ✅ Proper error handling
- ✅ Context support for cancellation
- ✅ Ready for Phase 3 (Services)

---

## 🚀 Next Phase Preview

**Phase 3** will implement:
- Template service (personalization)
- Campaign service (business logic)
- Mock sender service
- Integration with repositories

**Dependencies:** Phase 2 must be 100% complete.

---

**Ready to implement? Switch to Code mode!** 🛠️