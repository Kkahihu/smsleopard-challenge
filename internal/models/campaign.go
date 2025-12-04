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
	ID           int            `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	Channel      Channel        `json:"channel" db:"channel"`
	Status       CampaignStatus `json:"status" db:"status"`
	BaseTemplate string         `json:"base_template" db:"base_template"`
	ScheduledAt  *time.Time     `json:"scheduled_at,omitempty" db:"scheduled_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
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
