-- Create customers table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    location VARCHAR(100),
    preferred_product VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for phone lookups
CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers(phone);

-- Add comment for documentation
COMMENT ON TABLE customers IS 'Stores customer information for campaign targeting';