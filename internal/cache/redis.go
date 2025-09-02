package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Test the connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Connected to Redis cache")
	return &RedisCache{
		client: client,
		ctx:    ctx,
	}, nil
}

// SetStockData caches stock data with expiration
func (r *RedisCache) SetStockData(key string, data interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(r.ctx, key, jsonData, expiration).Err()
}

// GetStockData retrieves cached stock data
func (r *RedisCache) GetStockData(key string, dest interface{}) error {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// SetStocksList caches the full stocks list
func (r *RedisCache) SetStocksList(stocks interface{}, expiration time.Duration) error {
	return r.SetStockData("stocks:all", stocks, expiration)
}

// GetStocksList retrieves cached stocks list
func (r *RedisCache) GetStocksList(dest interface{}) error {
	return r.GetStockData("stocks:all", dest)
}

// SetMarketOverview caches market overview data
func (r *RedisCache) SetMarketOverview(overview interface{}, expiration time.Duration) error {
	return r.SetStockData("market:overview", overview, expiration)
}

// GetMarketOverview retrieves cached market overview
func (r *RedisCache) GetMarketOverview(dest interface{}) error {
	return r.GetStockData("market:overview", dest)
}

// SetPerformanceData caches performance rankings
func (r *RedisCache) SetPerformanceData(performance interface{}, expiration time.Duration) error {
	return r.SetStockData("performance:rankings", performance, expiration)
}

// GetPerformanceData retrieves cached performance rankings
func (r *RedisCache) GetPerformanceData(dest interface{}) error {
	return r.GetStockData("performance:rankings", dest)
}

// SetSectorData caches sector-specific stock data
func (r *RedisCache) SetSectorData(sector string, stocks interface{}, expiration time.Duration) error {
	key := "stocks:sector:" + sector
	return r.SetStockData(key, stocks, expiration)
}

// GetSectorData retrieves cached sector data
func (r *RedisCache) GetSectorData(sector string, dest interface{}) error {
	key := "stocks:sector:" + sector
	return r.GetStockData(key, dest)
}

// SetHistoricalData caches historical performance data
func (r *RedisCache) SetHistoricalData(symbol string, days int, data interface{}, expiration time.Duration) error {
	key := "historical:" + symbol + ":" + string(rune(days))
	return r.SetStockData(key, data, expiration)
}

// GetHistoricalData retrieves cached historical data
func (r *RedisCache) GetHistoricalData(symbol string, days int, dest interface{}) error {
	key := "historical:" + symbol + ":" + string(rune(days))
	return r.GetStockData(key, dest)
}

// InvalidateStock removes cached data for a specific stock
func (r *RedisCache) InvalidateStock(symbol string) error {
	pattern := "*" + symbol + "*"
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}
	return nil
}

// InvalidateAll removes all cached stock data
func (r *RedisCache) InvalidateAll() error {
	return r.client.FlushAll(r.ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}