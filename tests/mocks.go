package tests

import (
	"context"
	"smsleopard/internal/models"
	"smsleopard/internal/repository"
	"time"
)

// MockCustomerRepository mocks CustomerRepository
type MockCustomerRepository struct {
	CreateFunc   func(ctx context.Context, customer *models.Customer) error
	GetByIDFunc  func(ctx context.Context, id int) (*models.Customer, error)
	GetByIDsFunc func(ctx context.Context, ids []int) ([]*models.Customer, error)
	ListFunc     func(ctx context.Context, limit, offset int) ([]*models.Customer, error)
	UpdateFunc   func(ctx context.Context, customer *models.Customer) error
	DeleteFunc   func(ctx context.Context, id int) error

	Calls map[string]int // Track method calls
}

func NewMockCustomerRepository() *MockCustomerRepository {
	return &MockCustomerRepository{
		Calls: make(map[string]int),
	}
}

func (m *MockCustomerRepository) Create(ctx context.Context, customer *models.Customer) error {
	m.Calls["Create"]++
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, customer)
	}
	customer.ID = 1
	customer.CreatedAt = time.Now()
	return nil
}

func (m *MockCustomerRepository) GetByID(ctx context.Context, id int) (*models.Customer, error) {
	m.Calls["GetByID"]++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return NewTestCustomer(), nil
}

func (m *MockCustomerRepository) GetByIDs(ctx context.Context, ids []int) ([]*models.Customer, error) {
	m.Calls["GetByIDs"]++
	if m.GetByIDsFunc != nil {
		return m.GetByIDsFunc(ctx, ids)
	}
	customers := make([]*models.Customer, len(ids))
	for i, id := range ids {
		customers[i] = NewTestCustomerWithID(id)
	}
	return customers, nil
}

func (m *MockCustomerRepository) List(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	m.Calls["List"]++
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset)
	}
	return NewTestCustomers(limit), nil
}

func (m *MockCustomerRepository) Update(ctx context.Context, customer *models.Customer) error {
	m.Calls["Update"]++
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, customer)
	}
	return nil
}

func (m *MockCustomerRepository) Delete(ctx context.Context, id int) error {
	m.Calls["Delete"]++
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockCampaignRepository mocks CampaignRepository
type MockCampaignRepository struct {
	CreateFunc       func(ctx context.Context, campaign *models.Campaign) error
	GetByIDFunc      func(ctx context.Context, id int) (*models.Campaign, error)
	GetWithStatsFunc func(ctx context.Context, id int) (*models.CampaignWithStats, error)
	ListFunc         func(ctx context.Context, filters repository.CampaignFilters) ([]*models.Campaign, int, error)
	UpdateStatusFunc func(ctx context.Context, id int, status models.CampaignStatus) error
	DeleteFunc       func(ctx context.Context, id int) error

	Calls map[string]int
}

func NewMockCampaignRepository() *MockCampaignRepository {
	return &MockCampaignRepository{
		Calls: make(map[string]int),
	}
}

func (m *MockCampaignRepository) Create(ctx context.Context, campaign *models.Campaign) error {
	m.Calls["Create"]++
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, campaign)
	}
	campaign.ID = 1
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	return nil
}

func (m *MockCampaignRepository) GetByID(ctx context.Context, id int) (*models.Campaign, error) {
	m.Calls["GetByID"]++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return NewTestCampaign(), nil
}

func (m *MockCampaignRepository) GetWithStats(ctx context.Context, id int) (*models.CampaignWithStats, error) {
	m.Calls["GetWithStats"]++
	if m.GetWithStatsFunc != nil {
		return m.GetWithStatsFunc(ctx, id)
	}
	campaign := NewTestCampaign()
	return &models.CampaignWithStats{
		Campaign: *campaign,
		Stats: models.CampaignStats{
			Total:   10,
			Pending: 0,
			Sent:    8,
			Failed:  2,
		},
	}, nil
}

func (m *MockCampaignRepository) List(ctx context.Context, filters repository.CampaignFilters) ([]*models.Campaign, int, error) {
	m.Calls["List"]++
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filters)
	}
	campaigns := NewTestCampaigns(filters.PageSize)
	return campaigns, len(campaigns), nil
}

func (m *MockCampaignRepository) UpdateStatus(ctx context.Context, id int, status models.CampaignStatus) error {
	m.Calls["UpdateStatus"]++
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockCampaignRepository) Delete(ctx context.Context, id int) error {
	m.Calls["Delete"]++
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockMessageRepository mocks MessageRepository
type MockMessageRepository struct {
	CreateFunc             func(ctx context.Context, message *models.OutboundMessage) error
	CreateBatchFunc        func(ctx context.Context, messages []*models.OutboundMessage) error
	GetByIDFunc            func(ctx context.Context, id int) (*models.OutboundMessage, error)
	GetWithDetailsFunc     func(ctx context.Context, id int) (*models.OutboundMessageWithDetails, error)
	UpdateStatusFunc       func(ctx context.Context, id int, status models.MessageStatus, lastError *string) error
	GetPendingMessagesFunc func(ctx context.Context, limit int) ([]*models.OutboundMessage, error)
	GetByCampaignIDFunc    func(ctx context.Context, campaignID int) ([]*models.OutboundMessage, error)

	Calls map[string]int
}

func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		Calls: make(map[string]int),
	}
}

func (m *MockMessageRepository) Create(ctx context.Context, message *models.OutboundMessage) error {
	m.Calls["Create"]++
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, message)
	}
	message.ID = 1
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	return nil
}

func (m *MockMessageRepository) CreateBatch(ctx context.Context, messages []*models.OutboundMessage) error {
	m.Calls["CreateBatch"]++
	if m.CreateBatchFunc != nil {
		return m.CreateBatchFunc(ctx, messages)
	}
	for i, msg := range messages {
		msg.ID = i + 1
		msg.CreatedAt = time.Now()
		msg.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id int) (*models.OutboundMessage, error) {
	m.Calls["GetByID"]++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return NewTestMessage(1, 1), nil
}

func (m *MockMessageRepository) GetWithDetails(ctx context.Context, id int) (*models.OutboundMessageWithDetails, error) {
	m.Calls["GetWithDetails"]++
	if m.GetWithDetailsFunc != nil {
		return m.GetWithDetailsFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockMessageRepository) UpdateStatus(ctx context.Context, id int, status models.MessageStatus, lastError *string) error {
	m.Calls["UpdateStatus"]++
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status, lastError)
	}
	return nil
}

func (m *MockMessageRepository) GetPendingMessages(ctx context.Context, limit int) ([]*models.OutboundMessage, error) {
	m.Calls["GetPendingMessages"]++
	if m.GetPendingMessagesFunc != nil {
		return m.GetPendingMessagesFunc(ctx, limit)
	}
	return []*models.OutboundMessage{}, nil
}

func (m *MockMessageRepository) GetByCampaignID(ctx context.Context, campaignID int) ([]*models.OutboundMessage, error) {
	m.Calls["GetByCampaignID"]++
	if m.GetByCampaignIDFunc != nil {
		return m.GetByCampaignIDFunc(ctx, campaignID)
	}
	return NewTestMessages(campaignID, []int{1, 2, 3}), nil
}

// MockPublisher mocks queue.Publisher
type MockPublisher struct {
	PublishMessageFunc func(messageID, campaignID, customerID int) error
	Published          []PublishedJob
}

type PublishedJob struct {
	MessageID  int
	CampaignID int
	CustomerID int
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		Published: []PublishedJob{},
	}
}

func (m *MockPublisher) PublishMessage(messageID, campaignID, customerID int) error {
	if m.PublishMessageFunc != nil {
		return m.PublishMessageFunc(messageID, campaignID, customerID)
	}
	m.Published = append(m.Published, PublishedJob{
		MessageID:  messageID,
		CampaignID: campaignID,
		CustomerID: customerID,
	})
	return nil
}

func (m *MockPublisher) GetPublishedCount() int {
	return len(m.Published)
}

func (m *MockPublisher) Reset() {
	m.Published = []PublishedJob{}
}
