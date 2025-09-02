package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	baseURL := "http://localhost:8080"
	
	// First check if the server is running
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		log.Fatal("Server is not running. Please start the backend server first: cd backend && go run main.go")
	}
	resp.Body.Close()
	
	log.Println("Backend server is running. Checking system status...")
	
	// Check API status
	resp, err = http.Get(baseURL + "/api/v1/system/api-status")
	if err != nil {
		log.Fatal("Failed to check API status:", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read API status response:", err)
	}
	
	var apiStatus map[string]interface{}
	if err := json.Unmarshal(body, &apiStatus); err != nil {
		log.Fatal("Failed to parse API status:", err)
	}
	
	fmt.Printf("API Status Response: %s\n", string(body))
	
	// Get some stocks to sync
	stockSymbols := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA"}
	
	log.Printf("Triggering manual sync for %d stocks...", len(stockSymbols))
	
	successCount := 0
	for i, symbol := range stockSymbols {
		log.Printf("[%d/%d] Syncing %s...", i+1, len(stockSymbols), symbol)
		
		url := fmt.Sprintf("%s/api/v1/system/sync/%s", baseURL, symbol)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			log.Printf("Failed to sync %s: %v", symbol, err)
			continue
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Failed to read sync response for %s: %v", symbol, err)
			continue
		}
		
		if resp.StatusCode == 200 {
			log.Printf("✓ Successfully triggered sync for %s", symbol)
			successCount++
		} else {
			log.Printf("✗ Failed to sync %s: %s", symbol, string(body))
		}
		
		// Rate limiting - wait between requests
		if i < len(stockSymbols)-1 {
			log.Println("Waiting 15 seconds before next sync...")
			time.Sleep(15 * time.Second)
		}
	}
	
	log.Printf("Sync completed: %d/%d stocks triggered successfully", successCount, len(stockSymbols))
	
	// Wait a moment then check the data
	log.Println("Waiting 10 seconds then checking database...")
	time.Sleep(10 * time.Second)
	
	// Check stocks endpoint
	resp, err = http.Get(baseURL + "/api/v1/stocks")
	if err != nil {
		log.Printf("Failed to check stocks: %v", err)
		return
	}
	defer resp.Body.Close()
	
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read stocks response: %v", err)
		return
	}
	
	var stocksResp map[string]interface{}
	if err := json.Unmarshal(body, &stocksResp); err != nil {
		log.Printf("Failed to parse stocks response: %v", err)
		return
	}
	
	if count, ok := stocksResp["count"].(float64); ok {
		log.Printf("Current stocks in API: %.0f", count)
		if count > 0 {
			log.Println("✓ Success! Database now has stock data.")
		} else {
			log.Println("⚠ No stocks returned. Check logs for API rate limits or errors.")
		}
	}
}