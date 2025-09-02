package tasks

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"stock-intelligence-backend/internal/services"
)

type TaskRunner struct {
	db                 *sql.DB
	alphaVantageClient *services.AlphaVantageClient
}

func NewTaskRunner(db *sql.DB, alphaVantageClient *services.AlphaVantageClient) *TaskRunner {
	return &TaskRunner{
		db:                 db,
		alphaVantageClient: alphaVantageClient,
	}
}

// SeedDatabase seeds the database with initial stock symbols and sample historical data
func (t *TaskRunner) SeedDatabase() error {
	log.Println("Starting database seeding...")
	
	// First seed the stocks
	if err := t.SeedStocks(); err != nil {
		return fmt.Errorf("failed to seed stocks: %w", err)
	}
	
	// Then fetch some sample historical data (limited by rate limits)
	log.Println("Fetching sample historical data for top 5 stocks...")
	topStocks := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA"}
	
	for i, symbol := range topStocks {
		// Respect rate limits - only fetch if we can make requests
		canMake, err := t.alphaVantageClient.CanMakeRequest()
		if err != nil {
			log.Printf("Failed to check rate limit: %v", err)
			break
		}
		if !canMake {
			log.Printf("Rate limit reached after %d stocks. Run 'data:fetch:all' later to get remaining data.", i)
			break
		}
		
		log.Printf("Fetching historical data for %s (%d/%d)...", symbol, i+1, len(topStocks))
		if err := t.fetchHistoricalDataForSymbol(symbol); err != nil {
			log.Printf("Warning: Failed to fetch data for %s: %v", symbol, err)
			continue
		}
		
		// Small delay between requests
		time.Sleep(2 * time.Second)
	}
	
	return nil
}

// SeedStocks seeds the database with stock symbols (S&P 500 subset)
func (t *TaskRunner) SeedStocks() error {
	log.Println("Seeding stock symbols...")
	
	stocks := getStockSeeds()
	
	// Prepare insert statement
	insertQuery := `
		INSERT INTO stocks (symbol, company_name, sector, industry, exchange, market_cap, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (symbol) 
		DO UPDATE SET 
			company_name = EXCLUDED.company_name,
			sector = EXCLUDED.sector,
			industry = EXCLUDED.industry,
			market_cap = EXCLUDED.market_cap,
			updated_at = CURRENT_TIMESTAMP
	`
	
	stmt, err := t.db.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()
	
	inserted := 0
	updated := 0
	
	for _, stock := range stocks {
		result, err := stmt.Exec(
			stock.Symbol,
			stock.CompanyName,
			stock.Sector,
			stock.Industry,
			stock.Exchange,
			stock.MarketCap,
			stock.IsActive,
		)
		if err != nil {
			log.Printf("Failed to insert stock %s: %v", stock.Symbol, err)
			continue
		}
		
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			inserted++
		} else {
			updated++
		}
	}
	
	log.Printf("Stock seeding completed: %d inserted, %d updated", inserted, updated)
	return nil
}

// FetchHistoricalData fetches historical data for a specific symbol or all symbols
func (t *TaskRunner) FetchHistoricalData(symbol string) error {
	if symbol != "" {
		return t.fetchHistoricalDataForSymbol(symbol)
	}
	
	// Fetch for all active stocks
	return t.FetchAllHistoricalData()
}

// FetchAllHistoricalData fetches historical data for all active stocks (respects rate limits)
func (t *TaskRunner) FetchAllHistoricalData() error {
	log.Println("Fetching historical data for all active stocks...")
	
	// Get all active stock symbols
	query := `SELECT symbol FROM stocks WHERE is_active = true ORDER BY symbol`
	rows, err := t.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get stock symbols: %w", err)
	}
	defer rows.Close()
	
	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}
		symbols = append(symbols, symbol)
	}
	
	log.Printf("Found %d active stocks to fetch data for", len(symbols))
	
	fetched := 0
	skipped := 0
	
	for i, symbol := range symbols {
		// Check rate limits before each request
		canMake, err := t.alphaVantageClient.CanMakeRequest()
		if err != nil {
			log.Printf("Failed to check rate limit: %v", err)
			break
		}
		if !canMake {
			log.Printf("Rate limit reached after %d stocks. %d stocks skipped.", fetched, len(symbols)-i)
			skipped = len(symbols) - i
			break
		}
		
		log.Printf("Fetching data for %s (%d/%d)...", symbol, i+1, len(symbols))
		if err := t.fetchHistoricalDataForSymbol(symbol); err != nil {
			log.Printf("Warning: Failed to fetch data for %s: %v", symbol, err)
			continue
		}
		
		fetched++
		
		// Respectful delay between requests
		if i < len(symbols)-1 {
			time.Sleep(3 * time.Second)
		}
	}
	
	log.Printf("Historical data fetch completed: %d successful, %d skipped due to rate limits", fetched, skipped)
	
	if skipped > 0 {
		log.Printf("To fetch remaining data, run this task again tomorrow or upgrade to Alpha Vantage premium.")
	}
	
	return nil
}

// fetchHistoricalDataForSymbol fetches and saves historical data for a specific symbol
func (t *TaskRunner) fetchHistoricalDataForSymbol(symbol string) error {
	data, err := t.alphaVantageClient.FetchDailyData(symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch data from Alpha Vantage: %w", err)
	}
	
	if err := t.alphaVantageClient.SaveHistoricalData(symbol, data); err != nil {
		return fmt.Errorf("failed to save data to database: %w", err)
	}
	
	return nil
}

// DatabaseStatus shows current database statistics
func (t *TaskRunner) DatabaseStatus() error {
	log.Println("=== Database Status ===")
	
	// Stock count
	var stockCount int
	if err := t.db.QueryRow("SELECT COUNT(*) FROM stocks").Scan(&stockCount); err != nil {
		return err
	}
	log.Printf("Total stocks: %d", stockCount)
	
	var activeStockCount int
	if err := t.db.QueryRow("SELECT COUNT(*) FROM stocks WHERE is_active = true").Scan(&activeStockCount); err != nil {
		return err
	}
	log.Printf("Active stocks: %d", activeStockCount)
	
	// Historical data count
	var priceCount int
	if err := t.db.QueryRow("SELECT COUNT(*) FROM daily_prices").Scan(&priceCount); err != nil {
		return err
	}
	log.Printf("Historical price records: %d", priceCount)
	
	// Stocks with data
	var stocksWithData int
	if err := t.db.QueryRow("SELECT COUNT(DISTINCT stock_id) FROM daily_prices").Scan(&stocksWithData); err != nil {
		return err
	}
	log.Printf("Stocks with historical data: %d", stocksWithData)
	
	// Date range
	var minDate, maxDate sql.NullTime
	if err := t.db.QueryRow("SELECT MIN(date), MAX(date) FROM daily_prices").Scan(&minDate, &maxDate); err != nil {
		return err
	}
	
	if minDate.Valid && maxDate.Valid {
		log.Printf("Data date range: %s to %s", 
			minDate.Time.Format("2006-01-02"), 
			maxDate.Time.Format("2006-01-02"))
	} else {
		log.Printf("No historical data found")
	}
	
	return nil
}

// ClearCache clears various cached data
func (t *TaskRunner) ClearCache() error {
	log.Println("Clearing cache...")
	
	// Clear old API call logs (keep last 7 days)
	result, err := t.db.Exec("DELETE FROM api_calls WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '7 days'")
	if err != nil {
		return err
	}
	
	rowsDeleted, _ := result.RowsAffected()
	log.Printf("Cleared %d old API call records", rowsDeleted)
	
	return nil
}

// APIStatus shows Alpha Vantage API status
func (t *TaskRunner) APIStatus() error {
	log.Println("=== Alpha Vantage API Status ===")
	
	rateLimit, err := t.alphaVantageClient.GetRateLimit()
	if err != nil {
		return err
	}
	
	log.Printf("Daily limit: %d", rateLimit.DailyLimit)
	log.Printf("Daily used: %d", rateLimit.CurrentDailyCount)
	log.Printf("Daily remaining: %d", rateLimit.RemainingDaily())
	log.Printf("Can make request: %t", rateLimit.CanMakeRequest())
	log.Printf("Last reset: %s", rateLimit.LastResetDate.Format("2006-01-02"))
	
	// Recent API calls
	stats, err := t.alphaVantageClient.GetAPICallStats(1)
	if err != nil {
		return err
	}
	
	if len(stats) > 0 {
		log.Printf("Today's API calls: %d successful, %d failed", 
			stats[0].SuccessfulCalls, stats[0].FailedCalls)
	} else {
		log.Printf("No API calls made today")
	}
	
	return nil
}