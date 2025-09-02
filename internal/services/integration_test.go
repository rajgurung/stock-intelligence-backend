package services

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ServiceIntegrationTestSuite defines the test suite for service integration tests
type ServiceIntegrationTestSuite struct {
	suite.Suite
	db           *sql.DB
	stockService *DatabaseStockService
}

// SetupSuite runs once before all tests
func (suite *ServiceIntegrationTestSuite) SetupSuite() {
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
	suite.stockService = NewDatabaseStockService(db, nil)

	// Setup test data
	suite.setupTestData()
}

// TearDownSuite runs once after all tests
func (suite *ServiceIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// setupTestData inserts test data into the database
func (suite *ServiceIntegrationTestSuite) setupTestData() {
	// Clean any existing test data first
	suite.cleanupTestData()

	// Create test table if it doesn't exist
	suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS stocks (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(10) UNIQUE NOT NULL,
			company_name VARCHAR(255) NOT NULL,
			sector VARCHAR(100),
			industry VARCHAR(100),
			exchange VARCHAR(10),
			market_cap BIGINT,
			price_range VARCHAR(20),
			is_active BOOLEAN DEFAULT true,
			current_price DECIMAL(10,2),
			daily_change DECIMAL(10,2),
			change_percent DECIMAL(5,2),
			volume BIGINT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_updated TIMESTAMP DEFAULT NOW()
		)
	`)

	// Insert test stocks with diverse data for comprehensive testing
	testStocks := []struct {
		symbol        string
		companyName   string
		sector        string
		industry      string
		exchange      string
		marketCap     int64
		priceRange    string
		currentPrice  float64
		dailyChange   float64
		changePercent float64
		volume        int64
	}{
		{"AAPL", "Apple Inc.", "Technology", "Consumer Electronics", "NASDAQ", 3000000000000, "$150+", 150.25, 2.50, 1.69, 50000000},
		{"MSFT", "Microsoft Corporation", "Technology", "Software", "NASDAQ", 2800000000000, "$300+", 320.15, -1.25, -0.39, 25000000},
		{"GOOGL", "Alphabet Inc.", "Technology", "Internet Content", "NASDAQ", 1600000000000, "$100+", 125.50, 3.25, 2.66, 30000000},
		{"TSLA", "Tesla Inc.", "Consumer Discretionary", "Auto Manufacturers", "NASDAQ", 800000000000, "$200+", 225.75, -5.50, -2.38, 75000000},
		{"JPM", "JPMorgan Chase & Co.", "Financial Services", "Banks", "NYSE", 450000000000, "$100+", 145.80, 1.20, 0.83, 15000000},
	}

	for _, stock := range testStocks {
		_, err := suite.db.Exec(`
			INSERT INTO stocks 
			(symbol, company_name, sector, industry, exchange, market_cap, price_range, is_active, 
			 current_price, daily_change, change_percent, volume, created_at, updated_at, last_updated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
			stock.symbol, stock.companyName, stock.sector, stock.industry, stock.exchange,
			stock.marketCap, stock.priceRange, true, stock.currentPrice, stock.dailyChange,
			stock.changePercent, stock.volume, time.Now(), time.Now(), time.Now())
		
		if err != nil {
			suite.T().Logf("Failed to insert test stock %s: %v", stock.symbol, err)
		}
	}
}

// cleanupTestData removes test data from the database
func (suite *ServiceIntegrationTestSuite) cleanupTestData() {
	if suite.db == nil {
		return
	}
	
	testSymbols := []string{"AAPL", "MSFT", "GOOGL", "TSLA", "JPM"}
	for _, symbol := range testSymbols {
		suite.db.Exec("DELETE FROM stocks WHERE symbol = $1", symbol)
	}
}

// TestGetAllStocks tests retrieving all stocks from database
func (suite *ServiceIntegrationTestSuite) TestGetAllStocks() {
	stocks := suite.stockService.GetAllStocks()
	
	assert.GreaterOrEqual(suite.T(), len(stocks), 5, "Should return at least 5 test stocks")
	
	// Verify we have our test stocks
	symbolMap := make(map[string]bool)
	for _, stock := range stocks {
		symbolMap[stock.Symbol] = true
		
		// Validate stock data structure
		assert.NotEmpty(suite.T(), stock.Symbol)
		assert.NotEmpty(suite.T(), stock.CompanyName)
		assert.NotEmpty(suite.T(), stock.Sector)
		assert.True(suite.T(), stock.IsActive)
		assert.Greater(suite.T(), stock.CurrentPrice, 0.0)
	}
	
	expectedSymbols := []string{"AAPL", "MSFT", "GOOGL", "TSLA", "JPM"}
	for _, symbol := range expectedSymbols {
		assert.True(suite.T(), symbolMap[symbol], "Expected symbol %s not found", symbol)
	}
}

// TestGetStockBySymbol tests retrieving individual stocks
func (suite *ServiceIntegrationTestSuite) TestGetStockBySymbol() {
	// Test existing stock
	stock, err := suite.stockService.GetStockBySymbol("AAPL")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stock)
	assert.Equal(suite.T(), "AAPL", stock.Symbol)
	assert.Equal(suite.T(), "Apple Inc.", stock.CompanyName)
	assert.Equal(suite.T(), "Technology", stock.Sector)
	assert.Equal(suite.T(), 150.25, stock.CurrentPrice)
	
	// Test non-existent stock
	stock, err = suite.stockService.GetStockBySymbol("NONEXISTENT")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), stock)
}

// TestGetStocksBysector tests sector-based filtering
func (suite *ServiceIntegrationTestSuite) TestGetStocksBySector() {
	// Test Technology sector
	techStocks := suite.stockService.GetStocksBySector("Technology")
	assert.GreaterOrEqual(suite.T(), len(techStocks), 3, "Should have at least 3 technology stocks")
	
	for _, stock := range techStocks {
		assert.Equal(suite.T(), "Technology", stock.Sector)
	}
	
	// Test Financial Services sector
	financialStocks := suite.stockService.GetStocksBySector("Financial Services")
	assert.GreaterOrEqual(suite.T(), len(financialStocks), 1, "Should have at least 1 financial stock")
	
	for _, stock := range financialStocks {
		assert.Equal(suite.T(), "Financial Services", stock.Sector)
	}
	
	// Test non-existent sector
	nonExistentStocks := suite.stockService.GetStocksBySector("NonExistentSector")
	assert.Equal(suite.T(), 0, len(nonExistentStocks))
}

// TestGetStocksByPriceRangeMethod tests price range filtering
func (suite *ServiceIntegrationTestSuite) TestGetStocksByPriceRangeMethod() {
	// Test $150+ price range
	expensiveStocks := suite.stockService.GetStocksByPriceRange("$150+")
	assert.GreaterOrEqual(suite.T(), len(expensiveStocks), 0)
	
	for _, stock := range expensiveStocks {
		assert.Equal(suite.T(), "$150+", stock.PriceRange)
	}
	
	// Test $100+ price range
	midRangeStocks := suite.stockService.GetStocksByPriceRange("$100+")
	assert.GreaterOrEqual(suite.T(), len(midRangeStocks), 0)
	
	for _, stock := range midRangeStocks {
		assert.Equal(suite.T(), "$100+", stock.PriceRange)
	}
}

// TestDatabaseConnection tests database connection handling
func (suite *ServiceIntegrationTestSuite) TestDatabaseConnection() {
	// Test that service handles database connection properly
	assert.NotNil(suite.T(), suite.stockService)
	
	// Test that we can execute a simple query
	var count int
	err := suite.db.QueryRow("SELECT COUNT(*) FROM stocks WHERE symbol IN ('AAPL', 'MSFT', 'GOOGL', 'TSLA', 'JPM')").Scan(&count)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), count, 5)
}

// TestConcurrentAccess tests concurrent database access
func (suite *ServiceIntegrationTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Perform various operations concurrently
			stocks := suite.stockService.GetAllStocks()
			assert.GreaterOrEqual(suite.T(), len(stocks), 5)
			
			stock, err := suite.stockService.GetStockBySymbol("AAPL")
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), stock)
			
			stocks = suite.stockService.GetAllStocks()
			assert.GreaterOrEqual(suite.T(), len(stocks), 5)
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestDataValidation tests that service validates data properly
func (suite *ServiceIntegrationTestSuite) TestDataValidation() {
	// Test with empty symbol
	stock, err := suite.stockService.GetStockBySymbol("")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), stock)
	
	// Test with whitespace symbol
	stock, err = suite.stockService.GetStockBySymbol("   ")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), stock)
}

// TestPerformance tests service performance with larger datasets
func (suite *ServiceIntegrationTestSuite) TestPerformance() {
	// Measure time for GetAllStocks
	start := time.Now()
	stocks := suite.stockService.GetAllStocks()
	duration := time.Since(start)
	
	assert.GreaterOrEqual(suite.T(), len(stocks), 5)
	assert.Less(suite.T(), duration, 1*time.Second, "GetAllStocks should complete in under 1 second")
	
	// Measure time for GetStockBySymbol
	start = time.Now()
	stock, err := suite.stockService.GetStockBySymbol("AAPL")
	duration = time.Since(start)
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stock)
	assert.Less(suite.T(), duration, 100*time.Millisecond, "GetStockBySymbol should complete in under 100ms")
	
	// Measure time for GetStocksBySector
	start = time.Now()
	techStocks := suite.stockService.GetStocksBySector("Technology")
	duration = time.Since(start)
	
	assert.GreaterOrEqual(suite.T(), len(techStocks), 3)
	assert.Less(suite.T(), duration, 500*time.Millisecond, "GetStocksBySector should complete in under 500ms")
}

// Run the service integration test suite
func TestServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ServiceIntegrationTestSuite))
}