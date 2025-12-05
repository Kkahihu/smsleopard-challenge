package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"smsleopard/internal/models"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// AssertError checks if error matches expected
func AssertError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error %q but got nil", expected)
		return
	}
	if err.Error() != expected {
		t.Errorf("Expected error %q but got %q", expected, err.Error())
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %v but got %v", want, got)
	}
}

// AssertNotNil checks if value is not nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Error("Expected non-nil value but got nil")
	}
}

// AssertNil checks if value is nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Errorf("Expected nil but got %v", value)
	}
}

// AssertContains checks if string contains substring
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected %q to contain %q", haystack, needle)
	}
}

// NewMockDB creates a mock database for testing
func NewMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	return db, mock
}

// SetupTestDB creates a test database connection (integration tests)
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://smsleopard:smsleopard@localhost:5432/smsleopard_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: test database not available: %v", err)
		return nil
	}

	return db
}

// CleanupTestDB cleans up test data from database
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{"outbound_messages", "campaigns", "customers"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// NewJSONRequest creates an HTTP request with JSON body
func NewJSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// ParseJSONResponse parses JSON response body
func ParseJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}
}

// AssertStatusCode checks HTTP response status code
func AssertStatusCode(t *testing.T, resp *httptest.ResponseRecorder, want int) {
	t.Helper()
	if resp.Code != want {
		t.Errorf("Expected status code %d but got %d", want, resp.Code)
	}
}

// AssertJSONContentType checks Content-Type header
func AssertJSONContentType(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	contentType := resp.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json but got %s", contentType)
	}
}

// NewTestCustomer creates a test customer with all fields populated
func NewTestCustomer() *models.Customer {
	firstName := "John"
	lastName := "Doe"
	location := "Nairobi"
	product := "Premium Plan"

	return &models.Customer{
		ID:               1,
		Phone:            "+254700000001",
		FirstName:        &firstName,
		LastName:         &lastName,
		Location:         &location,
		PreferredProduct: &product,
		CreatedAt:        time.Now(),
	}
}

// NewTestCustomerWithID creates a customer with specific ID
func NewTestCustomerWithID(id int) *models.Customer {
	customer := NewTestCustomer()
	customer.ID = id
	customer.Phone = fmt.Sprintf("+25470000%04d", id)
	return customer
}

// NewTestCustomerNullFields creates a customer with nil optional fields
func NewTestCustomerNullFields() *models.Customer {
	return &models.Customer{
		ID:               1,
		Phone:            "+254700000001",
		FirstName:        nil,
		LastName:         nil,
		Location:         nil,
		PreferredProduct: nil,
		CreatedAt:        time.Now(),
	}
}

// NewTestCustomers creates multiple test customers
func NewTestCustomers(count int) []*models.Customer {
	customers := make([]*models.Customer, count)
	for i := 0; i < count; i++ {
		firstName := fmt.Sprintf("Customer%d", i+1)
		lastName := "Test"
		location := "Nairobi"
		product := "Product" + string(rune('A'+i%3))

		customers[i] = &models.Customer{
			ID:               i + 1,
			Phone:            fmt.Sprintf("+25470%07d", i+1),
			FirstName:        &firstName,
			LastName:         &lastName,
			Location:         &location,
			PreferredProduct: &product,
			CreatedAt:        time.Now(),
		}
	}
	return customers
}

// NewTestCampaign creates a test campaign
func NewTestCampaign() *models.Campaign {
	return &models.Campaign{
		ID:           1,
		Name:         "Test Campaign",
		Channel:      models.ChannelSMS,
		Status:       models.CampaignStatusDraft,
		BaseTemplate: "Hello {first_name}, welcome to {preferred_product}!",
		ScheduledAt:  nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// NewTestCampaignWithStatus creates a campaign with specific status
func NewTestCampaignWithStatus(status models.CampaignStatus) *models.Campaign {
	campaign := NewTestCampaign()
	campaign.Status = status
	return campaign
}

// NewTestCampaignWithTemplate creates a campaign with custom template
func NewTestCampaignWithTemplate(template string) *models.Campaign {
	campaign := NewTestCampaign()
	campaign.BaseTemplate = template
	return campaign
}

// NewTestCampaigns creates multiple test campaigns (for pagination tests)
func NewTestCampaigns(count int) []*models.Campaign {
	campaigns := make([]*models.Campaign, count)
	statuses := []models.CampaignStatus{
		models.CampaignStatusDraft,
		models.CampaignStatusScheduled,
		models.CampaignStatusSending,
		models.CampaignStatusSent,
	}
	channels := []models.Channel{models.ChannelSMS, models.ChannelWhatsApp}

	for i := 0; i < count; i++ {
		campaigns[i] = &models.Campaign{
			ID:           i + 1,
			Name:         fmt.Sprintf("Campaign %d", i+1),
			Channel:      channels[i%2],
			Status:       statuses[i%len(statuses)],
			BaseTemplate: "Test template",
			CreatedAt:    time.Now().Add(-time.Duration(count-i) * time.Hour),
			UpdatedAt:    time.Now().Add(-time.Duration(count-i) * time.Hour),
		}
	}
	return campaigns
}

// NewTestMessage creates a test outbound message
func NewTestMessage(campaignID, customerID int) *models.OutboundMessage {
	content := "Hello John, welcome to Premium Plan!"
	return &models.OutboundMessage{
		ID:              1,
		CampaignID:      campaignID,
		CustomerID:      customerID,
		Status:          models.MessageStatusPending,
		RenderedContent: &content,
		LastError:       nil,
		RetryCount:      0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// NewTestMessageWithStatus creates a message with specific status
func NewTestMessageWithStatus(status models.MessageStatus) *models.OutboundMessage {
	msg := NewTestMessage(1, 1)
	msg.Status = status
	return msg
}

// NewTestMessages creates multiple test messages
func NewTestMessages(campaignID int, customerIDs []int) []*models.OutboundMessage {
	messages := make([]*models.OutboundMessage, len(customerIDs))
	for i, customerID := range customerIDs {
		messages[i] = NewTestMessage(campaignID, customerID)
		messages[i].ID = i + 1
	}
	return messages
}

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}
