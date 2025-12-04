-- Create outbound_messages table
CREATE TABLE IF NOT EXISTS outbound_messages (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'sent', 'failed')),
    rendered_content TEXT,
    last_error TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_outbound_messages_campaign_id ON outbound_messages(campaign_id);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_status ON outbound_messages(status);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_created_at ON outbound_messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_customer_id ON outbound_messages(customer_id);

-- Add comments for documentation
COMMENT ON TABLE outbound_messages IS 'Tracks individual message deliveries for campaigns';
COMMENT ON COLUMN outbound_messages.retry_count IS 'Number of delivery attempts (max 3)';
COMMENT ON COLUMN outbound_messages.rendered_content IS 'Final personalized message content';