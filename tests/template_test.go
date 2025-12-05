package tests

import (
	"testing"

	"smsleopard/internal/models"
	"smsleopard/internal/service"
)

// TestTemplateRendering_AllFields tests placeholder substitution with all customer fields populated
// This verifies that all supported placeholders are correctly replaced
func TestTemplateRendering_AllFields(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()

	template := "Hi {first_name} {last_name} from {location}! Check out our {preferred_product}. Contact: {phone}"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify
	AssertNoError(t, err)
	expected := "Hi John Doe from Nairobi! Check out our Premium Plan. Contact: +254700000001"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_NullFields tests template rendering with NULL customer fields
// Strategy: NULL fields are replaced with empty strings
func TestTemplateRendering_NullFields(t *testing.T) {
	// Setup - customer with NULL optional fields
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomerNullFields()

	template := "Hi {first_name} {last_name} from {location}!"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - NULL fields are replaced with empty strings
	AssertNoError(t, err)
	expected := "Hi   from !"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_PartialNullFields tests template with some NULL and some populated fields
func TestTemplateRendering_PartialNullFields(t *testing.T) {
	// Setup - customer with mixed NULL and populated fields
	templateSvc := service.NewTemplateService()
	location := "Mombasa"
	customer := &models.Customer{
		ID:               1,
		Phone:            "+254700000001",
		FirstName:        nil, // NULL
		LastName:         nil, // NULL
		Location:         &location,
		PreferredProduct: nil, // NULL
	}

	template := "Hello {first_name}! We serve {location}. Try {preferred_product}."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - NULL fields become empty, populated fields are rendered
	AssertNoError(t, err)
	expected := "Hello ! We serve Mombasa. Try ."
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_MultipleCombinations tests various template and data combinations
func TestTemplateRendering_MultipleCombinations(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		customer *models.Customer
		expected string
	}{
		{
			name:     "first name only",
			template: "Hello {first_name}!",
			customer: &models.Customer{
				ID:        1,
				Phone:     "+254700000001",
				FirstName: StringPtr("Alice"),
			},
			expected: "Hello Alice!",
		},
		{
			name:     "location and product",
			template: "{location} residents love our {preferred_product}",
			customer: &models.Customer{
				ID:               2,
				Phone:            "+254700000002",
				Location:         StringPtr("Mombasa"),
				PreferredProduct: StringPtr("Basic Package"),
			},
			expected: "Mombasa residents love our Basic Package",
		},
		{
			name:     "phone only",
			template: "Contact us at {phone}",
			customer: &models.Customer{
				ID:    3,
				Phone: "+254700000003",
			},
			expected: "Contact us at +254700000003",
		},
		{
			name:     "full name format",
			template: "Dear {first_name} {last_name},",
			customer: &models.Customer{
				ID:        4,
				Phone:     "+254700000004",
				FirstName: StringPtr("Bob"),
				LastName:  StringPtr("Smith"),
			},
			expected: "Dear Bob Smith,",
		},
		{
			name:     "complex template",
			template: "{first_name} from {location}: Your {preferred_product} is ready. Call {phone}.",
			customer: &models.Customer{
				ID:               5,
				Phone:            "+254700000005",
				FirstName:        StringPtr("Carol"),
				Location:         StringPtr("Kisumu"),
				PreferredProduct: StringPtr("Enterprise Suite"),
			},
			expected: "Carol from Kisumu: Your Enterprise Suite is ready. Call +254700000005.",
		},
	}

	templateSvc := service.NewTemplateService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := templateSvc.Render(tc.template, tc.customer)
			AssertNoError(t, err)
			AssertEqual(t, result, tc.expected)
		})
	}
}

// TestTemplateRendering_EmptyTemplate tests rendering with an empty template
// Expected: Should return error
func TestTemplateRendering_EmptyTemplate(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := ""

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - should return error for empty template
	AssertError(t, err, "template cannot be empty")
	AssertEqual(t, result, "")
}

// TestTemplateRendering_NoPlaceholders tests template without any placeholders
// Expected: Template should be returned unchanged
func TestTemplateRendering_NoPlaceholders(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "This is a plain text message with no placeholders."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - should return template as-is
	AssertNoError(t, err)
	AssertEqual(t, result, template)
}

// TestTemplateRendering_UnknownPlaceholders tests template with unknown placeholders
// Expected: Unknown placeholders should be left as-is in the output
func TestTemplateRendering_UnknownPlaceholders(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Hello {first_name}, your {unknown_field} is ready."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - known placeholders replaced, unknown left as-is
	AssertNoError(t, err)
	expected := "Hello John, your {unknown_field} is ready."
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_MultipleUnknownPlaceholders tests multiple unknown placeholders
func TestTemplateRendering_MultipleUnknownPlaceholders(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Hi {first_name}, {unknown1} and {unknown2} at {phone}."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - all unknown placeholders remain
	AssertNoError(t, err)
	expected := "Hi John, {unknown1} and {unknown2} at +254700000001."
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_RepeatedPlaceholders tests template with repeated placeholders
// Expected: All occurrences should be replaced
func TestTemplateRendering_RepeatedPlaceholders(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Hi {first_name}, yes you {first_name}! Contact: {phone} or {phone}."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - both occurrences of each placeholder are replaced
	AssertNoError(t, err)
	expected := "Hi John, yes you John! Contact: +254700000001 or +254700000001."
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_SpecialCharacters tests template with special characters
// Expected: Special characters should not interfere with placeholder replacement
func TestTemplateRendering_SpecialCharacters(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Price: $99 for {preferred_product}! Call @{phone} #discount"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - special chars don't interfere with placeholders
	AssertNoError(t, err)
	expected := "Price: $99 for Premium Plan! Call @+254700000001 #discount"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_UnicodeCharacters tests template with unicode/emoji characters
func TestTemplateRendering_UnicodeCharacters(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "🎉 Hello {first_name}! ✨ Welcome to {location} 🌍"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - unicode characters work correctly
	AssertNoError(t, err)
	expected := "🎉 Hello John! ✨ Welcome to Nairobi 🌍"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_LongTemplate tests rendering with a long, complex template
func TestTemplateRendering_LongTemplate(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Dear {first_name} {last_name}, " +
		"Thank you for being a valued customer from {location}. " +
		"We're excited to announce that your {preferred_product} has been upgraded! " +
		"For any questions, please call us at {phone}. " +
		"Best regards, {first_name}!"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify
	AssertNoError(t, err)
	expected := "Dear John Doe, " +
		"Thank you for being a valued customer from Nairobi. " +
		"We're excited to announce that your Premium Plan has been upgraded! " +
		"For any questions, please call us at +254700000001. " +
		"Best regards, John!"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_AllNullFieldsCombinations tests all possible NULL field scenarios
func TestTemplateRendering_AllNullFieldsCombinations(t *testing.T) {
	testCases := []struct {
		name     string
		customer *models.Customer
		template string
		expected string
	}{
		{
			name: "all fields NULL",
			customer: &models.Customer{
				ID:               1,
				Phone:            "+254700000001",
				FirstName:        nil,
				LastName:         nil,
				Location:         nil,
				PreferredProduct: nil,
			},
			template: "{first_name} {last_name} {location} {preferred_product}",
			expected: "   ",
		},
		{
			name: "only first_name NULL",
			customer: &models.Customer{
				ID:               2,
				Phone:            "+254700000002",
				FirstName:        nil,
				LastName:         StringPtr("Doe"),
				Location:         StringPtr("Nairobi"),
				PreferredProduct: StringPtr("Plan A"),
			},
			template: "{first_name} {last_name} from {location}",
			expected: " Doe from Nairobi",
		},
		{
			name: "only preferred_product NULL",
			customer: &models.Customer{
				ID:               3,
				Phone:            "+254700000003",
				FirstName:        StringPtr("John"),
				LastName:         StringPtr("Doe"),
				Location:         StringPtr("Nairobi"),
				PreferredProduct: nil,
			},
			template: "Hi {first_name}! Try {preferred_product}.",
			expected: "Hi John! Try .",
		},
	}

	templateSvc := service.NewTemplateService()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := templateSvc.Render(tc.template, tc.customer)
			AssertNoError(t, err)
			AssertEqual(t, result, tc.expected)
		})
	}
}

// TestTemplateRendering_NilCustomer tests rendering with nil customer
// Expected: Should return error
func TestTemplateRendering_NilCustomer(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	template := "Hello {first_name}"

	// Execute
	result, err := templateSvc.Render(template, nil)

	// Verify - should return error for nil customer
	AssertError(t, err, "customer cannot be nil")
	AssertEqual(t, result, "")
}

// TestTemplateRendering_MixedPlaceholdersAndText tests complex mixing of text and placeholders
func TestTemplateRendering_MixedPlaceholdersAndText(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()
	template := "Hello{first_name}from{location}!Call{phone}now."

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - placeholders work even without spaces
	AssertNoError(t, err)
	expected := "HelloJohnfromNairobi!Call+254700000001now."
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_EmptyStringFields tests customer with empty string (not NULL) fields
func TestTemplateRendering_EmptyStringFields(t *testing.T) {
	// Setup
	templateSvc := service.NewTemplateService()
	emptyString := ""
	customer := &models.Customer{
		ID:               1,
		Phone:            "+254700000001",
		FirstName:        &emptyString, // Empty string, not NULL
		LastName:         StringPtr("Doe"),
		Location:         &emptyString, // Empty string, not NULL
		PreferredProduct: StringPtr("Plan"),
	}
	template := "Hi {first_name} {last_name} from {location}! Product: {preferred_product}"

	// Execute
	result, err := templateSvc.Render(template, customer)

	// Verify - empty strings are replaced with empty (same as NULL)
	AssertNoError(t, err)
	expected := "Hi  Doe from ! Product: Plan"
	AssertEqual(t, result, expected)
}

// TestTemplateRendering_WhitespaceHandling tests templates with various whitespace
func TestTemplateRendering_WhitespaceHandling(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "multiple spaces between placeholders",
			template: "Hello  {first_name}   {last_name}",
			expected: "Hello  John   Doe",
		},
		{
			name:     "newlines in template",
			template: "Hello {first_name}\nFrom {location}",
			expected: "Hello John\nFrom Nairobi",
		},
		{
			name:     "tabs in template",
			template: "Name:\t{first_name}\t{last_name}",
			expected: "Name:\tJohn\tDoe",
		},
	}

	templateSvc := service.NewTemplateService()
	customer := NewTestCustomer()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := templateSvc.Render(tc.template, customer)
			AssertNoError(t, err)
			AssertEqual(t, result, tc.expected)
		})
	}
}
