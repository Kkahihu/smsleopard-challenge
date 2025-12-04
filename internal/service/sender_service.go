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
