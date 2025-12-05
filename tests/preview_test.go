package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"smsleopard/internal/handler"
	"smsleopard/internal/models"
	"smsleopard/internal/repository"
	"smsleopard/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

// setupPreviewTestHandler creates a test handler with mock repositories
func setupPreviewTestHandler(t *testing.T, db *sql.DB) *handler.PreviewHandler {
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
		nil, // No queue publisher needed for preview
		db,
	)

	return handler.NewPreviewHandler(campaignSvc)
}

// setupPreviewTestRouter creates a test router with the preview endpoint
func setupPreviewTestRouter(previewHandler *handler.PreviewHandler) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/campaigns/{id}/personalized-preview", previewHandler.Preview).Methods("POST")
	return router
}

// TestPreviewEndpoint_Success tests successful preview rendering with different customers
func TestPreviewEndpoint_Success(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	// Test data
	campaign := NewTestCampaign()
	customer := NewTestCustomer()

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

	// Mock customer query
	customerRows := sqlmock.NewRows([]string{
		"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
	}).AddRow(
		customer.ID,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.CreatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
		WithArgs(customer.ID).
		WillReturnRows(customerRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_id": customer.ID,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)
	AssertJSONContentType(t, resp)

	// Parse response
	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify rendered message
	AssertNotNil(t, result["rendered_message"])
	renderedMsg := result["rendered_message"].(string)
	AssertContains(t, renderedMsg, "John")         // first_name
	AssertContains(t, renderedMsg, "Premium Plan") // preferred_product

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_DifferentCustomers tests preview with various customer data
func TestPreviewEndpoint_DifferentCustomers(t *testing.T) {
	testCases := []struct {
		name     string
		customer *models.Customer
		template string
		expected string
	}{
		{
			name:     "customer with all fields",
			customer: NewTestCustomer(),
			template: "Hello {first_name} {last_name} from {location}!",
			expected: "Hello John Doe from Nairobi!",
		},
		{
			name:     "customer with null fields",
			customer: NewTestCustomerNullFields(),
			template: "Hi {first_name}, welcome!",
			expected: "Hi , welcome!",
		},
		{
			name: "customer with partial fields",
			customer: &models.Customer{
				ID:               3,
				Phone:            "+254700000003",
				FirstName:        StringPtr("Alice"),
				LastName:         nil,
				Location:         StringPtr("Mombasa"),
				PreferredProduct: nil,
			},
			template: "Hello {first_name} from {location}!",
			expected: "Hello Alice from Mombasa!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock := NewMockDB(t)
			defer db.Close()

			campaign := NewTestCampaignWithTemplate(tc.template)

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

			// Mock customer query
			customerRows := sqlmock.NewRows([]string{
				"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
			}).AddRow(
				tc.customer.ID,
				tc.customer.Phone,
				tc.customer.FirstName,
				tc.customer.LastName,
				tc.customer.Location,
				tc.customer.PreferredProduct,
				tc.customer.CreatedAt,
			)
			mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
				WithArgs(tc.customer.ID).
				WillReturnRows(customerRows)

			// Setup handler and router
			previewHandler := setupPreviewTestHandler(t, db)
			router := setupPreviewTestRouter(previewHandler)

			// Create request
			requestBody := map[string]interface{}{
				"customer_id": tc.customer.ID,
			}
			req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

			// Execute request
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify response
			AssertStatusCode(t, resp, http.StatusOK)

			var result map[string]interface{}
			ParseJSONResponse(t, resp, &result)

			renderedMsg := result["rendered_message"].(string)
			AssertEqual(t, renderedMsg, tc.expected)

			// Verify expectations met
			AssertNoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestPreviewEndpoint_OverrideTemplate tests custom template override functionality
func TestPreviewEndpoint_OverrideTemplate(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaign := NewTestCampaign()
	customer := NewTestCustomer()
	overrideTemplate := "Custom template: {first_name} at {phone}"

	// Mock campaign query
	campaignRows := sqlmock.NewRows([]string{
		"id", "name", "channel", "status", "base_template", "scheduled_at", "created_at", "updated_at",
	}).AddRow(
		campaign.ID,
		campaign.Name,
		campaign.Channel,
		campaign.Status,
		campaign.BaseTemplate, // Original template (should be ignored)
		campaign.ScheduledAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE id").
		WithArgs(campaign.ID).
		WillReturnRows(campaignRows)

	// Mock customer query
	customerRows := sqlmock.NewRows([]string{
		"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
	}).AddRow(
		customer.ID,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.CreatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
		WithArgs(customer.ID).
		WillReturnRows(customerRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request with override_template
	requestBody := map[string]interface{}{
		"customer_id":       customer.ID,
		"override_template": overrideTemplate,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify override template was used
	renderedMsg := result["rendered_message"].(string)
	expectedMsg := "Custom template: John at +254700000001"
	AssertEqual(t, renderedMsg, expectedMsg)

	// Verify used_template field shows override template
	usedTemplate := result["used_template"].(string)
	AssertEqual(t, usedTemplate, overrideTemplate)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_MissingCustomer tests 404 error for non-existent customer
func TestPreviewEndpoint_MissingCustomer(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaign := NewTestCampaign()
	nonExistentCustomerID := 999

	// Mock campaign query (campaign exists)
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

	// Mock customer query (customer not found)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
		WithArgs(nonExistentCustomerID).
		WillReturnError(sql.ErrNoRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_id": nonExistentCustomerID,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 404 response
	AssertStatusCode(t, resp, http.StatusNotFound)
	AssertJSONContentType(t, resp)

	// Parse error response
	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	// Verify error structure
	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "RESOURCE_NOT_FOUND")
	AssertContains(t, errorDetail["message"].(string), "customer")
	AssertContains(t, errorDetail["message"].(string), "999")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_MissingCampaign tests 404 error for non-existent campaign
func TestPreviewEndpoint_MissingCampaign(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	nonExistentCampaignID := 888
	customerID := 1

	// Mock campaign query (campaign not found)
	mock.ExpectQuery("SELECT (.+) FROM campaigns WHERE id").
		WithArgs(nonExistentCampaignID).
		WillReturnError(sql.ErrNoRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_id": customerID,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", nonExistentCampaignID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 404 response
	AssertStatusCode(t, resp, http.StatusNotFound)
	AssertJSONContentType(t, resp)

	// Parse error response
	var errorResp map[string]interface{}
	ParseJSONResponse(t, resp, &errorResp)

	// Verify error structure
	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "RESOURCE_NOT_FOUND")
	AssertContains(t, errorDetail["message"].(string), "campaign")
	AssertContains(t, errorDetail["message"].(string), "888")

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_InvalidCampaignID tests validation error for invalid campaign ID
func TestPreviewEndpoint_InvalidCampaignID(t *testing.T) {
	testCases := []struct {
		name       string
		campaignID string
	}{
		{
			name:       "non-numeric campaign ID",
			campaignID: "invalid",
		},
		{
			name:       "negative campaign ID",
			campaignID: "-1",
		},
		{
			name:       "zero campaign ID",
			campaignID: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			previewHandler := setupPreviewTestHandler(t, db)
			router := setupPreviewTestRouter(previewHandler)

			// Create request
			requestBody := map[string]interface{}{
				"customer_id": 1,
			}
			req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%s/personalized-preview", tc.campaignID), requestBody)

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
		})
	}
}

// TestPreviewEndpoint_MissingCustomerID tests validation error for missing customer_id
func TestPreviewEndpoint_MissingCustomerID(t *testing.T) {
	// Setup mock DB (won't be queried)
	db, _ := NewMockDB(t)
	defer db.Close()

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request without customer_id
	requestBody := map[string]interface{}{}
	req := NewJSONRequest(t, "POST", "/campaigns/1/personalized-preview", requestBody)

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
	AssertContains(t, errorDetail["message"].(string), "customer_id")
}

// TestPreviewEndpoint_InvalidCustomerID tests validation error for invalid customer_id
func TestPreviewEndpoint_InvalidCustomerID(t *testing.T) {
	testCases := []struct {
		name       string
		customerID interface{}
	}{
		{
			name:       "negative customer_id",
			customerID: -1,
		},
		{
			name:       "zero customer_id",
			customerID: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB (won't be queried)
			db, _ := NewMockDB(t)
			defer db.Close()

			// Setup handler and router
			previewHandler := setupPreviewTestHandler(t, db)
			router := setupPreviewTestRouter(previewHandler)

			// Create request with invalid customer_id
			requestBody := map[string]interface{}{
				"customer_id": tc.customerID,
			}
			req := NewJSONRequest(t, "POST", "/campaigns/1/personalized-preview", requestBody)

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
			AssertContains(t, errorDetail["message"].(string), "customer_id")
		})
	}
}

// TestPreviewEndpoint_InvalidJSONBody tests error handling for malformed JSON
func TestPreviewEndpoint_InvalidJSONBody(t *testing.T) {
	// Setup mock DB (won't be queried)
	db, _ := NewMockDB(t)
	defer db.Close()

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request with invalid JSON
	req := httptest.NewRequest("POST", "/campaigns/1/personalized-preview", nil)
	req.Header.Set("Content-Type", "application/json")
	// Set body to invalid JSON
	req.Body = http.NoBody

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify 400 response
	AssertStatusCode(t, resp, http.StatusBadRequest)
	AssertJSONContentType(t, resp)

	// Parse error response
	var errorResp map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&errorResp)
	AssertNoError(t, err)

	// Verify error structure
	AssertNotNil(t, errorResp["error"])
	errorDetail := errorResp["error"].(map[string]interface{})
	AssertEqual(t, errorDetail["code"], "INVALID_JSON")
}

// TestPreviewEndpoint_EmptyOverrideTemplate tests with empty override template string
func TestPreviewEndpoint_EmptyOverrideTemplate(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	campaign := NewTestCampaign()
	customer := NewTestCustomer()

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

	// Mock customer query
	customerRows := sqlmock.NewRows([]string{
		"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
	}).AddRow(
		customer.ID,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.CreatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
		WithArgs(customer.ID).
		WillReturnRows(customerRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request with empty override_template (should use campaign template)
	emptyTemplate := ""
	requestBody := map[string]interface{}{
		"customer_id":       customer.ID,
		"override_template": emptyTemplate,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response (should use campaign template since override is empty)
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify campaign template was used (not the empty override)
	usedTemplate := result["used_template"].(string)
	AssertEqual(t, usedTemplate, campaign.BaseTemplate)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_ComplexTemplate tests with complex template containing multiple placeholders
func TestPreviewEndpoint_ComplexTemplate(t *testing.T) {
	// Setup mock DB
	db, mock := NewMockDB(t)
	defer db.Close()

	complexTemplate := "Dear {first_name} {last_name}, we're reaching out to you in {location} about our {preferred_product}. Contact us at {phone}."
	campaign := NewTestCampaignWithTemplate(complexTemplate)
	customer := NewTestCustomer()

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

	// Mock customer query
	customerRows := sqlmock.NewRows([]string{
		"id", "phone", "first_name", "last_name", "location", "preferred_product", "created_at",
	}).AddRow(
		customer.ID,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.CreatedAt,
	)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id").
		WithArgs(customer.ID).
		WillReturnRows(customerRows)

	// Setup handler and router
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_id": customer.ID,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify all placeholders were replaced correctly
	renderedMsg := result["rendered_message"].(string)
	expectedMsg := "Dear John Doe, we're reaching out to you in Nairobi about our Premium Plan. Contact us at +254700000001."
	AssertEqual(t, renderedMsg, expectedMsg)

	// Verify expectations met
	AssertNoError(t, mock.ExpectationsWereMet())
}

// TestPreviewEndpoint_Integration tests with real database (if available)
func TestPreviewEndpoint_Integration(t *testing.T) {
	// Setup test database
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return
	}
	defer db.Close()
	defer CleanupTestDB(t, db)

	// Create test repositories
	campaignRepo := repository.NewCampaignRepository(db)
	customerRepo := repository.NewCustomerRepository(db)

	// Create test customer
	customer := NewTestCustomer()
	err := customerRepo.Create(context.Background(), customer)
	AssertNoError(t, err)

	// Create test campaign
	campaign := NewTestCampaign()
	err = campaignRepo.Create(context.Background(), campaign)
	AssertNoError(t, err)

	// Setup handler
	previewHandler := setupPreviewTestHandler(t, db)
	router := setupPreviewTestRouter(previewHandler)

	// Create request
	requestBody := map[string]interface{}{
		"customer_id": customer.ID,
	}
	req := NewJSONRequest(t, "POST", fmt.Sprintf("/campaigns/%d/personalized-preview", campaign.ID), requestBody)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	AssertStatusCode(t, resp, http.StatusOK)

	var result map[string]interface{}
	ParseJSONResponse(t, resp, &result)

	// Verify rendered message
	AssertNotNil(t, result["rendered_message"])
	renderedMsg := result["rendered_message"].(string)
	AssertContains(t, renderedMsg, "John")
}
