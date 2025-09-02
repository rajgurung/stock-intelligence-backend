-- Migration 003: Stock Priority and Data Completeness Tracking
-- Purpose: Add fields to track S&P 500 priority and historical data completeness

-- Add priority and data tracking columns to stocks table
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 999;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS has_sufficient_data BOOLEAN DEFAULT false;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS last_data_sync TIMESTAMP;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS data_quality_score INTEGER DEFAULT 0;

-- Create index for efficient priority-based queries
CREATE INDEX IF NOT EXISTS idx_stocks_priority ON stocks(priority, has_sufficient_data);
CREATE INDEX IF NOT EXISTS idx_stocks_data_sync ON stocks(last_data_sync);

-- Create view for data-complete stocks only
CREATE OR REPLACE VIEW stocks_with_data AS
SELECT s.*, 
       dp_count.daily_price_count,
       dp_latest.latest_price_date
FROM stocks s
LEFT JOIN (
    SELECT stock_id, COUNT(*) as daily_price_count
    FROM daily_prices 
    GROUP BY stock_id
) dp_count ON s.id = dp_count.stock_id
LEFT JOIN (
    SELECT stock_id, MAX(date) as latest_price_date
    FROM daily_prices 
    GROUP BY stock_id
) dp_latest ON s.id = dp_latest.stock_id
WHERE s.is_active = true 
  AND s.has_sufficient_data = true
  AND dp_count.daily_price_count >= 30;

-- Create priority stocks view for S&P 500
CREATE OR REPLACE VIEW priority_stocks AS
SELECT s.*,
       COALESCE(dp_count.daily_price_count, 0) as daily_price_count,
       dp_latest.latest_price_date
FROM stocks s
LEFT JOIN (
    SELECT stock_id, COUNT(*) as daily_price_count
    FROM daily_prices 
    GROUP BY stock_id
) dp_count ON s.id = dp_count.stock_id
LEFT JOIN (
    SELECT stock_id, MAX(date) as latest_price_date
    FROM daily_prices 
    GROUP BY stock_id
) dp_latest ON s.id = dp_latest.stock_id
WHERE s.is_active = true 
  AND s.priority < 999  -- Only S&P 500 stocks
ORDER BY s.priority ASC;

-- Function to update data completeness status
CREATE OR REPLACE FUNCTION update_stock_data_status(stock_symbol TEXT) 
RETURNS void AS $$
DECLARE
    stock_record RECORD;
    price_count INTEGER;
    latest_date DATE;
BEGIN
    -- Get stock info
    SELECT id, symbol INTO stock_record 
    FROM stocks 
    WHERE symbol = stock_symbol AND is_active = true;
    
    IF NOT FOUND THEN
        RAISE NOTICE 'Stock % not found', stock_symbol;
        RETURN;
    END IF;
    
    -- Count daily prices and get latest date
    SELECT COUNT(*), MAX(date) INTO price_count, latest_date
    FROM daily_prices 
    WHERE stock_id = stock_record.id;
    
    -- Update data status
    UPDATE stocks 
    SET has_sufficient_data = (price_count >= 30),
        data_quality_score = LEAST(100, price_count),
        last_data_sync = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = stock_record.id;
    
    RAISE NOTICE 'Updated % - Has data: %, Price count: %, Latest: %', 
        stock_symbol, (price_count >= 30), price_count, latest_date;
END;
$$ LANGUAGE plpgsql;

-- Function to batch update all stock data statuses
CREATE OR REPLACE FUNCTION update_all_stock_data_status() 
RETURNS TABLE(symbol TEXT, has_data BOOLEAN, price_count BIGINT) AS $$
BEGIN
    RETURN QUERY
    WITH stock_data_summary AS (
        SELECT s.symbol,
               COUNT(dp.date) >= 30 as sufficient_data,
               COUNT(dp.date) as price_count
        FROM stocks s
        LEFT JOIN daily_prices dp ON s.id = dp.stock_id
        WHERE s.is_active = true
        GROUP BY s.id, s.symbol
    )
    UPDATE stocks 
    SET has_sufficient_data = sds.sufficient_data,
        data_quality_score = LEAST(100, sds.price_count::INTEGER),
        last_data_sync = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    FROM stock_data_summary sds
    WHERE stocks.symbol = sds.symbol
    RETURNING stocks.symbol, stocks.has_sufficient_data, sds.price_count;
END;
$$ LANGUAGE plpgsql;