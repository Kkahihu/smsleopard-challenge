package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smsleopard/internal/handler"
	"smsleopard/internal/models"
	"smsleopard/internal/repository"
	"smsleopard/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

// setupAPITestHandler creates a campaign handler with mock repositories
func setupAPITestHandler(t *testing.T, db *sql.DB) *handler.CampaignHandler {
	t.Helper()

	campaignRepo := repository.NewCampaignRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	templateSvc := service.NewTemplateService()

	campaignSvc := service.NewCampaignService(
		campaignRepo,
		customerRepo,
		messageRepo,
		templateSvc,
		nil, // No queue publisher needed for these tests
		db,
	)

	return handler.NewCampaignHandler(campaignSvc)
}

// setupAPITestRouter creates a test router with all campaign endpoints
func setupAPITestRouter(campaignHandler *handler.CampaignHandler) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/campaigns", campaignHandler.Create).Methods("POST")
	router.HandleFunc("/campaigns", campaignHandler.List).Methods("GET")
	router.HandleFunc("/campaigns/{id}", campaignHandler.GetByID).Methods("GET")
	router.HandleFunc("/campaigns/{id}/send", campaignHandler.Send).Methods("POST")
	return router
}

// ==================== POST /campaigns Tests ====================

// TestAPI_CreateCampaign_Success tests successful campaign creation
func TestAPI_CreateCampaign_Success(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Mock the INSERT query - only 5 params, RETURNING 3 columns
	mock.ExpectQuery("INSERT INTO campaigns").
		WithArgs(
			"Test Campaign",
			models.ChannelSMS,
			models.CampaignStatusDraft,
			"Hello {first_name}!",
			sqlmock.AnyArg(), // scheduled_at
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, time.Now(), time.Now()))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	requestBody := map[string]interface{}{
		"name":          "Test Campaign",
		"channel":       "sms",
		"base_template": "Hello {first_name}!",
	}
	req := NewJSONRequest(t, "POST", "/campaigns", requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusCreated)
	AssertJSONContentType(t, resp)

	// Parse response
	var result models.Campaign
	ParseJSONResponse(t, resp, &result)

	// Verify campaign created
	AssertEqual(t, result.Name, "Test Campaign")
	AssertEqual(t, result.Channel, models.ChannelSMS)
	AssertEqual(t, result.Status, models.CampaignStatusDraft)
	AssertEqual(t, result.BaseTemplate, "Hello {first_name}!")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_CreateCampaign_Scheduled tests creating a scheduled campaign
func TestAPI_CreateCampaign_Scheduled(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	scheduledAt := time.Now().Add(24 * time.Hour)

	// Mock the INSERT query - only 5 params, RETURNING 3 columns
	mock.ExpectQuery("INSERT INTO campaigns").
		WithArgs(
			"Scheduled Campaign",
			models.ChannelWhatsApp,
			models.CampaignStatusScheduled, // Should be scheduled, not draft
			"Welcome {first_name}!",
			sqlmock.AnyArg(), // scheduled_at
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, time.Now(), time.Now()))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with scheduled_at
	requestBody := map[string]interface{}{
		"name":          "Scheduled Campaign",
		"channel":       "whatsapp",
		"base_template": "Welcome {first_name}!",
		"scheduled_at":  scheduledAt.Format(time.RFC3339),
	}
	req := NewJSONRequest(t, "POST", "/campaigns", requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusCreated)

	var result models.Campaign
	ParseJSONResponse(t, resp, &result)

	// Verify campaign is scheduled
	AssertEqual(t, result.Status, models.CampaignStatusScheduled)
	AssertNotNil(t, result.ScheduledAt)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_CreateCampaign_ValidationErrors tests various validation errors
func TestAPI_CreateCampaign_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		requestBody map[string]interface{}
		expectedMsg string
	}{
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"channel":       "sms",
				"base_template": "Hello!",
			},
			expectedMsg: "name is required",
		},
		{
			name: "missing channel",
			requestBody: map[string]interface{}{
				"name":          "Test",
				"base_template": "Hello!",
			},
			expectedMsg: "invalid channel",
		},
		{
			name: "invalid channel",
			requestBody: map[string]interface{}{
				"name":          "Test",
				"channel":       "email",
				"base_template": "Hello!",
			},
			expectedMsg: "invalid channel",
		},
		{
			name: "missing base_template",
			requestBody: map[string]interface{}{
				"name":    "Test",
				"channel": "sms",
			},
			expectedMsg: "base_template is required",
		},
		{
			name: "empty base_template",
			requestBody: map[string]interface{}{
				"name":          "Test",
				"channel":       "sms",
				"base_template": "",
			},
			expectedMsg: "base_template is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			campaignHandler := setupAPITestHandler(t, db)
			router := setupAPITestRouter(campaignHandler)

			// Create request
			req := NewJSONRequest(t, "POST", "/campaigns", tc.requestBody)

			// Execute request
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify 400 response
			AssertStatusCode(t, resp, http.StatusBadRequest)
			AssertJSONContentType(t, resp)

			// Parse error response
			var errorResp map[string]interface{}
			ParseJSONResponse(t, resp, &errorResp)

			// Verify error structure
			AssertNotNil(t, errorResp["error"])
			errorDetail := errorResp["error"].(map[string]interface{})
			AssertEqual(t, errorDetail["code"], "VALIDATION_ERROR")
			AssertContains(t, errorDetail["message"].(string), tc.expectedMsg)
		})
	}
}

// TestAPI_CreateCampaign_InvalidJSON tests error handling for malformed JSON
func TestAPI_CreateCampaign_InvalidJSON(t *testing.T) {
	// Setup mock DB (won't be queried)
	db, _ := NewMockDB(t)
	defer db.Close()

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with invalid JSON
	req := httptest.NewRequest("POST", "/campaigns", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 400 response
	AssertStatusCode(t, resp, http.StatusBadRequest)

	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	// Verify error code
	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "INVALID_JSON")
}

// ==================== POST /campaigns/{id}/send Tests ====================

// TestAPI_SendCampaign_Success tests successful campaign sending
func TestAPI_SendCampaign_Success(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaign := NewTestCampaignWithStatus(models.CampaignStatusDraft)
	customers := NewTestCustomers(3)
	customerIDs := []int{1, 2, 3}

	// Mock campaign query
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	}).AddRow(
		campaign.ID,
		campaign.Name,
		campaign.Channel,
		campaign.Status,
		campaign.BaseTemplate,
		campaign.ScheduledAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE id").
		WithArgs(campaign.ID).
		WillReturnRows(campaignRows)

	// Mock customers query
	customerRows := sqlmock.NewRows([]string{
		"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
	})
	for _, customer := range customers {
		customerRows.AddRow(
			customer.ID,
			customer.Phone,
			customer.FirstName,
			customer.LastName,
			customer.Location,
			customer.PreferredProduct,
			customer.CreatedAt,
		)
	}
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id = ANY").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(customerRows)

	// Mock transaction for batch message insert
	mock.ExpectBegin()

	// Mock prepare statement for batch insert
	mock.ExpectPrepare("INSERT INTO outbound_messages")

	// Mock each individual insert query (3 customers)
	for i := 1; i <= 3; i++ {
		mock.ExpectQuery("INSERT INTO outbound_messages").
			WithArgs(campaign.ID, i, models.MessageStatusPending, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(i, time.Now(), time.Now()))
	}

	mock.ExpectCommit()

	// Mock campaign status update (separate transaction)
	mock.ExpectExec("UPDATE campaigns SET status").
		WithArgs(models.CampaignStatusSending, campaign.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_ids": customerIDs,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/send", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)
	AssertJSONContentType(t, resp)

	// Parse response
	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify result
	AssertEqual(t, int(result["campaign_id"].(float64)), campaign.ID)
	AssertEqual(t, int(result["messages_queued"].(float64)), 3)
	AssertEqual(t, result["status"], "sending")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_SendCampaign_InvalidCampaignID tests validation for invalid campaign ID
func TestAPI_SendCampaign_InvalidCampaignID(t *testing.T) {
	testCases := []struct {
		name       string
		campaignID string
	}{
		{
			name:       "non-numeric ID",
			campaignID: "invalid",
		},
		{
			name:       "negative ID",
			campaignID: "-1",
		},
		{
			name:       "zero ID",
			campaignID: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			campaignHandler := setupAPITestHandler(t, db)
			router := setupAPITestRouter(campaignHandler)

			// Create request
			requestBody := map[string]interface{}{
				"customer_ids": []int{1, 2},
			}
			req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%s/send", tc.campaignID), requestBody)

			// Execute request
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify 400 response
			AssertStatusCode(t, resp, http.StatusBadRequest)

			var errorResp map[string]interface{}
			ParseJSONResponse(t, resp, &errorResp)

			AssertNotNil(t, errorResp["error"])
			errorDetail := errorResp["error"].(map[string]interface{})
			AssertEqual(t, errorDetail["code"], "VALIDATION_ERROR")
		})
	}
}

// TestAPI_SendCampaign_NoCustomers tests error when no customer IDs provided
func TestAPI_SendCampaign_NoCustomers(t *testing.T) {
	// Setup mock DB (won't be queried)
	db, _ := NewMockDB(t)
	defer db.Close()

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with empty customer_ids
	requestBody := map[string]interface{}{
		"customer_ids": []int{},
	}
	req := NewJSONRequest(t, "POST", "/campaigns/1/send", requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 400 response
	AssertStatusCode(t, resp, http.StatusBadRequest)

	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "VALIDATION_ERROR")
	AssertContains(t, errorDetail["message"].(string), "customer_ids")
}

// TestAPI_SendCampaign_InvalidStatus tests error when campaign status is invalid
func TestAPI_SendCampaign_InvalidStatus(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Campaign with "sent" status (cannot be resent)
	campaign := NewTestCampaignWithStatus(models.CampaignStatusSent)

	// Mock campaign query
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	}).AddRow(
		campaign.ID,
		campaign.Name,
		campaign.Channel,
		campaign.Status,
		campaign.BaseTemplate,
		campaign.ScheduledAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE id").
		WithArgs(campaign.ID).
		WillReturnRows(campaignRows)

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_ids": []int{1, 2},
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/send", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 422 response
	AssertStatusCode(t, resp, http.StatusUnprocessableEntity)

	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "BUSINESS_LOGIC_ERROR")
	AssertContains(t, errorDetail["message"].(string), "cannot be sent")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_SendCampaign_CampaignNotFound tests 404 error for non-existent campaign
func TestAPI_SendCampaign_CampaignNotFound(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	nonExistentID := 999

	// Mock campaign query (not found)
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE id").
		WithArgs(nonExistentID).
		WillReturnError(sql.ErrNoRows)

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_ids": []int{1, 2},
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/send", nonExistentID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 404 response
	AssertStatusCode(t, resp, http.StatusNotFound)

	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "RESOURCE_NOT_FOUND")
	AssertContains(t, errorDetail["message"].(string), "campaign")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// ==================== GET /campaigns Tests ====================

// TestAPI_ListCampaigns_Pagination tests basic pagination
func TestAPI_ListCampaigns_Pagination(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaigns := NewTestCampaigns(5)

	// Mock campaigns query
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	})
	for _, campaign := range campaigns {
		campaignRows.AddRow(
			campaign.ID,
			campaign.Name,
			campaign.Channel,
			campaign.Status,
			campaign.BaseTemplate,
			campaign.ScheduledAt,
			campaign.CreatedAt,
			campaign.UpdatedAt,
		)
	}
	mock.ExpectQuery("SELECT (.+) FROM campaigns").
		WillReturnRows(campaignRows)

	// Mock total count query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with pagination
	req := httptest.NewRequest("GET", "/campaigns?page=1&per_page=20", nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)
	AssertJSONContentType(t, resp)

	// Parse response
	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaigns returned
	AssertNotNil(t, result["campaigns"])
	campaigns_list := result["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns_list), 5)

	// Verify pagination metadata
	AssertNotNil(t, result["pagination"])
	pagination := result["pagination"].(map[string]interface{})
	AssertEqual(t, int(pagination["page"].(float64)), 1)
	AssertEqual(t, int(pagination["page_size"].(float64)), 20)
	AssertEqual(t, int(pagination["total_count"].(float64)), 5)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_ListCampaigns_ChannelFilter tests filtering by channel
func TestAPI_ListCampaigns_ChannelFilter(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Create SMS campaigns only
	campaigns := []*models.Campaign{
		NewTestCampaign(), // SMS by default
		NewTestCampaign(),
	}

	// Mock campaigns query with channel filter
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	})
	for _, campaign := range campaigns {
		campaignRows.AddRow(
			campaign.ID,
			campaign.Name,
			campaign.Channel,
			campaign.Status,
			campaign.BaseTemplate,
			campaign.ScheduledAt,
			campaign.CreatedAt,
			campaign.UpdatedAt,
		)
	}
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE channel").
		WithArgs(models.ChannelSMS, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(campaignRows)

	// Mock total count query
	mock.ExpectQuery("SELECT COUNT(.+) FROM campaigns WHERE channel").
		WithArgs(models.ChannelSMS).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with channel filter
	req := httptest.NewRequest("GET", "/campaigns?channel=sms", nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaigns returned
	campaigns_list := result["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns_list), 2)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_ListCampaigns_StatusFilter tests filtering by status
func TestAPI_ListCampaigns_StatusFilter(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Create draft campaigns
	campaigns := []*models.Campaign{
		NewTestCampaignWithStatus(models.CampaignStatusDraft),
	}

	// Mock campaigns query with status filter
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	})
	for _, campaign := range campaigns {
		campaignRows.AddRow(
			campaign.ID,
			campaign.Name,
			campaign.Channel,
			campaign.Status,
			campaign.BaseTemplate,
			campaign.ScheduledAt,
			campaign.CreatedAt,
			campaign.UpdatedAt,
		)
	}
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE status").
		WithArgs(models.CampaignStatusDraft, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(campaignRows)

	// Mock total count query
	mock.ExpectQuery("SELECT COUNT(.+) FROM campaigns WHERE status").
		WithArgs(models.CampaignStatusDraft).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with status filter
	req := httptest.NewRequest("GET", "/campaigns?status=draft", nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaigns returned
	campaigns_list := result["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns_list), 1)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_ListCampaigns_CombinedFilters tests multiple filters combined
func TestAPI_ListCampaigns_CombinedFilters(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Create campaigns matching filters
	campaigns := []*models.Campaign{
		NewTestCampaignWithStatus(models.CampaignStatusDraft),
	}
	campaigns[0].Channel = models.ChannelSMS

	// Mock campaigns query with combined filters
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	})
	for _, campaign := range campaigns {
		campaignRows.AddRow(
			campaign.ID,
			campaign.Name,
			campaign.Channel,
			campaign.Status,
			campaign.BaseTemplate,
			campaign.ScheduledAt,
			campaign.CreatedAt,
			campaign.UpdatedAt,
		)
	}
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE").
		WithArgs(models.CampaignStatusDraft, models.ChannelSMS, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(campaignRows)

	// Mock total count query
	mock.ExpectQuery("SELECT COUNT(.+) FROM campaigns WHERE").
		WithArgs(models.CampaignStatusDraft, models.ChannelSMS).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request with multiple filters
	req := httptest.NewRequest("GET", "/campaigns?status=draft&channel=sms&page=1", nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaigns returned
	campaigns_list := result["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns_list), 1)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_ListCampaigns_EmptyResults tests response when no campaigns match filters
func TestAPI_ListCampaigns_EmptyResults(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Mock campaigns query (empty result)
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT (.+) FROM campaigns").
		WillReturnRows(campaignRows)

	// Mock total count query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	req := httptest.NewRequest("GET", "/campaigns", nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify empty campaigns list
	campaigns_list := result["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns_list), 0)

	// Verify pagination shows zero
	pagination := result["pagination"].(map[string]interface{})
	AssertEqual(t, int(pagination["total_count"].(float64)), 0)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_ListCampaigns_InvalidFilters tests error handling for invalid filters
func TestAPI_ListCampaigns_InvalidFilters(t *testing.T) {
	testCases := []struct {
		name        string
		queryString string
		expectedMsg string
	}{
		{
			name:        "invalid status",
			queryString: "?status=invalid",
			expectedMsg: "invalid status",
		},
		{
			name:        "invalid channel",
			queryString: "?channel=email",
			expectedMsg: "invalid channel",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			campaignHandler := setupAPITestHandler(t, db)
			router := setupAPITestRouter(campaignHandler)

			// Create request with invalid filter
			req := httptest.NewRequest("GET", "/campaigns"+tc.queryString, nil)

			// Execute request
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify 400 response
			AssertStatusCode(t, resp, http.StatusBadRequest)

			var errorResp map[string]interface{}
			ParseJSONResponse(t, resp, &errorResp)

			AssertNotNil(t, errorResp["error"])
			errorDetail := errorResp["error"].(map[string]interface{})
			AssertEqual(t, errorDetail["code"], "VALIDATION_ERROR")
			AssertContains(t, errorDetail["message"].(string), tc.expectedMsg)
		})
	}
}

// ==================== GET /campaigns/{id} Tests ====================

// TestAPI_GetCampaign_Success tests successful campaign retrieval with stats
func TestAPI_GetCampaign_Success(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaign := NewTestCampaign()

	// Mock campaign with stats query
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
		"total_messages", "pending", "sent", "failed",
	}).AddRow(
		campaign.ID,
		campaign.Name,
		campaign.Channel,
		campaign.Status,
		campaign.BaseTemplate,
		campaign.ScheduledAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
		100, // total_messages
		20,  // pending
		70,  // sent
		10,  // failed
	)
	mock.ExpectQuery("SELECT (.+) FROM campaigns (.+) LEFT JOIN").
		WithArgs(campaign.ID).
		WillReturnRows(campaignRows)

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/campaigns/%d", campaign.ID), nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)
	AssertJSONContentType(t, resp)

	// Parse response
	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaign details
	AssertEqual(t, int(result["id"].(float64)), campaign.ID)
	AssertEqual(t, result["name"], campaign.Name)

	// Verify stats are included
	AssertNotNil(t, result["stats"])
	stats := result["stats"].(map[string]interface{})
	AssertEqual(t, int(stats["total_messages"].(float64)), 100)
	AssertEqual(t, int(stats["pending"].(float64)), 20)
	AssertEqual(t, int(stats["sent"].(float64)), 70)
	AssertEqual(t, int(stats["failed"].(float64)), 10)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_GetCampaign_NotFound tests 404 error for non-existent campaign
func TestAPI_GetCampaign_NotFound(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	nonExistentID := 999

	// Mock campaign query (not found)
	mock.ExpectQuery("SELECT (.+) FROM campaigns (.+) LEFT JOIN").
		WithArgs(nonExistentID).
		WillReturnError(sql.ErrNoRows)

	// Setup handler and router
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/campaigns/%d", nonExistentID), nil)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 404 response
	AssertStatusCode(t, resp, http.StatusNotFound)

	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "RESOURCE_NOT_FOUND")
	AssertContains(t, errorDetail["message"].(string), "campaign")
	AssertContains(t, errorDetail["message"].(string), "999")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestAPI_GetCampaign_InvalidIDFormat tests validation for invalid ID format
func TestAPI_GetCampaign_InvalidIDFormat(t *testing.T) {
	testCases := []struct {
		name string
		id   string
	}{
		{
			name: "non-numeric ID",
			id:   "invalid",
		},
		{
			name: "negative ID",
			id:   "-1",
		},
		{
			name: "zero ID",
			id:   "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			campaignHandler := setupAPITestHandler(t, db)
			router := setupAPITestRouter(campaignHandler)

			// Create request
			req := httptest.NewRequest("GET", fmt.Sprintf("/campaigns/%s", tc.id), nil)

			// Execute request
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify 400 response
			AssertStatusCode(t, resp, http.StatusBadRequest)

			var errorResp map[string]interface{}
			ParseJSONResponse(t, resp, &errorResp)

			AssertNotNil(t, errorResp["error"])
			errorDetail := errorResp["error"].(map[string]interface{})
			AssertEqual(t, errorDetail["code"], "VALIDATION_ERROR")
		})
	}
}

// ==================== Integration Tests ====================

// TestAPI_Integration tests full API workflow with real database (if available)
func TestAPI_Integration(t *testing.T) {
	// Setup test database
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return
	}
	defer db.Close()
	defer CleanupTestDB(t, db)

	// Create test repository for customers
	customerRepo := repository.NewCustomerRepository(db)

	// Create test customers
	customers := NewTestCustomers(3)
	for _, customer := range customers {
		err := customerRepo.Create(context.Background(), customer)
		AssertNoError(t, err)
	}

	// Setup handler
	campaignHandler := setupAPITestHandler(t, db)
	router := setupAPITestRouter(campaignHandler)

	// Test 1: Create a campaign
	createReq := map[string]interface{}{
		"name":          "Integration Test Campaign",
		"channel":       "sms",
		"base_template": "Hello {first_name}!",
	}
	req := NewJSONRequest(t, "POST", "/campaigns", createReq)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	AssertStatusCode(t, resp, http.StatusCreated)

	var createdCampaign models.Campaign
	ParseJSONResponse(t, resp, &createdCampaign)
	campaignID := createdCampaign.ID

	// Test 2: List campaigns
	req = httptest.NewRequest("GET", "/campaigns", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	AssertStatusCode(t, resp, http.StatusOK)

	var listResp map[string]interface{}
	ParseJSONResponse(t, resp, &listResp)
	campaigns := listResp["campaigns"].([]interface{})
	AssertEqual(t, len(campaigns) >= 1, true)

	// Test 3: Get campaign by ID
	req = httptest.NewRequest("GET", fmt.Sprintf("/campaigns/%d", campaignID), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	AssertStatusCode(t, resp, http.StatusOK)

	var getCampaign map[string]interface{}
	ParseJSONResponse(t, resp, &getCampaign)
	AssertEqual(t, int(getCampaign["id"].(float64)), campaignID)

	// Test 4: Send campaign
	sendReq := map[string]interface{}{
		"customer_ids": []int{customers[0].ID, customers[1].ID},
	}
	req = NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/send", campaignID), sendReq)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	AssertStatusCode(t, resp, http.StatusOK)

	var sendResp map[string]interface{}
	ParseJSONResponse(t, resp, &sendResp)
	AssertEqual(t, int(sendResp["messages_queued"].(float64)), 2)
}
