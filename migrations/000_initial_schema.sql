-- Migration: 000_initial_schema
-- Description: Create initial database schema for stock intelligence platform
-- This migration creates the main tables that already exist in the database

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create update_updated_at_column function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create stocks table
CREATE TABLE IF NOT EXISTS stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) UNIQUE NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    sector VARCHAR(100),
    industry VARCHAR(150),
    market_cap BIGINT,
    price_range VARCHAR(20),
    exchange VARCHAR(10) DEFAULT 'NASDAQ',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create trigger for stocks updated_at
DROP TRIGGER IF EXISTS update_stocks_updated_at ON stocks;
CREATE TRIGGER update_stocks_updated_at
    BEFORE UPDATE ON stocks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create daily_prices table
CREATE TABLE IF NOT EXISTS daily_prices (
    id SERIAL PRIMARY KEY,
    stock_id INTEGER NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    open_price NUMERIC(12,4) NOT NULL,
    high_price NUMERIC(12,4) NOT NULL,
    low_price NUMERIC(12,4) NOT NULL,
    close_price NUMERIC(12,4) NOT NULL,
    adjusted_close NUMERIC(12,4) NOT NULL,
    volume BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stock_id, date)
);

-- Create portfolios table
CREATE TABLE IF NOT EXISTS portfolios (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create portfolio_holdings table
CREATE TABLE IF NOT EXISTS portfolio_holdings (
    id SERIAL PRIMARY KEY,
    portfolio_id INTEGER NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    stock_id INTEGER NOT NULL REFERENCES stocks(id),
    shares NUMERIC(15,6) NOT NULL DEFAULT 0,
    average_cost NUMERIC(12,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(portfolio_id, stock_id)
);

-- Create watchlists table
CREATE TABLE IF NOT EXISTS watchlists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stock_id INTEGER NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, stock_id)
);

-- Create forecasts table
CREATE TABLE IF NOT EXISTS forecasts (
    id SERIAL PRIMARY KEY,
    stock_id INTEGER NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    forecast_date DATE NOT NULL,
    target_price NUMERIC(12,4) NOT NULL,
    confidence_interval_lower NUMERIC(12,4),
    confidence_interval_upper NUMERIC(12,4),
    model_name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stock_id, forecast_date, model_name)
);

-- Create news_articles table
CREATE TABLE IF NOT EXISTS news_articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    url VARCHAR(1000) UNIQUE,
    published_at TIMESTAMP,
    source VARCHAR(100),
    sentiment_score NUMERIC(3,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create market_indices table
CREATE TABLE IF NOT EXISTS market_indices (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    current_value NUMERIC(15,2),
    daily_change NUMERIC(15,2),
    daily_change_percent NUMERIC(5,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_stocks_symbol ON stocks(symbol);
CREATE INDEX IF NOT EXISTS idx_stocks_sector ON stocks(sector);
CREATE INDEX IF NOT EXISTS idx_stocks_sector_active ON stocks(sector, is_active);
CREATE INDEX IF NOT EXISTS idx_stocks_market_cap ON stocks(market_cap);

CREATE INDEX IF NOT EXISTS idx_daily_prices_stock_date ON daily_prices(stock_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_prices_date ON daily_prices(date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_prices_close_price ON daily_prices(close_price);
CREATE INDEX IF NOT EXISTS idx_daily_prices_volume ON daily_prices(volume DESC);
CREATE INDEX IF NOT EXISTS idx_daily_prices_date_only ON daily_prices(date);
CREATE INDEX IF NOT EXISTS idx_daily_prices_stock_date_simple ON daily_prices(stock_id, date);
CREATE INDEX IF NOT EXISTS idx_daily_prices_symbol_date_range ON daily_prices(stock_id, date DESC, close_price);

CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_portfolio_id ON portfolio_holdings(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_stock_id ON portfolio_holdings(stock_id);

CREATE INDEX IF NOT EXISTS idx_watchlists_user_id ON watchlists(user_id);
CREATE INDEX IF NOT EXISTS idx_watchlists_stock_id ON watchlists(stock_id);

CREATE INDEX IF NOT EXISTS idx_forecasts_stock_id ON forecasts(stock_id);
CREATE INDEX IF NOT EXISTS idx_forecasts_date ON forecasts(forecast_date);

CREATE INDEX IF NOT EXISTS idx_news_published_at ON news_articles(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_sentiment ON news_articles(sentiment_score);