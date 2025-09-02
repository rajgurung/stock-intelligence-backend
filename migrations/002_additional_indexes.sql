-- Additional indexes for aggregated data queries (simplified)
-- Migration: Additional indexes simple
-- Created: 2025-01-29

-- Composite index for symbol + date range queries (most common for historical data)
CREATE INDEX IF NOT EXISTS idx_daily_prices_symbol_date_range ON daily_prices(stock_id, date DESC, close_price);

-- Index for volume-based queries
CREATE INDEX IF NOT EXISTS idx_daily_prices_volume ON daily_prices(volume DESC);

-- Index for price change calculations
CREATE INDEX IF NOT EXISTS idx_daily_prices_close_price ON daily_prices(close_price);

-- Index for stocks by market cap ranges
CREATE INDEX IF NOT EXISTS idx_stocks_market_cap_ranges ON stocks(market_cap);

-- Composite index for sector analysis
CREATE INDEX IF NOT EXISTS idx_stocks_sector_active ON stocks(sector, is_active);

-- Index for date-based aggregations 
CREATE INDEX IF NOT EXISTS idx_daily_prices_date_only ON daily_prices(date);

-- Simple stock + date index for aggregations
CREATE INDEX IF NOT EXISTS idx_daily_prices_stock_date_simple ON daily_prices(stock_id, date);

COMMENT ON INDEX idx_daily_prices_symbol_date_range IS 'Optimizes symbol + date range queries for historical data';