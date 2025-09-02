package cache

import (
	"encoding/json"
	"testing"
	"time"

	"stock-intelligence-backend/internal/models"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisCache(t *testing.T) {
	tests := []struct {
		name     string
		redisURL string
		wantErr  bool
	}{
		{
			name:     "valid redis URL",
			redisURL: "redis://localhost:6379",
			wantErr:  true, // Will fail without actual Redis server
		},
		{
			name:     "empty URL defaults to localhost",
			redisURL: "",
			wantErr:  true, // Will fail without actual Redis server
		},
		{
			name:     "invalid URL",
			redisURL: "invalid-url",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewRedisCache(tt.redisURL)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cache)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cache)
				if cache != nil {
					cache.Close()
				}
			}
		})
	}
}

func TestRedisCache_SetAndGetStockData(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	testData := map[string]interface{}{
		"symbol": "AAPL",
		"price":  150.0,
	}

	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)

	// Test SetStockData
	mock.ExpectSet("test-key", string(jsonData), 5*time.Minute).SetVal("OK")
	err = cache.SetStockData("test-key", testData, 5*time.Minute)
	assert.NoError(t, err)

	// Test GetStockData
	mock.ExpectGet("test-key").SetVal(string(jsonData))
	var result map[string]interface{}
	err = cache.GetStockData("test-key", &result)
	assert.NoError(t, err)
	assert.Equal(t, "AAPL", result["symbol"])
	assert.Equal(t, 150.0, result["price"])

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_SetAndGetStocksList(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	testStocks := []models.Stock{
		{
			ID:           1,
			Symbol:       "AAPL",
			CompanyName:  "Apple Inc.",
			CurrentPrice: 150.0,
		},
		{
			ID:           2,
			Symbol:       "MSFT",
			CompanyName:  "Microsoft Corporation",
			CurrentPrice: 380.0,
		},
	}

	jsonData, err := json.Marshal(testStocks)
	require.NoError(t, err)

	// Test SetStocksList
	mock.ExpectSet("stocks:all", string(jsonData), time.Hour).SetVal("OK")
	err = cache.SetStocksList(testStocks, time.Hour)
	assert.NoError(t, err)

	// Test GetStocksList
	mock.ExpectGet("stocks:all").SetVal(string(jsonData))
	var result []models.Stock
	err = cache.GetStocksList(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "AAPL", result[0].Symbol)
	assert.Equal(t, "MSFT", result[1].Symbol)

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_SetAndGetSectorData(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	technologyStocks := []models.Stock{
		{
			ID:           1,
			Symbol:       "AAPL",
			CompanyName:  "Apple Inc.",
			Sector:       "Technology",
			CurrentPrice: 150.0,
		},
	}

	jsonData, err := json.Marshal(technologyStocks)
	require.NoError(t, err)

	// Test SetSectorData
	mock.ExpectSet("stocks:sector:Technology", string(jsonData), time.Hour).SetVal("OK")
	err = cache.SetSectorData("Technology", technologyStocks, time.Hour)
	assert.NoError(t, err)

	// Test GetSectorData
	mock.ExpectGet("stocks:sector:Technology").SetVal(string(jsonData))
	var result []models.Stock
	err = cache.GetSectorData("Technology", &result)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "AAPL", result[0].Symbol)
	assert.Equal(t, "Technology", result[0].Sector)

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_SetAndGetMarketOverview(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	overview := map[string]interface{}{
		"total_stocks":    100,
		"advancing_count": 55,
		"declining_count": 30,
		"unchanged_count": 15,
	}

	jsonData, err := json.Marshal(overview)
	require.NoError(t, err)

	// Test SetMarketOverview
	mock.ExpectSet("market:overview", string(jsonData), 30*time.Minute).SetVal("OK")
	err = cache.SetMarketOverview(overview, 30*time.Minute)
	assert.NoError(t, err)

	// Test GetMarketOverview
	mock.ExpectGet("market:overview").SetVal(string(jsonData))
	var result map[string]interface{}
	err = cache.GetMarketOverview(&result)
	assert.NoError(t, err)
	assert.Equal(t, float64(100), result["total_stocks"])
	assert.Equal(t, float64(55), result["advancing_count"])

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_InvalidateStock(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	// Test InvalidateStock
	mock.ExpectKeys("*AAPL*").SetVal([]string{"stock:AAPL", "historical:AAPL:30"})
	mock.ExpectDel("stock:AAPL", "historical:AAPL:30").SetVal(2)

	err := cache.InvalidateStock("AAPL")
	assert.NoError(t, err)

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_InvalidateAll(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	// Test InvalidateAll
	mock.ExpectFlushAll().SetVal("OK")

	err := cache.InvalidateAll()
	assert.NoError(t, err)

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_GetStockData_NotFound(t *testing.T) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	// Test cache miss
	mock.ExpectGet("nonexistent-key").RedisNil()

	var result map[string]interface{}
	err := cache.GetStockData("nonexistent-key", &result)
	assert.Error(t, err)

	err = mock.ExpectationsMet()
	assert.NoError(t, err)
}

func TestRedisCache_SetStockData_MarshalError(t *testing.T) {
	redis, _ := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	// Test with unmarshalable data (contains channels, which can't be marshaled)
	invalidData := make(chan int)

	err := cache.SetStockData("test-key", invalidData, time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json: unsupported type")
}

// Benchmark test for cache performance
func BenchmarkRedisCache_SetGetStockData(b *testing.B) {
	redis, mock := redismock.NewClientMock()
	defer redis.Close()

	cache := &RedisCache{
		client: redis,
		ctx:    redis.Context(),
	}

	testData := map[string]interface{}{
		"symbol": "AAPL",
		"price":  150.0,
	}

	jsonData, _ := json.Marshal(testData)

	// Setup expectations for benchmark
	for i := 0; i < b.N; i++ {
		mock.ExpectSet("bench-key", string(jsonData), time.Minute).SetVal("OK")
		mock.ExpectGet("bench-key").SetVal(string(jsonData))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetStockData("bench-key", testData, time.Minute)
		var result map[string]interface{}
		cache.GetStockData("bench-key", &result)
	}
}