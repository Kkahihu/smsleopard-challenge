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
