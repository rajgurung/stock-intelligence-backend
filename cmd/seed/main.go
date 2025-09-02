package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"stock-intelligence-backend/internal/database"
	"stock-intelligence-backend/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: No .env file found: %v", err)
	}

	log.Println("=== Stock Intelligence Platform - Database Seeder ===")
	log.Println()

	// Check environment
	env := strings.ToLower(os.Getenv("NODE_ENV"))
	if env == "" {
		env = "development"
	}
	log.Printf("Environment: %s", env)

	// Check API key
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	if apiKey == "" || apiKey == "your_alpha_vantage_api_key_here" || apiKey == "your_alpha_vantage_api_key_here" {
		log.Println("❌ ALPHA_VANTAGE_API_KEY is not configured.")
		log.Println("📋 To get an API key:")
		log.Println("   1. Visit: https://www.alphavantage.co/support/#api-key")
		log.Println("   2. Fill out the form (takes 2 minutes)")
		log.Println("   3. Update your .env file: ALPHA_VANTAGE_API_KEY=your_actual_key")
		log.Println()
		log.Fatal("❌ Cannot proceed without valid API key.")
	}

	log.Println("✅ Alpha Vantage API key found")
	log.Println("🚀 Starting database seeding process...")

	// Initialize database
	db, err := database.InitializeDatabase()
	if err != nil {
		log.Fatal("❌ Failed to initialize database:", err)
	}
	defer db.Close()

	// 🔒 PRODUCTION SAFETY: Check for existing data
	existingCount, err := checkExistingData(db)
	if err != nil {
		log.Fatal("❌ Failed to check existing data:", err)
	}

	if existingCount > 0 {
		log.Printf("⚠️  Database already contains %d price records", existingCount)
		
		// Production protection: Never overwrite in production
		if env == "production" {
			log.Println()
			log.Println("🔒 PRODUCTION PROTECTION ACTIVATED")
			log.Println("❌ Refusing to overwrite existing data in production environment.")
			log.Println("💡 To seed fresh data in production:")
			log.Println("   1. Backup existing data first")
			log.Println("   2. Clear daily_prices table manually")
			log.Println("   3. Re-run this script")
			log.Println()
			log.Fatal("❌ Seeding aborted for production safety.")
		}

		// Development/Test: Ask for permission
		if !askForPermission(existingCount, env) {
			log.Println("✋ Seeding cancelled by user.")
			return
		}

		// Clear existing data
		log.Println("🧹 Clearing existing price data...")
		if err := clearExistingData(db); err != nil {
			log.Fatal("❌ Failed to clear existing data:", err)
		}
		log.Println("✅ Existing data cleared")
	}

	// Create Alpha Vantage client
	alphaVantageClient := services.NewAlphaVantageClient(apiKey, db)

	// Get list of stocks to seed
	stocks, err := getStocksToSeed(db)
	if err != nil {
		log.Fatal("❌ Failed to get stocks list:", err)
	}

	log.Printf("📊 Found %d stocks to seed", len(stocks))
se
	log.Println()
	log.Println("📡 Starting Alpha Vantage API data fetching...")
	log.Printf("⏱️  Rate limit: 15-second delays between calls (respecting free tier limits)")
	log.Println()

	// Seed data for each stock
	successful := 0
	failed := 0
	
	for i, symbol := range stocks {
		log.Printf("📈 [%d/%d] Fetching data for %s...", i+1, len(stocks), symbol)
		
		err := seedStockData(alphaVantageClient, symbol)
		if err != nil {
			log.Printf("❌ Failed to seed %s: %v", symbol, err)
			failed++
		} else {
			log.Printf("✅ Successfully seeded %s", symbol)
			successful++
		}

		// Rate limiting: Alpha Vantage allows 5 calls per minute for free tier
		if i < len(stocks)-1 {
			log.Printf("⏳ Waiting 15 seconds before next API call...")
			time.Sleep(15 * time.Second)
		}
	}

	log.Println()
	log.Printf("🎯 Seeding completed: %d successful, %d failed", successful, failed)
	
	// Verify seeded data
	if successful > 0 {
		verifySeededData(db)
	}
}

func getStocksToSeed(db *sql.DB) ([]string, error) {
	query := `
		SELECT symbol 
		FROM stocks 
		WHERE is_active = true 
		ORDER BY symbol 
		LIMIT 10
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	
	return symbols, nil
}

func seedStockData(client *services.AlphaVantageClient, symbol string) error {
	// Fetch daily time series data for the stock
	log.Printf("Fetching Alpha Vantage data for %s", symbol)
	
	// Fetch data from Alpha Vantage API
	data, err := client.FetchDailyData(symbol)
	if err != nil {
		return err
	}
	
	// Save historical data to database
	return client.SaveHistoricalData(symbol, data)
}

func verifySeededData(db *sql.DB) {
	log.Println("Verifying seeded data...")
	
	// Check total daily prices records
	var count int
	query := "SELECT COUNT(*) FROM daily_prices"
	if err := db.QueryRow(query).Scan(&count); err != nil {
		log.Printf("Error checking daily_prices count: %v", err)
		return
	}
	
	log.Printf("Total daily_prices records: %d", count)
	
	// Check latest dates
	query = `
		SELECT dp.stock_id, s.symbol, MAX(dp.date) as latest_date
		FROM daily_prices dp
		JOIN stocks s ON s.id = dp.stock_id
		GROUP BY dp.stock_id, s.symbol
		ORDER BY s.symbol
		LIMIT 5
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error checking latest dates: %v", err)
		return
	}
	defer rows.Close()
	
	log.Println("Sample of latest data dates:")
	for rows.Next() {
		var stockId int
		var symbol string
		var latestDate time.Time
		
		if err := rows.Scan(&stockId, &symbol, &latestDate); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		log.Printf("  %s: %s", symbol, latestDate.Format("2006-01-02"))
	}
}

// checkExistingData returns the count of existing daily_prices records
func checkExistingData(db *sql.DB) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM daily_prices"
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// askForPermission prompts user to confirm overwriting existing data
func askForPermission(existingCount int, env string) bool {
	log.Println()
	log.Printf("⚠️  This will DELETE %d existing price records!", existingCount)
	log.Printf("🌍 Current environment: %s", env)
	log.Println()
	log.Print("❓ Do you want to continue? (type 'yes' to confirm): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read input: %v", err)
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	confirmed := response == "yes" || response == "y"
	
	if confirmed {
		log.Println("✅ User confirmed data overwrite")
	} else {
		log.Println("❌ User cancelled operation")
	}
	
	return confirmed
}

// clearExistingData removes all daily_prices records
func clearExistingData(db *sql.DB) error {
	query := "DELETE FROM daily_prices"
	result, err := db.Exec(query)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err == nil {
		log.Printf("🗑️  Deleted %d price records", rowsAffected)
	}
	
	return nil
}