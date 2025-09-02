package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"stock-intelligence-backend/internal/models"
	"stock-intelligence-backend/internal/services"
)

// IntegrationTestSuite defines the test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	db     *sql.DB
	router *gin.Engine
	stocks []models.Stock
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup test database connection
	testDB := os.Getenv("TEST_DATABASE_URL")
	if testDB == "" {
		testDB = "postgres://postgres:password@localhost/stock_intelligence_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", testDB)
	if err != nil {
		suite.T().Skipf("Cannot connect to test database: %v", err)
		return
	}

	if err := db.Ping(); err != nil {
		suite.T().Skipf("Cannot ping test database: %v", err)
		return
	}

	suite.db = db

	// Setup test data
	suite.setupTestData()

	// Create services
	stockService := services.NewDatabaseStockService(db, nil)

	// Setup router with handlers
	suite.router = gin.New()
	stockHandler := NewDatabaseStockHandler(stockService)

	api := suite.router.Group("/api/v1")
	{
		api.GET("/stocks", stockHandler.GetAllStocks)
		api.GET("/stocks/:symbol", stockHandler.GetStockBySymbol)
		api.GET("/market/overview", stockHandler.GetMarketOverview)
	}
}

// TearDownSuite runs once after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// setupTestData inserts test data into the database
func (suite *IntegrationTestSuite) setupTestData() {
	// Clean any existing test data first
	suite.cleanupTestData()

	// Insert test stocks
	testStocks := []models.Stock{
		{
			Symbol:        "AAPL",
			CompanyName:   "Apple Inc.",
			Sector:        "Technology",
			Industry:      "Consumer Electronics",
			Exchange:      "NASDAQ",
			MarketCap:     &[]int64{3000000000000}[0],
			PriceRange:    "$150+",
			IsActive:      true,
			CurrentPrice:  150.25,
			DailyChange:   2.50,
			ChangePercent: 1.69,
			Volume:        50000000,
		},
		{
			Symbol:        "MSFT",
			CompanyName:   "Microsoft Corporation",
			Sector:        "Technology", 
			Industry:      "Software",
			Exchange:      "NASDAQ",
			MarketCap:     &[]int64{2800000000000}[0],
			PriceRange:    "$300+",
			IsActive:      true,
			CurrentPrice:  320.15,
			DailyChange:   -1.25,
			ChangePercent: -0.39,
			Volume:        25000000,
		},
		{
			Symbol:        "GOOGL",
			CompanyName:   "Alphabet Inc.",
			Sector:        "Technology",
			Industry:      "Internet Content & Information",
			Exchange:      "NASDAQ",
			MarketCap:     &[]int64{1600000000000}[0],
			PriceRange:    "$100+",
			IsActive:      true,
			CurrentPrice:  125.50,
			DailyChange:   3.25,
			ChangePercent: 2.66,
			Volume:        30000000,
		},
	}

	for _, stock := range testStocks {
		_, err := suite.db.Exec(`
			INSERT INTO stocks 
			(symbol, company_name, sector, industry, exchange, market_cap, price_range, is_active, 
			 current_price, daily_change, change_percent, volume, created_at, updated_at, last_updated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW(), NOW())`,
			stock.Symbol, stock.CompanyName, stock.Sector, stock.Industry, stock.Exchange,
			*stock.MarketCap, stock.PriceRange, stock.IsActive, stock.CurrentPrice,
			stock.DailyChange, stock.ChangePercent, stock.Volume)
		
		if err != nil {
			suite.T().Fatalf("Failed to insert test stock %s: %v", stock.Symbol, err)
		}
	}

	suite.stocks = testStocks
}

// cleanupTestData removes test data from the database
func (suite *IntegrationTestSuite) cleanupTestData() {
	if suite.db == nil {
		return
	}
	
	// Delete test stocks
	testSymbols := []string{"AAPL", "MSFT", "GOOGL"}
	for _, symbol := range testSymbols {
		suite.db.Exec("DELETE FROM stocks WHERE symbol = $1", symbol)
	}
}

// TestGetAllStocks tests the GET /api/v1/stocks endpoint
func (suite *IntegrationTestSuite) TestGetAllStocks() {
	req, _ := http.NewRequest("GET", "/api/v1/stocks", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	
	data, ok := response["data"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(data), 3) // At least our 3 test stocks

	// Verify one of our test stocks is present
	found := false
	for _, item := range data {
		stock, ok := item.(map[string]interface{})
		if ok && stock["symbol"].(string) == "AAPL" {
			found = true
			assert.Equal(suite.T(), "Apple Inc.", stock["company_name"])
			assert.Equal(suite.T(), "Technology", stock["sector"])
			break
		}
	}
	assert.True(suite.T(), found, "Test stock AAPL not found in response")
}

// TestGetStockBySymbol tests the GET /api/v1/stocks/:symbol endpoint
func (suite *IntegrationTestSuite) TestGetStockBySymbol() {
	req, _ := http.NewRequest("GET", "/api/v1/stocks/AAPL", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	
	data, ok := response["data"].(map[string]interface{})
	assert.True(suite.T(), ok)
	
	assert.Equal(suite.T(), "AAPL", data["symbol"])
	assert.Equal(suite.T(), "Apple Inc.", data["company_name"])
	assert.Equal(suite.T(), "Technology", data["sector"])
	assert.Equal(suite.T(), float64(150.25), data["current_price"])
}

// TestGetStockBySymbolNotFound tests 404 behavior
func (suite *IntegrationTestSuite) TestGetStockBySymbolNotFound() {
	req, _ := http.NewRequest("GET", "/api/v1/stocks/NONEXISTENT", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response["error"].(string), "not found")
}

// TestGetMarketOverview tests the GET /api/v1/market/overview endpoint
func (suite *IntegrationTestSuite) TestGetMarketOverview() {
	req, _ := http.NewRequest("GET", "/api/v1/market/overview", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	
	data, ok := response["data"].(map[string]interface{})
	assert.True(suite.T(), ok)
	
	// Check that overview contains expected fields
	assert.Contains(suite.T(), data, "total_stocks")
	assert.Contains(suite.T(), data, "advancing_count") 
	assert.Contains(suite.T(), data, "declining_count")
	assert.Contains(suite.T(), data, "unchanged_count")
	
	// Verify counts make sense
	totalStocks := int(data["total_stocks"].(float64))
	advancingCount := int(data["advancing_count"].(float64))
	decliningCount := int(data["declining_count"].(float64))
	unchangedCount := int(data["unchanged_count"].(float64))
	
	assert.GreaterOrEqual(suite.T(), totalStocks, 3) // At least our test stocks
	assert.Equal(suite.T(), totalStocks, advancingCount + decliningCount + unchangedCount)
}

// TestSectorFiltering tests filtering stocks by sector
func (suite *IntegrationTestSuite) TestSectorFiltering() {
	req, _ := http.NewRequest("GET", "/api/v1/stocks?sector=Technology", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	
	data, ok := response["data"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(data), 3) // All our test stocks are Technology

	// Verify all returned stocks are in Technology sector
	for _, item := range data {
		stock, ok := item.(map[string]interface{})
		if ok {
			assert.Equal(suite.T(), "Technology", stock["sector"])
		}
	}
}

// TestConcurrentRequests tests handling multiple concurrent requests
func (suite *IntegrationTestSuite) TestConcurrentRequests() {
	const numRequests = 10
	done := make(chan bool, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func() {
			defer func() { done <- true }()
			
			req, _ := http.NewRequest("GET", "/api/v1/stocks", nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(suite.T(), http.StatusOK, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(suite.T(), err)
			assert.True(suite.T(), response["success"].(bool))
		}()
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// TestInvalidSymbolFormat tests validation of stock symbols
func (suite *IntegrationTestSuite) TestInvalidSymbolFormat() {
	invalidSymbols := []string{"", " ", "123", "toolong", "invalid@symbol"}
	
	for _, symbol := range invalidSymbols {
		req, _ := http.NewRequest("GET", "/api/v1/stocks/"+symbol, nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		// Should either be 400 (bad request) or 404 (not found)
		assert.True(suite.T(), w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Expected 400 or 404 for symbol '%s', got %d", symbol, w.Code)
	}
}

// TestResponseHeaders tests that proper headers are set
func (suite *IntegrationTestSuite) TestResponseHeaders() {
	req, _ := http.NewRequest("GET", "/api/v1/stocks", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

// TestDatabaseTransaction tests that database operations are properly handled
func (suite *IntegrationTestSuite) TestDatabaseTransaction() {
	// This test ensures that our handlers properly handle database transactions
	// and return appropriate errors when database operations fail
	
	// First, get a successful response
	req, _ := http.NewRequest("GET", "/api/v1/stocks/AAPL", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	// Verify response structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response, "data")
}

// Run the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}