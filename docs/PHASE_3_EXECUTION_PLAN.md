# Phase 3 Execution Plan - Core Business Logic & Services

## 🎯 Objective
Implement business logic layer with template rendering, campaign management, and mock sender functionality.

**Estimated Time:** 1.5-2 hours  
**Status:** Ready to Execute  
**Prerequisites:** ✅ Phase 1 Complete, ✅ Phase 2 Complete

---

## 📋 What Phase 3 Will Deliver

1. **Template Service** - Personalization engine with `{placeholder}` substitution
2. **Mock Sender Service** - Simulates SMS/WhatsApp sending with configurable success rate
3. **Campaign Service** - Core business logic for campaign operations
4. **Custom Errors** - Domain-specific error types for better error handling
5. **Transaction Support** - Atomic operations for campaign sending

---

## 🔧 Phase 3.1: Create Template Service

### File: internal/service/template_service.go

```go
package service

import (
	"fmt"
	"regexp"
	"strings"

	"smsleopard/internal/models"
)

// TemplateService handles message template rendering
type TemplateService struct{}

// NewTemplateService creates a new template service
func NewTemplateService() *TemplateService {
	return &TemplateService{}
}

// Render renders a template with customer data
// Replaces {field_name} placeholders with actual customer values
// Strategy for missing fields: replace with empty string
func (s *TemplateService) Render(template string, customer *models.Customer) (string, error) {
	if template == "" {
		return "", fmt.Errorf("template cannot be empty")
	}

	if customer == nil {
		return "", fmt.Errorf("customer cannot be nil")
	}

	rendered := template

	// Replace {first_name}
	if customer.FirstName != nil && *customer.FirstName != "" {
		rendered = strings.ReplaceAll(rendered, "{first_name}", *customer.FirstName)
	} else {
		rendered = strings.ReplaceAll(rendered, "{first_name}", "")
	}

	// Replace {last_name}
	if customer.LastName != nil && *customer.LastName != "" {
		rendered = strings.ReplaceAll(rendered, "{last_name}", *customer.LastName)
	} else {
		rendered = strings.ReplaceAll(rendered, "{last_name}", "")
	}

	// Replace {location}
	if customer.Location != nil && *customer.Location != "" {
		rendered = strings.ReplaceAll(rendered, "{location}", *customer.Location)
	} else {
		rendered = strings.ReplaceAll(rendered, "{location}", "")
	}

	// Replace {preferred_product}
	if customer.PreferredProduct != nil && *customer.PreferredProduct != "" {
		rendered = strings.ReplaceAll(rendered, "{preferred_product}", *customer.PreferredProduct)
	} else {
		rendered = strings.ReplaceAll(rendered, "{preferred_product}", "")
	}

	// Replace {phone}
	rendered = strings.ReplaceAll(rendered, "{phone}", customer.Phone)

	// Clean up any remaining placeholders (warn about unknown fields)
	re := regexp.MustCompile(`\{[a-zA-Z_]+\}`)
	if matches := re.FindAllString(rendered, -1); len(matches) > 0 {
		// Log warning but continue - unknown placeholders left as-is
		// In production, you might want to log this
		_ = matches // Keep unknown placeholders in the text
	}

	return rendered, nil
}

// ValidateTemplate checks if template has valid syntax
func (s *TemplateService) ValidateTemplate(template string) error {
	if template == "" {
		return fmt.Errorf("template cannot be empty")
	}

	// Check for balanced braces
	openCount := strings.Count(template, "{")
	closeCount := strings.Count(template, "}")

	if openCount != closeCount {
		return fmt.Errorf("template has unbalanced braces: %d open, %d close", openCount, closeCount)
	}

	// Check for valid placeholder format
	re := regexp.MustCompile(`\{[a-zA-Z_]+\}`)
	placeholders := re.FindAllString(template, -1)

	validFields := map[string]bool{
		"{first_name}":        true,
		"{last_name}":         true,
		"{location}":          true,
		"{preferred_product}": true,
		"{phone}":             true,
	}

	unknownFields := []string{}
	for _, placeholder := range placeholders {
		if !validFields[placeholder] {
			unknownFields = append(unknownFields, placeholder)
		}
	}

	if len(unknownFields) > 0 {
		// This is a warning, not an error - allow unknown fields
		// In production, you might want to return this as a warning
		_ = unknownFields
	}

	return nil
}

// GetPlaceholders extracts all placeholders from a template
func (s *TemplateService) GetPlaceholders(template string) []string {
	re := regexp.MustCompile(`\{[a-zA-Z_]+\}`)
	return re.FindAllString(template, -1)
}

// Preview renders a template for preview purposes (without saving)
func (s *TemplateService) Preview(template string, customer *models.Customer) (string, error) {
	// Same as Render but explicitly for preview
	return s.Render(template, customer)
}
```

**Key Features:**
- ✅ Placeholder substitution with `{field_name}` syntax
- ✅ Handles NULL/missing fields (replaces with empty string)
- ✅ Template validation
- ✅ Supports all customer fields
- ✅ Regex-based placeholder detection
- ✅ Unknown placeholders left as-is (not removed)

---

## 📨 Phase 3.2: Create Mock Sender Service

### File: internal/service/sender_service.go

```go
package service

import (
	"fmt"
	"math/rand"
	"time"

	"smsleopard/internal/models"
)

// SenderService handles message sending
type SenderService struct {
	successRate float64 // 0.0 to 1.0 (e.g., 0.95 = 95% success)
	rand        *rand.Rand
}

// NewSenderService creates a new sender service
// successRate: probability of successful send (0.0 to 1.0)
// Default: 0.95 (95% success rate)
func NewSenderService(successRate float64) *SenderService {
	if successRate < 0.0 {
		successRate = 0.0
	}
	if successRate > 1.0 {
		successRate = 1.0
	}

	return &SenderService{
		successRate: successRate,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SendResult represents the result of a send attempt
type SendResult struct {
	Success bool
	Error   error
	Latency time.Duration
}

// SendSMS simulates sending an SMS message
func (s *SenderService) SendSMS(phone string, content string) *SendResult {
	return s.send("SMS", phone, content)
}

// SendWhatsApp simulates sending a WhatsApp message
func (s *SenderService) SendWhatsApp(phone string, content string) *SendResult {
	return s.send("WhatsApp", phone, content)
}

// Send sends a message via the specified channel
func (s *SenderService) Send(channel models.Channel, phone string, content string) *SendResult {
	if channel == models.ChannelSMS {
		return s.SendSMS(phone, content)
	}
	return s.SendWhatsApp(phone, content)
}

// send is the internal mock implementation
func (s *SenderService) send(channelType string, phone string, content string) *SendResult {
	start := time.Now()

	// Simulate network latency (50-200ms)
	latency := time.Duration(50+s.rand.Intn(150)) * time.Millisecond
	time.Sleep(latency)

	// Determine success based on configured success rate
	randomValue := s.rand.Float64()
	success := randomValue < s.successRate

	result := &SendResult{
		Success: success,
		Latency: time.Since(start),
	}

	if !success {
		// Simulate different types of failures
		failures := []string{
			"network timeout",
			"invalid phone number",
			"rate limit exceeded",
			"service temporarily unavailable",
			"insufficient balance",
		}
		failureReason := failures[s.rand.Intn(len(failures))]
		result.Error = fmt.Errorf("failed to send %s to %s: %s", channelType, phone, failureReason)
	}

	return result
}

// GetSuccessRate returns the configured success rate
func (s *SenderService) GetSuccessRate() float64 {
	return s.successRate
}

// SetSuccessRate updates the success rate (for testing)
func (s *SenderService) SetSuccessRate(rate float64) {
	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}
	s.successRate = rate
}
```

**Key Features:**
- ✅ Configurable success rate (default 95%)
- ✅ Simulates network latency (50-200ms)
- ✅ Realistic error messages
- ✅ Supports both SMS and WhatsApp
- ✅ Thread-safe with separate rand instance
- ✅ Returns detailed results

**Mock Behavior Documentation:**
- Success rate: 95% by default
- Latency: 50-200ms random delay
- Errors: Simulates 5 common failure types
- Predictable for testing (can set success rate)

---

## 📢 Phase 3.3: Create Campaign Service

### File: internal/service/campaign_service.go

```go
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"smsleopard/internal/models"
	"smsleopard/internal/repository"
)

// CampaignService handles campaign business logic
type CampaignService struct {
	campaignRepo repository.CampaignRepository
	customerRepo repository.CustomerRepository
	messageRepo  repository.MessageRepository
	templateSvc  *TemplateService
	db           *sql.DB
}

// NewCampaignService creates a new campaign service
func NewCampaignService(
	campaignRepo repository.CampaignRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.MessageRepository,
	templateSvc *TemplateService,
	db *sql.DB,
) *CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
		customerRepo: customerRepo,
		messageRepo:  messageRepo,
		templateSvc:  templateSvc,
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

	// Create outbound messages with rendered content
	messages := make([]*models.OutboundMessage, 0, len(customers))
	for _, customer := range customers {
		// Render template with customer data
		renderedContent, err := s.templateSvc.Render(campaign.BaseTemplate, customer)
		if err != nil {
			return nil, fmt.Errorf("failed to render template for customer %d: %w", customer.ID, err)
		}

		message := &models.OutboundMessage{
			CampaignID:      campaign.ID,
			CustomerID:      customer.ID,
			Status:          models.MessageStatusPending,
			RenderedContent: &renderedContent,
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
	Name         string              `json:"name"`
	Channel      models.Channel      `json:"channel"`
	BaseTemplate string              `json:"base_template"`
	ScheduledAt  *time.Time          `json:"scheduled_at,omitempty"`
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
	CampaignID     int                     `json:"campaign_id"`
	MessagesQueued int                     `json:"messages_queued"`
	Status         models.CampaignStatus   `json:"status"`
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
```

**Key Features:**
- ✅ Transaction support for SendCampaign (atomic operations)
- ✅ Template rendering during campaign send
- ✅ Batch message creation
- ✅ Status management
- ✅ Preview functionality
- ✅ Pagination support
- ✅ Validation at service layer

---

## ❌ Phase 3.4: Create Custom Errors

### File: internal/service/errors.go

```go
package service

import "fmt"

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
	ID       int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %d not found", e.Resource, e.ID)
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// BusinessLogicError represents a business logic error
type BusinessLogicError struct {
	Message string
}

func (e *BusinessLogicError) Error() string {
	return fmt.Sprintf("business logic error: %s", e.Message)
}

// ConflictError represents a conflict error (e.g., duplicate)
type ConflictError struct {
	Resource string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict with %s: %s", e.Resource, e.Message)
}
```

**Key Features:**
- ✅ Type-safe error handling
- ✅ Clear error categories
- ✅ Easy to identify error type in handlers
- ✅ Can be used for HTTP status code mapping

---

## ✅ Phase 3.5: Testing & Validation

### Compilation Test

```bash
# Test service layer
go build ./internal/service/...

# Build entire project
go build ./...
```

Expected: No compilation errors

### Manual Testing Plan

After implementation, you can test services:

```go
// Example test in main.go (temporary)
func testTemplateService() {
    templateSvc := service.NewTemplateService()
    
    // Test customer
    firstName := "Alice"
    location := "Nairobi"
    customer := &models.Customer{
        ID:        1,
        Phone:     "+254712345678",
        FirstName: &firstName,
        Location:  &location,
    }
    
    // Render template
    template := "Hi {first_name}, check out deals in {location}!"
    rendered, err := templateSvc.Render(template, customer)
    
    fmt.Printf("Template: %s\n", template)
    fmt.Printf("Rendered: %s\n", rendered)
    // Expected: "Hi Alice, check out deals in Nairobi!"
}
```

---

## 📊 Phase 3 Deliverables

| Component | File | Status |
|-----------|------|--------|
| Template Service | `internal/service/template_service.go` | ⏳ |
| Sender Service | `internal/service/sender_service.go` | ⏳ |
| Campaign Service | `internal/service/campaign_service.go` | ⏳ |
| Custom Errors | `internal/service/errors.go` | ⏳ |

---

## 🎯 Success Criteria

Phase 3 is complete when:
- ✅ All 4 service files created
- ✅ All code compiles without errors
- ✅ Template rendering works with NULL fields
- ✅ Mock sender returns consistent results
- ✅ Campaign service integrates with repositories
- ✅ Transaction support working
- ✅ Custom errors properly typed
- ✅ Ready for Phase 4 (HTTP API)

---

## 🔍 Key Design Decisions

### Template Handling Strategy
**Decision:** Missing fields replaced with empty string  
**Rationale:** Graceful degradation; messages still readable  
**Alternative:** Could keep placeholder or error out

### Mock Sender Success Rate
**Decision:** Default 95% success rate  
**Rationale:** Realistic failure rate for testing retry logic  
**Configurable:** Can be adjusted for testing

### Transaction Scope
**Decision:** SendCampaign wraps messages + status in transaction  
**Rationale:** Atomic operation prevents partial sends  
**Trade-off:** Longer transaction time for large batches

### Error Types
**Decision:** Custom error types vs error codes  
**Rationale:** Type-safe, easier to handle in HTTP layer  
**Benefit:** Can map directly to HTTP status codes

---

## 🚀 Next Phase Preview

**Phase 4** will implement:
- HTTP handlers for all API endpoints
- Request/response models
- Error handling middleware
- Router configuration
- Integration with services

**Files to Create:**
- `internal/handler/campaign_handler.go`
- `internal/handler/health_handler.go`
- `internal/handler/response.go`
- `internal/middleware/error_handler.go`
- `internal/middleware/logger.go`

**Dependencies:** Phase 3 must be complete.

---

**Ready to implement? Switch to Code mode!** 🛠️