package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"smsleopard/internal/models"
	"smsleopard/internal/repository"
	"smsleopard/internal/service"

	"github.com/gorilla/mux"
)

// CampaignHandler handles HTTP requests for campaign operations
type CampaignHandler struct {
	campaignService *service.CampaignService
}

// NewCampaignHandler creates a new campaign handler
func NewCampaignHandler(campaignService *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
	}
}

// Create handles POST /campaigns - creates a new campaign
func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateCampaignRequest

	// Parse JSON body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is empty")
			return
		}
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format")
		return
	}

	// Call service to create campaign
	campaign, err := h.campaignService.CreateCampaign(r.Context(), &req)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	// Return 201 Created
	WriteCreated(w, campaign)
}

// List handles GET /campaigns - lists campaigns with filters
func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse pagination parameters
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	if perPageStr := query.Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
		}
	}

	// Validate per_page max 100
	if perPage > 100 {
		perPage = 100
	}

	// Build filters
	filters := repository.CampaignFilters{
		Page:     page,
		PageSize: perPage,
	}

	// Parse status filter
	if statusStr := query.Get("status"); statusStr != "" {
		// Validate status
		validStatuses := map[string]models.CampaignStatus{
			"draft":     models.CampaignStatusDraft,
			"scheduled": models.CampaignStatusScheduled,
			"sending":   models.CampaignStatusSending,
			"sent":      models.CampaignStatusSent,
			"failed":    models.CampaignStatusFailed,
		}
		if status, ok := validStatuses[statusStr]; ok {
			filters.Status = &status
		} else {
			WriteValidationError(w, "invalid status: must be one of draft, scheduled, sending, sent, failed")
			return
		}
	}

	// Parse channel filter
	if channelStr := query.Get("channel"); channelStr != "" {
		// Validate channel
		validChannels := map[string]models.Channel{
			"sms":      models.ChannelSMS,
			"whatsapp": models.ChannelWhatsApp,
		}
		if channel, ok := validChannels[channelStr]; ok {
			filters.Channel = &channel
		} else {
			WriteValidationError(w, "invalid channel: must be 'sms' or 'whatsapp'")
			return
		}
	}

	// Call service to list campaigns
	campaigns, pagination, err := h.campaignService.ListCampaigns(r.Context(), filters)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	// Create response
	response := ListCampaignsResponse{
		Campaigns:  campaigns,
		Pagination: pagination,
	}

	// Return 200 OK
	WriteOK(w, response)
}

// GetByID handles GET /campaigns/{id} - gets a campaign by ID
func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Convert to integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		WriteValidationError(w, "invalid campaign ID format")
		return
	}

	// Validate ID > 0
	if id <= 0 {
		WriteValidationError(w, "campaign ID must be greater than 0")
		return
	}

	// Call service to get campaign with stats
	campaign, err := h.campaignService.GetCampaignWithStats(r.Context(), id)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	// Return 200 OK
	WriteOK(w, campaign)
}

// Send handles POST /campaigns/{id}/send - sends a campaign to customers
func (h *CampaignHandler) Send(w http.ResponseWriter, r *http.Request) {
	// Extract campaign ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Convert to integer
	campaignID, err := strconv.Atoi(idStr)
	if err != nil {
		WriteValidationError(w, "invalid campaign ID format")
		return
	}

	// Validate ID > 0
	if campaignID <= 0 {
		WriteValidationError(w, "campaign ID must be greater than 0")
		return
	}

	// Parse JSON body
	var req SendCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is empty")
			return
		}
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format")
		return
	}

	// Validate customer_ids not empty
	if len(req.CustomerIDs) == 0 {
		WriteValidationError(w, "customer_ids cannot be empty")
		return
	}

	// Call service to send campaign
	result, err := h.campaignService.SendCampaign(r.Context(), campaignID, req.CustomerIDs)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	// Return 200 OK
	WriteOK(w, result)
}

// Request/Response types

// ListCampaignsResponse represents the response for listing campaigns
type ListCampaignsResponse struct {
	Campaigns  []*models.Campaign      `json:"campaigns"`
	Pagination *service.PaginationInfo `json:"pagination"`
}

// SendCampaignRequest represents the request to send a campaign
type SendCampaignRequest struct {
	CustomerIDs []int `json:"customer_ids"`
}
