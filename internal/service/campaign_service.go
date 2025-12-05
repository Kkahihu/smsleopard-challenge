package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"smsleopard/internal/models"
	"smsleopard/internal/queue"
	"smsleopard/internal/repository"
)

// CampaignService handles campaign business logic
type CampaignService struct {
	campaignRepo repository.CampaignRepository
	customerRepo repository.CustomerRepository
	messageRepo  repository.MessageRepository
	templateSvc  *TemplateService
	publisher    *queue.Publisher
	db           *sql.DB
}

// NewCampaignService creates a new campaign service
func NewCampaignService(
	campaignRepo repository.CampaignRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.MessageRepository,
	templateSvc *TemplateService,
	publisher *queue.Publisher,
	db *sql.DB,
) *CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
		customerRepo: customerRepo,
		messageRepo:  messageRepo,
		templateSvc:  templateSvc,
		publisher:    publisher,
		db:           db,
	}
}

// CreateCampaign creates a new campaign
func (s *CampaignService) CreateCampaign(ctx context.Context, req *CreateCampaignRequest) (*models.Campaign, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, &ValidationError{Message: err.Error()}
	}

	// Validate template syntax
	if err := s.templateSvc.ValidateTemplate(req.BaseTemplate); err != nil {
		return nil, &ValidationError{Message: fmt.Sprintf("invalid template: %v", err)}
	}

	// Create campaign model
	campaign := &models.Campaign{
		Name:         req.Name,
		Channel:      req.Channel,
		Status:       models.CampaignStatusDraft,
		BaseTemplate: req.BaseTemplate,
		ScheduledAt:  req.ScheduledAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set status to scheduled if scheduled_at is in future
	if campaign.IsScheduled() {
		campaign.Status = models.CampaignStatusScheduled
	}

	// Save to database
	if err := s.campaignRepo.Create(ctx, campaign); err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	return campaign, nil
}

// GetCampaign retrieves a campaign by ID
func (s *CampaignService) GetCampaign(ctx context.Context, id int) (*models.Campaign, error) {
	campaign, err := s.campaignRepo.GetByID(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "campaign", ID: id}
	}
	return campaign, nil
}

// GetCampaignWithStats retrieves a campaign with statistics
func (s *CampaignService) GetCampaignWithStats(ctx context.Context, id int) (*models.CampaignWithStats, error) {
	campaign, err := s.campaignRepo.GetWithStats(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "campaign", ID: id}
	}
	return campaign, nil
}

// ListCampaigns lists campaigns with filters
func (s *CampaignService) ListCampaigns(ctx context.Context, filters repository.CampaignFilters) ([]*models.Campaign, *PaginationInfo, error) {
	campaigns, total, err := s.campaignRepo.List(ctx, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list campaigns: %w", err)
	}

	pageSize := filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	pagination := &PaginationInfo{
		Page:       filters.Page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: (total + pageSize - 1) / pageSize,
	}

	return campaigns, pagination, nil
}

// SendCampaign sends a campaign to specified customers
func (s *CampaignService) SendCampaign(ctx context.Context, campaignID int, customerIDs []int) (*SendCampaignResult, error) {
	// Get campaign
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, &NotFoundError{Resource: "campaign", ID: campaignID}
	}

	// Validate campaign can be sent
	if !campaign.CanSend() {
		return nil, &BusinessLogicError{
			Message: fmt.Sprintf("campaign cannot be sent: status is %s", campaign.Status),
		}
	}

	// Validate customer IDs provided
	if len(customerIDs) == 0 {
		return nil, &ValidationError{Message: "at least one customer ID required"}
	}

	// Get customers
	customers, err := s.customerRepo.GetByIDs(ctx, customerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}

	if len(customers) == 0 {
		return nil, &ValidationError{Message: "no valid customers found"}
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create outbound messages without rendered content (will be rendered by worker)
	messages := make([]*models.OutboundMessage, 0, len(customers))
	for _, customer := range customers {
		message := &models.OutboundMessage{
			CampaignID:      campaign.ID,
			CustomerID:      customer.ID,
			Status:          models.MessageStatusPending,
			RenderedContent: nil, // Will be set by worker
			RetryCount:      0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		messages = append(messages, message)
	}

	// Save messages in batch
	if err := s.messageRepo.CreateBatch(ctx, messages); err != nil {
		return nil, fmt.Errorf("failed to create messages: %w", err)
	}

	// Update campaign status to sending
	if err := s.campaignRepo.UpdateStatus(ctx, campaign.ID, models.CampaignStatusSending); err != nil {
		return nil, fmt.Errorf("failed to update campaign status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish jobs to queue (outside transaction)
	for _, message := range messages {
		err := s.publisher.PublishMessage(message.ID, campaign.ID, message.CustomerID)
		if err != nil {
			// Log error but don't fail - worker will retry
			log.Printf("Warning: Failed to publish message %d to queue: %v", message.ID, err)
		}
	}

	return &SendCampaignResult{
		CampaignID:     campaign.ID,
		MessagesQueued: len(messages),
		Status:         models.CampaignStatusSending,
	}, nil
}

// PreviewMessage previews how a message will render for a customer
func (s *CampaignService) PreviewMessage(ctx context.Context, req *PreviewMessageRequest) (*PreviewMessageResult, error) {
	// Get campaign
	campaign, err := s.campaignRepo.GetByID(ctx, req.CampaignID)
	if err != nil {
		return nil, &NotFoundError{Resource: "campaign", ID: req.CampaignID}
	}

	// Get customer
	customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, &NotFoundError{Resource: "customer", ID: req.CustomerID}
	}

	// Use override template if provided, otherwise use campaign template
	template := campaign.BaseTemplate
	if req.OverrideTemplate != nil && *req.OverrideTemplate != "" {
		template = *req.OverrideTemplate
	}

	// Render template
	renderedMessage, err := s.templateSvc.Render(template, customer)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &PreviewMessageResult{
		RenderedMessage: renderedMessage,
		UsedTemplate:    template,
		Customer: struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
		}{
			ID:        customer.ID,
			FirstName: customer.FullName(),
		},
	}, nil
}

// Request/Response types

// CreateCampaignRequest represents a request to create a campaign
type CreateCampaignRequest struct {
	Name         string         `json:"name"`
	Channel      models.Channel `json:"channel"`
	BaseTemplate string         `json:"base_template"`
	ScheduledAt  *time.Time     `json:"scheduled_at,omitempty"`
}

// Validate validates the create campaign request
func (r *CreateCampaignRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Channel != models.ChannelSMS && r.Channel != models.ChannelWhatsApp {
		return fmt.Errorf("invalid channel: must be 'sms' or 'whatsapp'")
	}
	if r.BaseTemplate == "" {
		return fmt.Errorf("base_template is required")
	}
	return nil
}

// SendCampaignResult represents the result of sending a campaign
type SendCampaignResult struct {
	CampaignID     int                   `json:"campaign_id"`
	MessagesQueued int                   `json:"messages_queued"`
	Status         models.CampaignStatus `json:"status"`
}

// PreviewMessageRequest represents a request to preview a message
type PreviewMessageRequest struct {
	CampaignID       int     `json:"campaign_id"`
	CustomerID       int     `json:"customer_id"`
	OverrideTemplate *string `json:"override_template,omitempty"`
}

// PreviewMessageResult represents the result of previewing a message
type PreviewMessageResult struct {
	RenderedMessage string `json:"rendered_message"`
	UsedTemplate    string `json:"used_template"`
	Customer        struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
	} `json:"customer"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}
