package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Stock represents a stock in the database
type Stock struct {
	ID          int    `json:"id"`
	Symbol      string `json:"symbol"`
	CompanyName string `json:"company_name"`
	Sector      string `json:"sector"`
}

// AlphaVantageResponse represents the Alpha Vantage API response
type AlphaVantageResponse struct {
	MetaData     map[string]string            `json:"Meta Data"`
	TimeSeries   map[string]map[string]string `json:"Time Series (Daily)"`
	ErrorMessage string                       `json:"Error Message"`
	Note         string                       `json:"Note"`
	Information  string                       `json:"Information"`
}

// DataFetcher handles fetching stock data
type DataFetcher struct {
	db     *sql.DB
	apiKey string
	client *http.Client
}

func main() {
	log.Println("🚀 Starting Stock Data Fetcher Service...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database successfully")

	// Initialize Alpha Vantage API key
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	if apiKey == "" {
		log.Fatal("ALPHA_VANTAGE_API_KEY environment variable is required")
	}

	// Create data fetcher
	fetcher := &DataFetcher{
		db:     db,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Run the data fetching process
	if err := fetcher.Run(); err != nil {
		log.Fatalf("Data fetching failed: %v", err)
	}

	log.Println("✅ Data fetching completed successfully")
}

// Run executes the main data fetching logic
func (df *DataFetcher) Run() error {
	log.Println("📊 Starting intelligent data fetching process...")

	// Step 1: Check current rate limit status
	canMakeRequests, remaining, err := df.checkRateLimit()
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %v", err)
	}

	if !canMakeRequests {
		log.Println("⏸️ Rate limit reached for today. No API calls will be made.")
		log.Println("💡 Alpha Vantage free tier allows 25 requests/day. Limit resets daily.")
		return nil
	}

	log.Printf("📈 Can make %d API calls today", remaining)

	// Step 2: Get stocks prioritized by missing data
	stocks, err := df.getPrioritizedStocks()
	if err != nil {
		return fmt.Errorf("failed to get prioritized stocks: %v", err)
	}

	if len(stocks) == 0 {
		log.Println("🎉 All stocks already have price data!")
		return nil
	}

	log.Printf("🎯 Found %d stocks needing price data", len(stocks))

	// Step 3: Fetch data for stocks within rate limit
	successCount := 0
	errorCount := 0

	for i, stock := range stocks {
		if i >= remaining {
			log.Printf("⏸️ Reached rate limit. Processed %d/%d stocks", i, len(stocks))
			break
		}

		log.Printf("📥 Fetching data for %s (%s) [%d/%d]", 
			stock.Symbol, stock.CompanyName, i+1, len(stocks))

		if err := df.fetchStockData(stock); err != nil {
			log.Printf("❌ Failed to fetch %s: %v", stock.Symbol, err)
			errorCount++
			
			// Add delay after errors to avoid hammering the API
			time.Sleep(2 * time.Second)
		} else {
			log.Printf("✅ Successfully fetched %s", stock.Symbol)
			successCount++
		}

		// Update rate limit after each call
		df.updateRateLimit()

		// Respectful delay between API calls (Alpha Vantage recommends this)
		time.Sleep(12 * time.Second) // 5 calls per minute max
	}

	// Step 4: Log summary
	log.Printf("📊 Fetch Summary:")
	log.Printf("   ✅ Successful: %d stocks", successCount)
	log.Printf("   ❌ Failed: %d stocks", errorCount)
	log.Printf("   📈 Total API calls made: %d", successCount+errorCount)

	return nil
}

// checkRateLimit checks if we can make API calls today
func (df *DataFetcher) checkRateLimit() (bool, int, error) {
	query := `
		SELECT daily_limit, current_daily_count, last_reset_date 
		FROM api_rate_limits 
		WHERE service_name = 'alphavantage' 
		LIMIT 1
	`

	var dailyLimit, currentCount int
	var lastResetDate string
	
	err := df.db.QueryRow(query).Scan(&dailyLimit, &currentCount, &lastResetDate)
	if err == sql.ErrNoRows {
		// Initialize rate limit tracking
		return df.initializeRateLimit()
	}
	if err != nil {
		return false, 0, err
	}

	// Check if we need to reset daily count
	today := time.Now().Format("2006-01-02")
	if lastResetDate != today {
		// Reset daily count
		currentCount = 0
		_, err := df.db.Exec(`
			UPDATE api_rate_limits 
			SET current_daily_count = 0, last_reset_date = $1, updated_at = CURRENT_TIMESTAMP 
			WHERE service_name = 'alphavantage'
		`, today)
		if err != nil {
			return false, 0, err
		}
		log.Println("🔄 Daily rate limit reset")
	}

	remaining := dailyLimit - currentCount
	canMake := remaining > 0

	log.Printf("📊 Rate Limit Status: %d/%d used, %d remaining", 
		currentCount, dailyLimit, remaining)

	return canMake, remaining, nil
}

// initializeRateLimit sets up rate limit tracking for Alpha Vantage
func (df *DataFetcher) initializeRateLimit() (bool, int, error) {
	dailyLimit := 25 // Alpha Vantage free tier limit
	today := time.Now().Format("2006-01-02")

	_, err := df.db.Exec(`
		INSERT INTO api_rate_limits 
		(service_name, daily_limit, current_daily_count, last_reset_date, created_at, updated_at)
		VALUES ('alphavantage', $1, 0, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (service_name) DO UPDATE SET
			daily_limit = EXCLUDED.daily_limit,
			last_reset_date = EXCLUDED.last_reset_date,
			updated_at = CURRENT_TIMESTAMP
	`, dailyLimit, today)
	
	if err != nil {
		return false, 0, err
	}

	log.Printf("✅ Initialized rate limit tracking: %d calls/day", dailyLimit)
	return true, dailyLimit, nil
}

// getPrioritizedStocks returns stocks prioritized by missing data
func (df *DataFetcher) getPrioritizedStocks() ([]Stock, error) {
	query := `
		SELECT s.id, s.symbol, s.company_name, s.sector
		FROM stocks s
		LEFT JOIN daily_prices dp ON s.id = dp.stock_id
		WHERE s.is_active = true
		GROUP BY s.id, s.symbol, s.company_name, s.sector
		ORDER BY 
			CASE WHEN COUNT(dp.id) = 0 THEN 1 ELSE 2 END,  -- Prioritize stocks with no price data
			s.market_cap DESC NULLS LAST,                   -- Then by market cap
			s.symbol                                        -- Finally alphabetically
	`

	rows, err := df.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []Stock
	for rows.Next() {
		var stock Stock
		if err := rows.Scan(&stock.ID, &stock.Symbol, &stock.CompanyName, &stock.Sector); err != nil {
			log.Printf("Warning: Failed to scan stock: %v", err)
			continue
		}
		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// fetchStockData fetches and stores daily price data for a stock
func (df *DataFetcher) fetchStockData(stock Stock) error {
	// Build API URL
	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s",
		stock.Symbol, df.apiKey,
	)

	// Make API request
	resp, err := df.client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	// Log API call
	df.logAPICall("alphavantage", "TIME_SERIES_DAILY", stock.Symbol, resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var data AlphaVantageResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("JSON parsing failed: %v", err)
	}

	// Check for API errors
	if data.ErrorMessage != "" {
		return fmt.Errorf("API error: %s", data.ErrorMessage)
	}
	if data.Note != "" {
		return fmt.Errorf("API rate limit note: %s", data.Note)
	}
	if data.Information != "" {
		return fmt.Errorf("API information (likely rate limit): %s", data.Information)
	}

	// Extract and store daily prices
	if len(data.TimeSeries) == 0 {
		return fmt.Errorf("no time series data returned")
	}

	return df.storeDailyPrices(stock.ID, data.TimeSeries)
}

// storeDailyPrices stores daily price data in the database
func (df *DataFetcher) storeDailyPrices(stockID int, timeSeries map[string]map[string]string) error {
	tx, err := df.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertCount := 0
	for dateStr, prices := range timeSeries {
		// Parse date
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Warning: Invalid date format %s", dateStr)
			continue
		}

		// Parse prices
		open, _ := strconv.ParseFloat(prices["1. open"], 64)
		high, _ := strconv.ParseFloat(prices["2. high"], 64)
		low, _ := strconv.ParseFloat(prices["3. low"], 64)
		closePrice, _ := strconv.ParseFloat(prices["4. close"], 64)
		volume, _ := strconv.ParseInt(prices["5. volume"], 10, 64)

		// Insert or update daily price
		_, err = tx.Exec(`
			INSERT INTO daily_prices (stock_id, date, open_price, high_price, low_price, close_price, volume, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT (stock_id, date) DO UPDATE SET
				open_price = EXCLUDED.open_price,
				high_price = EXCLUDED.high_price,
				low_price = EXCLUDED.low_price,
				close_price = EXCLUDED.close_price,
				volume = EXCLUDED.volume,
				updated_at = CURRENT_TIMESTAMP
		`, stockID, date, open, high, low, closePrice, volume)

		if err != nil {
			return fmt.Errorf("failed to insert daily price: %v", err)
		}
		insertCount++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("📈 Stored %d daily prices for stock ID %d", insertCount, stockID)
	return nil
}

// logAPICall logs API call details
func (df *DataFetcher) logAPICall(service, endpoint, symbol string, status int, response string) {
	requestParams := fmt.Sprintf(`{"symbol": "%s"}`, symbol)
	
	_, err := df.db.Exec(`
		INSERT INTO api_calls 
		(service_name, endpoint, request_params, response_status, response_body, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
	`, service, endpoint, requestParams, status, response)
	
	if err != nil {
		log.Printf("Warning: Failed to log API call: %v", err)
	}
}

// updateRateLimit increments the daily API call count
func (df *DataFetcher) updateRateLimit() {
	_, err := df.db.Exec(`
		UPDATE api_rate_limits 
		SET current_daily_count = current_daily_count + 1, updated_at = CURRENT_TIMESTAMP 
		WHERE service_name = 'alphavantage'
	`)
	if err != nil {
		log.Printf("Warning: Failed to update rate limit: %v", err)
	}
}