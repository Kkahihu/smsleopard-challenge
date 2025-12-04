package service

import "fmt"

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
	ID       int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %d not found", e.Resource, e.ID)
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// BusinessLogicError represents a business logic error
type BusinessLogicError struct {
	Message string
}

func (e *BusinessLogicError) Error() string {
	return fmt.Sprintf("business logic error: %s", e.Message)
}

// ConflictError represents a conflict error (e.g., duplicate)
type ConflictError struct {
	Resource string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict with %s: %s", e.Resource, e.Message)
}
