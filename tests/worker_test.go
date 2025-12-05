package tests

import (
	"context"
	"database/sql"
	"testing"

	"smsleopard/internal/models"
	"smsleopard/internal/repository"
	"smsleopard/internal/service"
)

// setupWorkerTest creates test database and services for worker tests
func setupWorkerTest(t *testing.T) (*sql.DB, repository.MessageRepository, repository.CampaignRepository, repository.CustomerRepository, *service.TemplateService, *service.SenderService, func()) {
	t.Helper()

	// Setup test database
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil, nil, nil, nil, nil, nil
	}

	// Clean up test data
	CleanupTestDB(t, db)

	// Create repositories
	messageRepo := repository.NewMessageRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	customerRepo := repository.NewCustomerRepository(db)

	// Create services
	templateSvc := service.NewTemplateService()
	senderSvc := service.NewSenderService(0.95) // Default 95% success

	cleanup := func() {
		CleanupTestDB(t, db)
		db.Close()
	}

	return db, messageRepo, campaignRepo, customerRepo, templateSvc, senderSvc, cleanup
}

// TestWorker_SuccessfulProcessing tests complete successful message processing flow
func TestWorker_SuccessfulProcessing(t *testing.T) {
	_, msgRepo, campRepo, custRepo, templateSvc, senderSvc, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: Create customer, campaign, and message
	customer := &models.Customer{
		Phone:     "+254700000001",
		FirstName: StringPtr("John"),
		LastName:  StringPtr("Doe"),
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Test Campaign",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Hi {first_name}, welcome!",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	message := &models.OutboundMessage{
		CampaignID: campaign.ID,
		CustomerID: customer.ID,
		Status:     models.MessageStatusPending,
		RetryCount: 0,
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

	// Execute: Simulate worker processing
	// 1. Fetch message with campaign and customer
	fetchedMsg, err := msgRepo.GetWithDetails(ctx, message.ID)
	AssertNoError(t, err)
	AssertNotNil(t, fetchedMsg)

	// 2. Render template
	renderedContent, err := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)
	AssertNoError(t, err)
	AssertEqual(t, renderedContent, "Hi John, welcome!")

	// 3. Send message
	senderSvc.SetSuccessRate(1.0) // 100% success for this test
	result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)
	AssertEqual(t, result.Success, true)
	AssertNil(t, result.Error)

	// 4. Update message status
	err = msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusSent, nil)
	AssertNoError(t, err)

	// Verify: Check message status updated
	updatedMsg, err := msgRepo.GetByID(ctx, message.ID)
	AssertNoError(t, err)
	AssertEqual(t, updatedMsg.Status, models.MessageStatusSent)
	AssertNil(t, updatedMsg.LastError)
	AssertEqual(t, updatedMsg.RetryCount, 0)
}

// TestWorker_FailureScenario tests worker handling of sender service failures
func TestWorker_FailureScenario(t *testing.T) {
	_, msgRepo, campRepo, custRepo, templateSvc, senderSvc, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: Create customer, campaign, and message
	customer := &models.Customer{
		Phone:     "+254700000002",
		FirstName: StringPtr("Jane"),
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Test Campaign Fail",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Hi {first_name}!",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	message := &models.OutboundMessage{
		CampaignID: campaign.ID,
		CustomerID: customer.ID,
		Status:     models.MessageStatusPending,
		RetryCount: 0,
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

	// Execute: Simulate sender failure
	fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
	renderedContent, _ := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)

	senderSvc.SetSuccessRate(0.0) // 0% success - always fail
	result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)

	// Verify: Error occurred
	AssertEqual(t, result.Success, false)
	AssertNotNil(t, result.Error)

	// Update status to failed
	errorMsg := result.Error.Error()
	err := msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusFailed, &errorMsg)
	AssertNoError(t, err)

	// Verify status updated
	updatedMsg, _ := msgRepo.GetByID(ctx, message.ID)
	AssertEqual(t, updatedMsg.Status, models.MessageStatusFailed)
	AssertNotNil(t, updatedMsg.LastError)
	AssertContains(t, *updatedMsg.LastError, "failed to send")
}

// TestWorker_RetryLogic tests message retry mechanism
func TestWorker_RetryLogic(t *testing.T) {
	db, msgRepo, campRepo, custRepo, templateSvc, senderSvc, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup
	customer := &models.Customer{
		Phone:     "+254700000003",
		FirstName: StringPtr("Bob"),
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Test Retry Campaign",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Hello {first_name}",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	message := &models.OutboundMessage{
		CampaignID: campaign.ID,
		CustomerID: customer.ID,
		Status:     models.MessageStatusPending,
		RetryCount: 0,
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

	fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
	renderedContent, _ := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)

	// Simulate 3 failures (max retries)
	senderSvc.SetSuccessRate(0.0)

	for retry := 0; retry < 3; retry++ {
		// Try to send
		result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)
		AssertEqual(t, result.Success, false)
		AssertNotNil(t, result.Error)

		// Increment retry count using direct SQL (simulating worker behavior)
		_, err := db.ExecContext(ctx, "UPDATE outbound_messages SET retry_count = retry_count + 1, status = 'pending', updated_at = CURRENT_TIMESTAMP WHERE id = $1", message.ID)
		AssertNoError(t, err)

		// Verify retry count incremented
		msg, _ := msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, msg.RetryCount, retry+1)
		AssertEqual(t, msg.Status, models.MessageStatusPending)
	}

	// After 3 retries, mark as permanent failure
	finalMsg, _ := msgRepo.GetByID(ctx, message.ID)
	if finalMsg.RetryCount >= 3 {
		errorMsg := "max retries exceeded"
		err := msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusFailed, &errorMsg)
		AssertNoError(t, err)
	}

	// Verify final state
	finalMsg, _ = msgRepo.GetByID(ctx, message.ID)
	AssertEqual(t, finalMsg.RetryCount, 3)
	AssertEqual(t, finalMsg.Status, models.MessageStatusFailed)
	AssertNotNil(t, finalMsg.LastError)
	AssertEqual(t, *finalMsg.LastError, "max retries exceeded")
}

// TestWorker_StatusTransitions tests various status transition scenarios
func TestWorker_StatusTransitions(t *testing.T) {
	_, msgRepo, campRepo, custRepo, templateSvc, senderSvc, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test 1: pending → sent (success path)
	t.Run("PendingToSent", func(t *testing.T) {
		customer := &models.Customer{
			Phone:     "+254700000004",
			FirstName: StringPtr("Alice"),
		}
		AssertNoError(t, custRepo.Create(ctx, customer))

		campaign := &models.Campaign{
			Name:         "Status Test 1",
			Channel:      models.ChannelSMS,
			Status:       models.CampaignStatusSending,
			BaseTemplate: "Hi {first_name}",
		}
		AssertNoError(t, campRepo.Create(ctx, campaign))

		message := &models.OutboundMessage{
			CampaignID: campaign.ID,
			CustomerID: customer.ID,
			Status:     models.MessageStatusPending,
			RetryCount: 0,
		}
		AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

		// Process and send successfully
		fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
		renderedContent, _ := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)
		senderSvc.SetSuccessRate(1.0)
		result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)
		AssertEqual(t, result.Success, true)

		// Update to sent
		err := msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusSent, nil)
		AssertNoError(t, err)

		updatedMsg, _ := msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, updatedMsg.Status, models.MessageStatusSent)
	})

	// Test 2: pending → failed → pending (retry) → sent (recovery)
	t.Run("FailureRecovery", func(t *testing.T) {
		customer := &models.Customer{
			Phone:     "+254700000005",
			FirstName: StringPtr("Charlie"),
		}
		AssertNoError(t, custRepo.Create(ctx, customer))

		campaign := &models.Campaign{
			Name:         "Status Test 2",
			Channel:      models.ChannelSMS,
			Status:       models.CampaignStatusSending,
			BaseTemplate: "Hi {first_name}",
		}
		AssertNoError(t, campRepo.Create(ctx, campaign))

		message := &models.OutboundMessage{
			CampaignID: campaign.ID,
			CustomerID: customer.ID,
			Status:     models.MessageStatusPending,
			RetryCount: 0,
		}
		AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

		fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
		renderedContent, _ := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)

		// First attempt fails
		senderSvc.SetSuccessRate(0.0)
		result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)
		AssertEqual(t, result.Success, false)

		errorMsg := result.Error.Error()
		err := msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusFailed, &errorMsg)
		AssertNoError(t, err)

		msg, _ := msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, msg.Status, models.MessageStatusFailed)

		// Retry (set back to pending with incremented retry count)
		err = msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusPending, nil)
		AssertNoError(t, err)

		msg, _ = msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, msg.Status, models.MessageStatusPending)

		// Second attempt succeeds
		senderSvc.SetSuccessRate(1.0)
		result = senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)
		AssertEqual(t, result.Success, true)

		err = msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusSent, nil)
		AssertNoError(t, err)

		msg, _ = msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, msg.Status, models.MessageStatusSent)
	})
}

// TestWorker_TemplateRenderingError tests worker handling of template errors
func TestWorker_TemplateRenderingError(t *testing.T) {
	_, msgRepo, campRepo, custRepo, templateSvc, _, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup with nil customer (edge case)
	customer := &models.Customer{
		Phone:     "+254700000006",
		FirstName: nil, // No first name
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Template Error Test",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Hi {first_name}!",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	message := &models.OutboundMessage{
		CampaignID: campaign.ID,
		CustomerID: customer.ID,
		Status:     models.MessageStatusPending,
		RetryCount: 0,
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

	// Template rendering should handle nil fields gracefully
	fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
	renderedContent, err := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)
	AssertNoError(t, err)
	AssertEqual(t, renderedContent, "Hi !") // Empty first_name replaced with empty string
}

// TestWorker_MessageWithDetails tests fetching message with campaign and customer details
func TestWorker_MessageWithDetails(t *testing.T) {
	_, msgRepo, campRepo, custRepo, _, _, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup
	customer := &models.Customer{
		Phone:            "+254700000007",
		FirstName:        StringPtr("David"),
		LastName:         StringPtr("Smith"),
		Location:         StringPtr("Nairobi"),
		PreferredProduct: StringPtr("Premium"),
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Details Test",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Hi {first_name} {last_name} from {location}!",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	message := &models.OutboundMessage{
		CampaignID: campaign.ID,
		CustomerID: customer.ID,
		Status:     models.MessageStatusPending,
		RetryCount: 0,
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

	// Fetch with details
	msgWithDetails, err := msgRepo.GetWithDetails(ctx, message.ID)
	AssertNoError(t, err)
	AssertNotNil(t, msgWithDetails)

	// Verify campaign details
	AssertEqual(t, msgWithDetails.Campaign.ID, campaign.ID)
	AssertEqual(t, msgWithDetails.Campaign.Name, "Details Test")
	AssertEqual(t, msgWithDetails.Campaign.BaseTemplate, "Hi {first_name} {last_name} from {location}!")

	// Verify customer details
	AssertEqual(t, msgWithDetails.Customer.ID, customer.ID)
	AssertEqual(t, msgWithDetails.Customer.Phone, "+254700000007")
	AssertNotNil(t, msgWithDetails.Customer.FirstName)
	AssertEqual(t, *msgWithDetails.Customer.FirstName, "David")
	AssertNotNil(t, msgWithDetails.Customer.Location)
	AssertEqual(t, *msgWithDetails.Customer.Location, "Nairobi")
}

// TestWorker_PendingMessagesQuery tests retrieval of pending messages
func TestWorker_PendingMessagesQuery(t *testing.T) {
	_, msgRepo, campRepo, custRepo, _, _, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: Create multiple messages with different statuses
	customer := &models.Customer{
		Phone:     "+254700000008",
		FirstName: StringPtr("Eve"),
	}
	AssertNoError(t, custRepo.Create(ctx, customer))

	campaign := &models.Campaign{
		Name:         "Pending Query Test",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusSending,
		BaseTemplate: "Test",
	}
	AssertNoError(t, campRepo.Create(ctx, campaign))

	// Create messages with different statuses
	messages := []*models.OutboundMessage{
		{CampaignID: campaign.ID, CustomerID: customer.ID, Status: models.MessageStatusPending, RetryCount: 0},
		{CampaignID: campaign.ID, CustomerID: customer.ID, Status: models.MessageStatusPending, RetryCount: 1},
		{CampaignID: campaign.ID, CustomerID: customer.ID, Status: models.MessageStatusSent, RetryCount: 0},
		{CampaignID: campaign.ID, CustomerID: customer.ID, Status: models.MessageStatusFailed, RetryCount: 3}, // Max retries
		{CampaignID: campaign.ID, CustomerID: customer.ID, Status: models.MessageStatusPending, RetryCount: 2},
	}
	AssertNoError(t, msgRepo.CreateBatch(ctx, messages))

	// Query pending messages (should exclude messages with retry_count >= 3)
	pendingMsgs, err := msgRepo.GetPendingMessages(ctx, 10)
	AssertNoError(t, err)

	// Should get 3 pending messages (retry_count < 3)
	AssertEqual(t, len(pendingMsgs), 3)

	// All should be pending status
	for _, msg := range pendingMsgs {
		AssertEqual(t, msg.Status, models.MessageStatusPending)
		if msg.RetryCount >= 3 {
			t.Errorf("Message with retry_count %d should not be in pending messages", msg.RetryCount)
		}
	}
}

// TestWorker_MultipleChannels tests worker processing for different channels
func TestWorker_MultipleChannels(t *testing.T) {
	_, msgRepo, campRepo, custRepo, templateSvc, senderSvc, cleanup := setupWorkerTest(t)
	defer cleanup()

	ctx := context.Background()
	senderSvc.SetSuccessRate(1.0)

	channels := []models.Channel{models.ChannelSMS, models.ChannelWhatsApp}

	for i, channel := range channels {
		customer := &models.Customer{
			Phone:     "+25470000001" + string(rune('0'+i)),
			FirstName: StringPtr("User"),
		}
		AssertNoError(t, custRepo.Create(ctx, customer))

		campaign := &models.Campaign{
			Name:         "Channel Test " + string(channel),
			Channel:      channel,
			Status:       models.CampaignStatusSending,
			BaseTemplate: "Hi {first_name}",
		}
		AssertNoError(t, campRepo.Create(ctx, campaign))

		message := &models.OutboundMessage{
			CampaignID: campaign.ID,
			CustomerID: customer.ID,
			Status:     models.MessageStatusPending,
			RetryCount: 0,
		}
		AssertNoError(t, msgRepo.CreateBatch(ctx, []*models.OutboundMessage{message}))

		// Process message
		fetchedMsg, _ := msgRepo.GetWithDetails(ctx, message.ID)
		renderedContent, _ := templateSvc.Render(campaign.BaseTemplate, &fetchedMsg.Customer)
		result := senderSvc.Send(fetchedMsg.Campaign.Channel, fetchedMsg.Customer.Phone, renderedContent)

		// Verify success
		AssertEqual(t, result.Success, true)

		// Update status
		err := msgRepo.UpdateStatus(ctx, message.ID, models.MessageStatusSent, nil)
		AssertNoError(t, err)

		updatedMsg, _ := msgRepo.GetByID(ctx, message.ID)
		AssertEqual(t, updatedMsg.Status, models.MessageStatusSent)
	}
}
