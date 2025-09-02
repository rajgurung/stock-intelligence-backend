-- Migration: 004_performance_indexes
-- Description: Add critical performance indexes identified by performance analysis

-- Index for latest daily prices lookup (most critical for performance)
CREATE INDEX IF NOT EXISTS idx_daily_prices_stock_latest_partial 
ON daily_prices(stock_id, date DESC) 
WHERE date >= '2024-08-01'::date;

-- Index for active stocks with market cap ordering
CREATE INDEX IF NOT EXISTS idx_stocks_active_market_cap 
ON stocks(is_active, market_cap DESC) 
WHERE is_active = true;

-- Composite index for price change calculations
CREATE INDEX IF NOT EXISTS idx_daily_prices_performance 
ON daily_prices(stock_id, date DESC, close_price, volume);

-- Index for sector-based filtering
CREATE INDEX IF NOT EXISTS idx_stocks_sector_active 
ON stocks(sector) 
WHERE is_active = true AND sector IS NOT NULL;

-- Index for symbol lookups (if not already exists)
CREATE INDEX IF NOT EXISTS idx_stocks_symbol_unique 
ON stocks(symbol) 
WHERE is_active = true;

-- Index for price range filtering
CREATE INDEX IF NOT EXISTS idx_stocks_price_range 
ON stocks(price_range) 
WHERE is_active = true AND price_range IS NOT NULL;

-- Covering index for stock list queries (includes all commonly selected columns)
CREATE INDEX IF NOT EXISTS idx_stocks_list_covering 
ON stocks(is_active, market_cap DESC) 
INCLUDE (id, symbol, company_name, sector, industry, price_range, exchange, created_at, updated_at)
WHERE is_active = true;

-- Index for historical performance queries with volume
CREATE INDEX IF NOT EXISTS idx_daily_prices_symbol_date_volume 
ON daily_prices(stock_id, date DESC, volume DESC);