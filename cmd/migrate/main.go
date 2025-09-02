package main

import (
	"flag"
	"log"
	"os"

	"stock-intelligence-backend/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Parse command line flags
	var command = flag.String("command", "up", "Migration command: up, status")
	flag.Parse()

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create migrator
	migrator := database.NewMigrator(db, "./migrations")

	// Execute command
	switch *command {
	case "up":
		if err := migrator.Up(); err != nil {
			log.Fatal("Migration failed:", err)
		}
		log.Println("Migrations completed successfully")

	case "status":
		if err := migrator.Status(); err != nil {
			log.Fatal("Status check failed:", err)
		}

	default:
		log.Printf("Unknown command: %s", *command)
		log.Println("Available commands: up, status")
		os.Exit(1)
	}
}