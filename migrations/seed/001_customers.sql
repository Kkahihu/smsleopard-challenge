-- ============================================================================
-- Seed File: Customer Test Data
-- ============================================================================
-- Purpose: Populate customers table with diverse test data for development
-- and testing purposes.
--
-- Data Characteristics:
-- - 15 diverse customers (exceeds minimum of 10)
-- - Phone numbers follow pattern: +2547000000XX for easy identification
-- - Includes NULL field variations for testing edge cases
-- - Uses realistic Kenyan names and locations
-- - Diverse product preferences
-- - Idempotent: Safe to run multiple times
-- ============================================================================

-- Insert diverse customer records with various NULL field combinations
INSERT INTO customers (phone, first_name, last_name, location, preferred_product) VALUES
    -- Complete records (all fields populated)
    ('+254700000001', 'James', 'Kamau', 'Nairobi', 'Smartphones'),
    ('+254700000002', 'Mary', 'Wanjiru', 'Mombasa', 'Laptops'),
    ('+254700000003', 'Peter', 'Ochieng', 'Kisumu', 'Tablets'),
    ('+254700000004', 'Grace', 'Akinyi', 'Eldoret', 'Cameras'),
    ('+254700000005', 'David', 'Kipchoge', 'Nakuru', 'Headphones'),
    
    -- Records with NULL last_name (testing single NULL field)
    ('+254700000006', 'Sarah', NULL, 'Thika', 'Watches'),
    ('+254700000007', 'John', NULL, 'Nyeri', 'Speakers'),
    
    -- Records with NULL location (testing different NULL field)
    ('+254700000008', 'Agnes', 'Muthoni', NULL, 'Smartwatches'),
    ('+254700000009', 'Michael', 'Otieno', NULL, 'Gaming Consoles'),
    
    -- Records with NULL preferred_product (testing another NULL field)
    ('+254700000010', 'Faith', 'Njeri', 'Kitale', NULL),
    ('+254700000011', 'Daniel', 'Kibet', 'Machakos', NULL),
    
    -- Records with multiple NULL fields (edge case testing)
    ('+254700000012', 'Alice', NULL, NULL, 'Laptops'),
    ('+254700000013', 'Joseph', NULL, 'Kakamega', NULL),
    ('+254700000014', 'Lucy', 'Wambui', NULL, NULL),
    
    -- Record with only phone and first_name (maximum NULLs)
    ('+254700000015', 'Samuel', NULL, NULL, NULL)

-- Handle conflicts: If phone already exists, do nothing (idempotency)
ON CONFLICT (phone) DO NOTHING;

-- ============================================================================
-- Verification Query: Display inserted customer count and sample records
-- ============================================================================
-- This query helps verify the seed operation was successful

-- Count total customers with seeded phone pattern
SELECT 
    COUNT(*) as total_seeded_customers,
    COUNT(last_name) as customers_with_lastname,
    COUNT(location) as customers_with_location,
    COUNT(preferred_product) as customers_with_product,
    COUNT(*) - COUNT(last_name) as null_lastname_count,
    COUNT(*) - COUNT(location) as null_location_count,
    COUNT(*) - COUNT(preferred_product) as null_product_count
FROM customers
WHERE phone LIKE '+25470000001%';

-- Display sample of seeded customers
SELECT 
    id,
    phone,
    first_name,
    last_name,
    location,
    preferred_product,
    created_at
FROM customers
WHERE phone LIKE '+25470000001%'
ORDER BY phone
LIMIT 15;