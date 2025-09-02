-- Migration: 003_add_updated_at_to_daily_prices
-- Description: Add missing updated_at column to daily_prices table for proper timestamp tracking

-- Add updated_at column to daily_prices table
ALTER TABLE daily_prices 
ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create trigger to automatically update updated_at column
DROP TRIGGER IF EXISTS update_daily_prices_updated_at ON daily_prices;
CREATE TRIGGER update_daily_prices_updated_at
    BEFORE UPDATE ON daily_prices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Update existing rows to have current timestamp
UPDATE daily_prices 
SET updated_at = created_at 
WHERE updated_at IS NULL;

-- Create index on updated_at for performance
CREATE INDEX IF NOT EXISTS idx_daily_prices_updated_at ON daily_prices(updated_at DESC);