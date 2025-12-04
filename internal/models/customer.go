package models

import "time"

// Customer represents a customer in the system
type Customer struct {
	ID               int       `json:"id" db:"id"`
	Phone            string    `json:"phone" db:"phone"`
	FirstName        *string   `json:"first_name,omitempty" db:"first_name"`
	LastName         *string   `json:"last_name,omitempty" db:"last_name"`
	Location         *string   `json:"location,omitempty" db:"location"`
	PreferredProduct *string   `json:"preferred_product,omitempty" db:"preferred_product"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// FullName returns the customer's full name
func (c *Customer) FullName() string {
	var firstName, lastName string

	if c.FirstName != nil {
		firstName = *c.FirstName
	}
	if c.LastName != nil {
		lastName = *c.LastName
	}

	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	return "Customer"
}
