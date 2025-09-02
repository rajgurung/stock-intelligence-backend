package main

import (
	"database/sql"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Scheduler handles background data fetching tasks
type Scheduler struct {
	db *sql.DB
}

func main() {
	log.Println("🕐 Starting Stock Data Scheduler...")

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

	scheduler := &Scheduler{db: db}

	// Run initial fetch immediately
	log.Println("🚀 Running initial data fetch...")
	scheduler.runDataFetcher()

	// Set up periodic scheduling for daily compliance (24 hours)
	ticker := time.NewTicker(24 * time.Hour) // Run once daily for API compliance
	defer ticker.Stop()

	log.Println("⏰ Scheduler started - will run daily at this time for API compliance")

	for {
		select {
		case <-ticker.C:
			log.Println("⏰ Scheduled run starting...")
			scheduler.runDataFetcher()
		}
	}
}

// runDataFetcher executes the data fetcher command
func (s *Scheduler) runDataFetcher() {
	log.Println("📊 Starting data fetch process...")
	
	cmd := exec.Command("go", "run", "./cmd/data-fetcher/main.go")
	cmd.Dir = "/Users/rajg/Codes/stock_app/backend"
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ Data fetcher failed: %v", err)
		log.Printf("Output: %s", string(output))
	} else {
		log.Println("✅ Data fetcher completed successfully")
		log.Printf("Output: %s", string(output))
	}

	// Log the execution
	s.logScheduledRun(err == nil)
}

// logScheduledRun logs when the scheduler runs
func (s *Scheduler) logScheduledRun(success bool) {
	status := "success"
	if !success {
		status = "failed"
	}

	_, err := s.db.Exec(`
		INSERT INTO api_calls 
		(service_name, endpoint, request_params, response_status, response_body, created_at)
		VALUES ('scheduler', 'data_fetch', '{}', $1, $2, CURRENT_TIMESTAMP)
	`, map[bool]int{true: 200, false: 500}[success], status)
	
	if err != nil {
		log.Printf("Warning: Failed to log scheduled run: %v", err)
	}
}