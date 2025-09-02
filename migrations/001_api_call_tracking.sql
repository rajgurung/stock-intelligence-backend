-- API call tracking for external services
-- Migration: API call tracking
-- Created: 2025-01-29

-- Table to track API calls to external services (Alpha Vantage, etc.)
CREATE TABLE IF NOT EXISTS api_calls (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(50) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    request_params JSONB,
    response_status INTEGER NOT NULL,
    response_body TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processing_time_ms INTEGER DEFAULT 0
);

-- Indexes for API call tracking
CREATE INDEX idx_api_calls_service_created ON api_calls(service_name, created_at DESC);
CREATE INDEX idx_api_calls_status ON api_calls(response_status);
CREATE INDEX idx_api_calls_endpoint ON api_calls(endpoint);

-- API rate limiting table
CREATE TABLE IF NOT EXISTS api_rate_limits (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(50) NOT NULL,
    daily_limit INTEGER NOT NULL DEFAULT 25,
    hourly_limit INTEGER,
    current_daily_count INTEGER DEFAULT 0,
    current_hourly_count INTEGER DEFAULT 0,
    last_reset_date DATE DEFAULT CURRENT_DATE,
    last_reset_hour INTEGER DEFAULT EXTRACT(HOUR FROM CURRENT_TIMESTAMP),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(service_name)
);

-- Insert default rate limit for Alpha Vantage
INSERT INTO api_rate_limits (service_name, daily_limit, hourly_limit) 
VALUES ('alphavantage', 25, NULL)
ON CONFLICT (service_name) DO NOTHING;

-- Function to update rate limits
CREATE OR REPLACE FUNCTION update_api_rate_limit()
RETURNS TRIGGER AS $$
BEGIN
    -- Reset daily count if date changed
    IF NEW.last_reset_date < CURRENT_DATE THEN
        NEW.current_daily_count = 0;
        NEW.current_hourly_count = 0;
        NEW.last_reset_date = CURRENT_DATE;
        NEW.last_reset_hour = EXTRACT(HOUR FROM CURRENT_TIMESTAMP);
    -- Reset hourly count if hour changed
    ELSIF NEW.last_reset_hour < EXTRACT(HOUR FROM CURRENT_TIMESTAMP) THEN
        NEW.current_hourly_count = 0;
        NEW.last_reset_hour = EXTRACT(HOUR FROM CURRENT_TIMESTAMP);
    END IF;
    
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for rate limit updates
CREATE TRIGGER update_api_rate_limits_trigger 
    BEFORE UPDATE ON api_rate_limits
    FOR EACH ROW EXECUTE FUNCTION update_api_rate_limit();

-- View for API call statistics
CREATE OR REPLACE VIEW api_call_stats AS
SELECT 
    service_name,
    endpoint,
    COUNT(*) as total_calls,
    COUNT(*) FILTER (WHERE response_status = 200) as successful_calls,
    COUNT(*) FILTER (WHERE response_status >= 400) as failed_calls,
    ROUND(AVG(processing_time_ms), 2) as avg_processing_time_ms,
    MAX(created_at) as last_call_at,
    DATE(created_at) as call_date
FROM api_calls 
GROUP BY service_name, endpoint, DATE(created_at)
ORDER BY call_date DESC, service_name, endpoint;

COMMENT ON TABLE api_calls IS 'Log of all external API calls made by the application';
COMMENT ON TABLE api_rate_limits IS 'Rate limiting configuration and tracking for external APIs';
COMMENT ON VIEW api_call_stats IS 'Statistics for API calls grouped by service and endpoint';