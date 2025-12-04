package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"smsleopard/internal/service"

	"github.com/gorilla/mux"
)

// PreviewHandler handles HTTP requests for message preview functionality
type PreviewHandler struct {
	campaignService *service.CampaignService
}

// NewPreviewHandler creates a new PreviewHandler instance
func NewPreviewHandler(campaignService *service.CampaignService) *PreviewHandler {
	return &PreviewHandler{
		campaignService: campaignService,
	}
}

// PreviewRequest represents the request body for message preview
type PreviewRequest struct {
	CustomerID       int     `json:"customer_id"`
	OverrideTemplate *string `json:"override_template,omitempty"`
}

// Preview handles POST /campaigns/{id}/personalized-preview
// It previews how a message will render for a specific customer
func (h *PreviewHandler) Preview(w http.ResponseWriter, r *http.Request) {
	// Extract campaign ID from URL
	campaignIDStr := mux.Vars(r)["id"]

	// Convert campaign ID to integer and validate
	campaignID, err := strconv.Atoi(campaignIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid campaign ID")
		return
	}

	if campaignID <= 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "campaign ID must be positive")
		return
	}

	// Parse JSON body
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	// Validate customer_id
	if req.CustomerID <= 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "customer_id is required and must be positive")
		return
	}

	// Build service request
	previewReq := &service.PreviewMessageRequest{
		CampaignID:       campaignID,
		CustomerID:       req.CustomerID,
		OverrideTemplate: req.OverrideTemplate,
	}

	// Call service to generate preview
	result, err := h.campaignService.PreviewMessage(r.Context(), previewReq)
	if err != nil {
		// Handle service errors using response helper
		HandleServiceError(w, err)
		return
	}

	// Return success response with preview result
	WriteOK(w, result)
}
