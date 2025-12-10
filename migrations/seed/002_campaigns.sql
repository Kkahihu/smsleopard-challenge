-- ============================================================================
-- Seed File: Campaign Test Data
-- ============================================================================
-- Purpose: Populate campaigns table with diverse test campaigns for
-- development and testing purposes.
--
-- Campaign Characteristics:
-- - 3 campaigns with different statuses and channels
-- - Templates with various placeholder combinations
-- - Unicode emoji support for edge case testing
-- - Different scheduling scenarios
-- - Idempotent: Safe to run multiple times
-- ============================================================================

-- Insert three diverse campaign records
INSERT INTO campaigns (name, channel, status, base_template, scheduled_at) VALUES
    -- Campaign 1: Welcome Campaign (SMS, Draft)
    -- Purpose: Simple welcome message for new customers
    -- Status: Draft (not yet scheduled)
    -- Placeholders: {first_name} only
    (
        'Welcome Campaign',
        'sms',
        'draft',
        'Hi {first_name}! 🎉 Welcome to SMSLeopard. We''re excited to have you with us!',
        NULL
    ),
    
    -- Campaign 2: Product Launch Announcement (WhatsApp, Sent)
    -- Purpose: Rich product launch with multiple personalization fields
    -- Status: Sent (already executed)
    -- Placeholders: {first_name}, {last_name}, {location}, {preferred_product}
    (
        'Product Launch Announcement',
        'whatsapp',
        'sent',
        'Hello {first_name} {last_name}! 🎁 Great news from {location}! We just launched new {preferred_product} with amazing features. Check them out today and get 20% off your first purchase!',
        NOW() - INTERVAL '2 days'
    ),
    
    -- Campaign 3: Special Offer (SMS, Scheduled)
    -- Purpose: Time-sensitive promotional offer
    -- Status: Scheduled (will be sent tomorrow)
    -- Placeholders: {first_name}, {preferred_product}
    (
        'Special Offer - Flash Sale',
        'sms',
        'scheduled',
        'Hey {first_name}! 🎉 Flash Sale Alert! Get 30% off on {preferred_product} for the next 24 hours only. Don''t miss out!',
        NOW() + INTERVAL '1 day'
    )

-- Handle conflicts: If campaign with same name exists, do nothing (idempotency)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Verification Query: Display inserted campaigns with key information
-- ============================================================================
-- This query helps verify the seed operation was successful

-- Display all seeded campaigns with key details
SELECT 
    id,
    name,
    channel,
    status,
    LENGTH(base_template) as template_length,
    CASE 
        WHEN base_template LIKE '%{first_name}%' THEN 'Yes'
        ELSE 'No'
    END as has_first_name_placeholder,
    CASE 
        WHEN base_template LIKE '%{last_name}%' THEN 'Yes'
        ELSE 'No'
    END as has_last_name_placeholder,
    CASE 
        WHEN base_template LIKE '%{location}%' THEN 'Yes'
        ELSE 'No'
    END as has_location_placeholder,
    CASE 
        WHEN base_template LIKE '%{preferred_product}%' THEN 'Yes'
        ELSE 'No'
    END as has_product_placeholder,
    CASE 
        WHEN base_template LIKE '%🎉%' OR base_template LIKE '%🎁%' THEN 'Yes'
        ELSE 'No'
    END as has_emoji,
    scheduled_at,
    created_at
FROM campaigns
WHERE name IN (
    'Welcome Campaign',
    'Product Launch Announcement',
    'Special Offer - Flash Sale'
)
ORDER BY created_at;

-- Summary statistics
SELECT 
    COUNT(*) as total_campaigns,
    COUNT(CASE WHEN channel = 'sms' THEN 1 END) as sms_campaigns,
    COUNT(CASE WHEN channel = 'whatsapp' THEN 1 END) as whatsapp_campaigns,
    COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft_campaigns,
    COUNT(CASE WHEN status = 'scheduled' THEN 1 END) as scheduled_campaigns,
    COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent_campaigns
FROM campaigns
WHERE name IN (
    'Welcome Campaign',
    'Product Launch Announcement',
    'Special Offer - Flash Sale'
);