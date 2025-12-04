package repository

import (
	"context"
	"database/sql"
	"fmt"

	"smsleopard/internal/models"

	"github.com/lib/pq"
)

type customerRepository struct {
	db *sql.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepository{db: db}
}

// Create creates a new customer
func (r *customerRepository) Create(ctx context.Context, customer *models.Customer) error {
	query := `
		INSERT INTO customers (phone, first_name, last_name, location, preferred_product)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
	).Scan(&customer.ID, &customer.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	return nil
}

// GetByID retrieves a customer by ID
func (r *customerRepository) GetByID(ctx context.Context, id int) (*models.Customer, error) {
	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		WHERE id = $1
	`

	customer := &models.Customer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&customer.ID,
		&customer.Phone,
		&customer.FirstName,
		&customer.LastName,
		&customer.Location,
		&customer.PreferredProduct,
		&customer.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("customer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return customer, nil
}

// GetByIDs retrieves multiple customers by IDs
func (r *customerRepository) GetByIDs(ctx context.Context, ids []int) ([]*models.Customer, error) {
	if len(ids) == 0 {
		return []*models.Customer{}, nil
	}

	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		WHERE id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}
	defer rows.Close()

	customers := []*models.Customer{}
	for rows.Next() {
		customer := &models.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.Phone,
			&customer.FirstName,
			&customer.LastName,
			&customer.Location,
			&customer.PreferredProduct,
			&customer.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// List retrieves customers with pagination
func (r *customerRepository) List(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	query := `
		SELECT id, phone, first_name, last_name, location, preferred_product, created_at
		FROM customers
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	customers := []*models.Customer{}
	for rows.Next() {
		customer := &models.Customer{}
		err := rows.Scan(
			&customer.ID,
			&customer.Phone,
			&customer.FirstName,
			&customer.LastName,
			&customer.Location,
			&customer.PreferredProduct,
			&customer.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// Update updates a customer
func (r *customerRepository) Update(ctx context.Context, customer *models.Customer) error {
	query := `
		UPDATE customers
		SET phone = $1, first_name = $2, last_name = $3, location = $4, preferred_product = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		customer.Phone,
		customer.FirstName,
		customer.LastName,
		customer.Location,
		customer.PreferredProduct,
		customer.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// Delete deletes a customer
func (r *customerRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM customers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}
