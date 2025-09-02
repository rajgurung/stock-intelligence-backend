package services

import (
	"database/sql"
	"testing"
	"time"

	"stock-intelligence-backend/internal/cache"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseStockService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisCache := &cache.RedisCache{}
	service := NewDatabaseStockService(db, redisCache)
	
	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, redisCache, service.cache)
}

func TestGetAllStocks_DatabaseQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock the database query
	rows := sqlmock.NewRows([]string{
		"id", "symbol", "company_name", "sector", "industry", "market_cap",
		"price_range", "exchange", "is_active", "created_at", "updated_at",
		"current_price", "daily_change", "change_percent", "volume", "last_updated",
	}).AddRow(
		1, "AAPL", "Apple Inc.", "Technology", "Consumer Electronics", int64(3000000000000),
		"$100+", "NASDAQ", true, time.Now(), time.Now(),
		150.0, 2.5, 1.69, int64(50000000), time.Now(),
	).AddRow(
		2, "MSFT", "Microsoft Corporation", "Technology", "Software", int64(2800000000000),
		"$100+", "NASDAQ", true, time.Now(), time.Now(),
		380.0, -1.2, -0.31, int64(30000000), time.Now(),
	)

	// Expect any query starting with SELECT
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	service := NewDatabaseStockService(db, nil) // No cache for this test
	stocks := service.GetAllStocks()

	assert.Len(t, stocks, 2)
	assert.Equal(t, "AAPL", stocks[0].Symbol)
	assert.Equal(t, "MSFT", stocks[1].Symbol)
	assert.Equal(t, 150.0, stocks[0].CurrentPrice)
	assert.Equal(t, 380.0, stocks[1].CurrentPrice)
}

func TestGetStockBySymbol_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").
		WithArgs("INVALID").
		WillReturnError(sql.ErrNoRows)

	service := NewDatabaseStockService(db, nil)
	stock, err := service.GetStockBySymbol("INVALID")

	assert.Error(t, err)
	assert.Nil(t, stock)
	assert.Contains(t, err.Error(), "stock not found")
}

func TestGetStocksBySector(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock the GetAllStocks call that happens inside GetStocksBySector
	rows := sqlmock.NewRows([]string{
		"id", "symbol", "company_name", "sector", "industry", "market_cap",
		"price_range", "exchange", "is_active", "created_at", "updated_at",
		"current_price", "daily_change", "change_percent", "volume", "last_updated",
	}).AddRow(
		1, "AAPL", "Apple Inc.", "Technology", "Consumer Electronics", int64(3000000000000),
		"$100+", "NASDAQ", true, time.Now(), time.Now(),
		150.0, 2.5, 1.69, int64(50000000), time.Now(),
	).AddRow(
		2, "MSFT", "Microsoft Corporation", "Technology", "Software", int64(2800000000000),
		"$100+", "NASDAQ", true, time.Now(), time.Now(),
		380.0, -1.2, -0.31, int64(30000000), time.Now(),
	).AddRow(
		3, "WMT", "Walmart Inc.", "Consumer Staples", "Retail", int64(520000000000),
		"$50-100", "NYSE", true, time.Now(), time.Now(),
		97.0, 0.87, 0.90, int64(15000000), time.Now(),
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	service := NewDatabaseStockService(db, nil)
	technologyStocks := service.GetStocksBySector("Technology")

	assert.Len(t, technologyStocks, 2)
	assert.Equal(t, "AAPL", technologyStocks[0].Symbol)
	assert.Equal(t, "MSFT", technologyStocks[1].Symbol)
	assert.Equal(t, "Technology", technologyStocks[0].Sector)
	assert.Equal(t, "Technology", technologyStocks[1].Sector)
}

func TestGetDB(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewDatabaseStockService(db, nil)
	returnedDB := service.GetDB()

	assert.Equal(t, db, returnedDB)
}

// Benchmark tests for performance validation
func BenchmarkGetAllStocks(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	// Create mock data for 100 stocks
	rows := sqlmock.NewRows([]string{
		"id", "symbol", "company_name", "sector", "industry", "market_cap",
		"price_range", "exchange", "is_active", "created_at", "updated_at",
		"current_price", "daily_change", "change_percent", "volume", "last_updated",
	})

	for i := 1; i <= 100; i++ {
		rows.AddRow(
			i, "SYM"+string(rune(i)), "Company "+string(rune(i)), "Technology", "Software", int64(1000000000),
			"$50-100", "NASDAQ", true, time.Now(), time.Now(),
			100.0, 1.0, 1.0, int64(1000000), time.Now(),
		)
	}

	// Set up expectation for benchmark runs
	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("SELECT").WillReturnRows(rows)
	}

	service := NewDatabaseStockService(db, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetAllStocks()
	}
}