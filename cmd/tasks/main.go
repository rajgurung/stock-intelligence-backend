package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"stock-intelligence-backend/internal/database"
	"stock-intelligence-backend/internal/services"
	"stock-intelligence-backend/internal/tasks"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	taskName := os.Args[1]
	taskArgs := os.Args[2:]

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize services
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	alphaVantageClient := services.NewAlphaVantageClient(apiKey, db)

	// Create task runner
	taskRunner := tasks.NewTaskRunner(db, alphaVantageClient)

	// Execute task
	switch taskName {
	case "db:seed":
		if err := taskRunner.SeedDatabase(); err != nil {
			log.Fatal("Seed task failed:", err)
		}
		log.Println("Database seeded successfully!")

	case "db:seed:stocks":
		if err := taskRunner.SeedStocks(); err != nil {
			log.Fatal("Stock seed task failed:", err)
		}
		log.Println("Stocks seeded successfully!")

	case "data:fetch":
		symbol := ""
		if len(taskArgs) > 0 {
			symbol = strings.ToUpper(taskArgs[0])
		}
		if err := taskRunner.FetchHistoricalData(symbol); err != nil {
			log.Fatal("Data fetch task failed:", err)
		}
		if symbol != "" {
			log.Printf("Historical data fetched for %s successfully!", symbol)
		} else {
			log.Println("Historical data fetched for all stocks successfully!")
		}

	case "data:fetch:all":
		if err := taskRunner.FetchAllHistoricalData(); err != nil {
			log.Fatal("Fetch all data task failed:", err)
		}
		log.Println("All historical data fetched successfully!")

	case "db:status":
		if err := taskRunner.DatabaseStatus(); err != nil {
			log.Fatal("Status check failed:", err)
		}

	case "cache:clear":
		if err := taskRunner.ClearCache(); err != nil {
			log.Fatal("Cache clear failed:", err)
		}
		log.Println("Cache cleared successfully!")

	case "api:status":
		if err := taskRunner.APIStatus(); err != nil {
			log.Fatal("API status check failed:", err)
		}

	default:
		fmt.Printf("Unknown task: %s\n", taskName)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Stock Intelligence Task Runner")
	fmt.Println("Usage: ./tasks <task> [args...]")
	fmt.Println()
	fmt.Println("Available tasks:")
	fmt.Println("  db:seed              - Seed database with initial data (stocks + sample historical data)")
	fmt.Println("  db:seed:stocks       - Seed only stock symbols (no historical data)")
	fmt.Println("  db:status            - Show database status and stock counts")
	fmt.Println("  data:fetch [SYMBOL]  - Fetch historical data for specific symbol (or all if none specified)")
	fmt.Println("  data:fetch:all       - Fetch historical data for all stocks (respects rate limits)")
	fmt.Println("  cache:clear          - Clear all cached data")
	fmt.Println("  api:status           - Show Alpha Vantage API status and rate limits")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./tasks db:seed")
	fmt.Println("  ./tasks data:fetch AAPL")
	fmt.Println("  ./tasks data:fetch:all")
	fmt.Println("  ./tasks db:status")
}