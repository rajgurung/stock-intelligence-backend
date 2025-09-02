# Stock Data Fetcher Services

This directory contains dedicated services for populating the stock database with real market data from Alpha Vantage.

## 🎯 Overview

The data fetcher system consists of three components:

1. **Data Fetcher** (`cmd/data-fetcher/main.go`) - Core fetching logic
2. **Scheduler** (`cmd/scheduler/main.go`) - Background automation
3. **Fetch Script** (`scripts/fetch-data.sh`) - Manual execution

## 🚀 Quick Start

### Manual Data Fetch
```bash
# Run the convenient script
./scripts/fetch-data.sh

# Or run directly
go run cmd/data-fetcher/main.go
```

### Background Scheduling
```bash
# Start the background scheduler (runs every 2 hours)
go run cmd/scheduler/main.go
```

## 🧠 Smart Prioritization

The data fetcher intelligently prioritizes stocks:

1. **Missing Data First** - Stocks with no price data get highest priority
2. **Market Cap Order** - Within each priority level, larger companies first
3. **Alphabetical** - Final tiebreaker for consistent ordering

## 📊 Rate Limit Management

### Alpha Vantage Free Tier Limits
- **Daily Limit**: 25 API calls per day
- **Rate Limiting**: 5 calls per minute recommended
- **Reset Time**: Daily reset at midnight EST

### Built-in Protections
- ✅ **Rate Limit Tracking** - Automatically tracks daily usage
- ✅ **Smart Delays** - 12-second delays between calls (5/minute max)
- ✅ **Daily Resets** - Automatically resets counters each day  
- ✅ **Graceful Degradation** - Stops when limit reached

## 📈 Current Status

Based on your database:
- **Total Stocks**: 88 active stocks
- **Stocks with Data**: 19 stocks (22%)
- **Missing Data**: 69 stocks (78%)
- **Estimated Time**: 3-4 days to complete at 25 stocks/day

## 🔧 Configuration

### Environment Variables
```bash
DATABASE_URL=postgresql://rajg:your_password@localhost:5432/stock_intelligence?sslmode=disable
ALPHA_VANTAGE_API_KEY=your_api_key_here
```

### Database Tables Used
- `stocks` - Stock master data
- `daily_prices` - Historical price data
- `api_calls` - API call logging
- `api_rate_limits` - Rate limit tracking

## 📋 Execution Flow

1. **Rate Limit Check** - Verifies daily quota available
2. **Stock Prioritization** - Orders stocks by data gaps
3. **Data Fetching** - Retrieves daily prices from Alpha Vantage
4. **Database Storage** - Stores price data with conflict resolution
5. **Progress Logging** - Tracks success/failure counts

## 🐛 Troubleshooting

### Common Issues

**"Rate limit reached for today"**
- Solution: Wait for daily reset or upgrade Alpha Vantage plan
- Check: `SELECT * FROM api_rate_limits WHERE service_name = 'alphavantage'`

**"API information (likely rate limit)"**
- This means you've exceeded the 25 daily calls
- The fetcher will automatically stop and resume tomorrow

**"No time series data returned"**
- Usually indicates API rate limiting
- Could also mean invalid stock symbol

**Database connection errors**
- Verify `DATABASE_URL` in `.env` file
- Ensure PostgreSQL is running
- Check SSL mode setting (`sslmode=disable` for local)

### Monitoring

Check rate limit status:
```sql
SELECT * FROM api_rate_limits WHERE service_name = 'alphavantage';
```

Check recent API calls:
```sql
SELECT endpoint, response_status, created_at 
FROM api_calls 
ORDER BY created_at DESC 
LIMIT 10;
```

View stocks missing price data:
```sql
SELECT s.symbol, s.company_name 
FROM stocks s 
LEFT JOIN daily_prices dp ON s.id = dp.stock_id 
WHERE s.is_active = true 
  AND dp.close_price IS NULL
GROUP BY s.id, s.symbol, s.company_name
ORDER BY s.market_cap DESC;
```

## 🚀 Production Deployment

### Systemd Service Example
```ini
[Unit]
Description=Stock Data Fetcher Scheduler
After=network.target

[Service]
Type=simple
User=stock-app
WorkingDirectory=/path/to/backend
ExecStart=/usr/local/go/bin/go run cmd/scheduler/main.go
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
```

### Docker Container
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go mod download
CMD ["go", "run", "cmd/scheduler/main.go"]
```

### Cron Job Alternative
```bash
# Run every 2 hours
0 */2 * * * cd /path/to/backend && go run cmd/data-fetcher/main.go
```

## 📊 Upgrade Path

To increase data population speed:

### Alpha Vantage Premium Plans
- **Standard**: $25/month, 75 calls/day → Complete in 2 days
- **Intermediate**: $75/month, 1,200 calls/day → Complete in hours

### Alternative APIs
- Replace Alpha Vantage with higher-limit APIs
- IEX Cloud, Finnhub, Yahoo Finance (unofficial)
- Requires code modifications

## 🔍 Logs and Monitoring

The data fetcher provides detailed logging:

```
🚀 Starting Stock Data Fetcher Service...
✅ Connected to database successfully
📊 Rate Limit Status: 5/25 used, 20 remaining
🎯 Found 69 stocks needing price data
📥 Fetching data for AAPL (Apple Inc.) [1/69]
✅ Successfully fetched AAPL
📈 Stored 100 daily prices for stock ID 1
📊 Fetch Summary:
   ✅ Successful: 20 stocks
   ❌ Failed: 0 stocks
   📈 Total API calls made: 20
```

## 💡 Tips

1. **Run Daily** - Execute the fetcher once per day to maximize API usage
2. **Monitor Logs** - Check for rate limiting and API errors
3. **Check Progress** - Query the database to see completion status
4. **Plan Upgrades** - Consider premium API plans for faster population
5. **Backup Data** - Alpha Vantage data is valuable, back it up regularly