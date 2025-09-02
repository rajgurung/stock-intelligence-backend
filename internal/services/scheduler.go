package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"stock-intelligence-backend/internal/cache"

	"github.com/robfig/cron/v3"
)

type SchedulerService struct {
	cron             *cron.Cron
	db               *sql.DB
	alphaVantageClient *AlphaVantageClient
	cache            *cache.RedisCache
	mu               sync.RWMutex
	isRunning        bool
	ctx              context.Context
	cancel           context.CancelFunc
	lastDataSync     time.Time
	syncErrors       []string
}

type DataSyncStatus struct {
	IsRunning     bool      `json:"is_running"`
	LastSync      time.Time `json:"last_sync"`
	NextSync      time.Time `json:"next_sync"`
	TotalStocks   int       `json:"total_stocks"`
	ProcessedToday int      `json:"processed_today"`
	Errors        []string  `json:"errors,omitempty"`
}

func NewSchedulerService(db *sql.DB, alphaVantageClient *AlphaVantageClient, redisCache *cache.RedisCache) *SchedulerService {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create cron with seconds precision for more flexible scheduling
	c := cron.New(cron.WithSeconds())
	
	service := &SchedulerService{
		cron:               c,
		db:                 db,
		alphaVantageClient: alphaVantageClient,
		cache:              redisCache,
		ctx:                ctx,
		cancel:             cancel,
		syncErrors:         make([]string, 0),
	}
	
	return service
}

// Start initializes and starts the scheduler
func (s *SchedulerService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isRunning {
		return nil
	}
	
	// Schedule hourly data sync job at the top of each hour
	_, err := s.cron.AddFunc("0 0 * * * *", s.syncStockDataJob)
	if err != nil {
		return err
	}
	
	// Schedule daily cleanup job at 2 AM
	_, err = s.cron.AddFunc("0 0 2 * * *", s.cleanupOldDataJob)
	if err != nil {
		return err
	}
	
	// Schedule rate limit reset job every hour
	_, err = s.cron.AddFunc("0 0 * * * *", s.resetRateLimitsJob)
	if err != nil {
		return err
	}
	
	s.cron.Start()
	s.isRunning = true
	
	log.Println("Scheduler service started successfully")
	log.Println("Jobs scheduled:")
	log.Println("  - Stock data sync: Every hour at :00 minutes")
	log.Println("  - Cleanup old data: Daily at 2:00 AM")
	log.Println("  - Rate limit reset: Every hour at :00 minutes")
	
	return nil
}

// Stop gracefully stops the scheduler
func (s *SchedulerService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return
	}
	
	s.cancel()
	s.cron.Stop()
	s.isRunning = false
	
	log.Println("Scheduler service stopped")
}

// syncStockDataJob fetches data for one stock per hour to respect rate limits
func (s *SchedulerService) syncStockDataJob() {
	log.Println("Starting hourly stock data sync job")
	
	select {
	case <-s.ctx.Done():
		log.Println("Sync job cancelled")
		return
	default:
	}
	
	// Check if we can make an API request
	canMake, err := s.alphaVantageClient.CanMakeRequest()
	if err != nil {
		s.addError("Failed to check rate limit: " + err.Error())
		return
	}
	
	if !canMake {
		log.Println("Rate limit reached, skipping this sync cycle")
		return
	}
	
	// Get next stock to sync
	symbol, err := s.getNextStockToSync()
	if err != nil {
		s.addError("Failed to get next stock to sync: " + err.Error())
		return
	}
	
	if symbol == "" {
		log.Println("No stocks need syncing at this time")
		return
	}
	
	// Fetch and save data for the stock
	log.Printf("Syncing data for %s", symbol)
	
	data, err := s.alphaVantageClient.FetchDailyData(symbol)
	if err != nil {
		s.addError("Failed to fetch data for " + symbol + ": " + err.Error())
		return
	}
	
	err = s.alphaVantageClient.SaveHistoricalData(symbol, data)
	if err != nil {
		s.addError("Failed to save data for " + symbol + ": " + err.Error())
		return
	}
	
	// Invalidate all caches immediately when new data arrives
	if s.cache != nil {
		err = s.cache.InvalidateAll()
		if err != nil {
			log.Printf("Warning: Failed to invalidate cache after data update: %v", err)
		} else {
			log.Printf("🔄 Cache invalidated after data update for %s", symbol)
		}
	}
	
	// Update stock's last sync time
	err = s.updateStockSyncTime(symbol)
	if err != nil {
		s.addError("Failed to update sync time for " + symbol + ": " + err.Error())
	}
	
	s.mu.Lock()
	s.lastDataSync = time.Now()
	s.mu.Unlock()
	
	log.Printf("✅ Successfully synced data for %s", symbol)
}

// getNextStockToSync returns the stock symbol that needs syncing most urgently
func (s *SchedulerService) getNextStockToSync() (string, error) {
	query := `
		SELECT s.symbol 
		FROM stocks s
		LEFT JOIN daily_prices dp ON s.id = dp.stock_id
		WHERE s.is_active = true
		GROUP BY s.id, s.symbol
		ORDER BY MAX(dp.date) ASC NULLS FIRST, s.symbol
		LIMIT 1
	`
	
	var symbol string
	err := s.db.QueryRow(query).Scan(&symbol)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	
	return symbol, nil
}

// updateStockSyncTime updates the updated_at timestamp for a stock
func (s *SchedulerService) updateStockSyncTime(symbol string) error {
	query := `UPDATE stocks SET updated_at = CURRENT_TIMESTAMP WHERE symbol = $1`
	_, err := s.db.Exec(query, symbol)
	return err
}

// cleanupOldDataJob removes old API call logs and performs maintenance
func (s *SchedulerService) cleanupOldDataJob() {
	log.Println("Starting daily cleanup job")
	
	// Keep API call logs for last 30 days
	query := `DELETE FROM api_calls WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '30 days'`
	result, err := s.db.Exec(query)
	if err != nil {
		s.addError("Failed to cleanup old API calls: " + err.Error())
		return
	}
	
	rowsDeleted, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old API call records", rowsDeleted)
	
	// Clear error list if it gets too long
	s.mu.Lock()
	if len(s.syncErrors) > 50 {
		s.syncErrors = s.syncErrors[len(s.syncErrors)-25:] // Keep last 25 errors
	}
	s.mu.Unlock()
	
	log.Println("Daily cleanup job completed")
}

// resetRateLimitsJob ensures rate limits are properly reset
func (s *SchedulerService) resetRateLimitsJob() {
	// The database trigger handles most of this, but we can add extra validation here
	query := `
		UPDATE api_rate_limits 
		SET current_daily_count = 0,
		    current_hourly_count = 0,
		    last_reset_date = CURRENT_DATE,
		    last_reset_hour = EXTRACT(HOUR FROM CURRENT_TIMESTAMP),
		    updated_at = CURRENT_TIMESTAMP
		WHERE service_name = 'alphavantage' 
		  AND (last_reset_date < CURRENT_DATE 
		       OR (last_reset_date = CURRENT_DATE 
		           AND last_reset_hour < EXTRACT(HOUR FROM CURRENT_TIMESTAMP)))
	`
	
	result, err := s.db.Exec(query)
	if err != nil {
		log.Printf("Failed to reset rate limits: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Reset rate limits for %d services", rowsAffected)
	}
}

// GetStatus returns the current status of the data sync service
func (s *SchedulerService) GetStatus() DataSyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get total active stocks
	var totalStocks int
	s.db.QueryRow("SELECT COUNT(*) FROM stocks WHERE is_active = true").Scan(&totalStocks)
	
	// Get stocks processed today
	var processedToday int
	query := `
		SELECT COUNT(DISTINCT stock_id) 
		FROM daily_prices 
		WHERE DATE(created_at) = CURRENT_DATE
	`
	s.db.QueryRow(query).Scan(&processedToday)
	
	// Calculate next sync time (next hour)
	now := time.Now()
	nextSync := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	
	// Copy errors to avoid holding lock too long
	errors := make([]string, len(s.syncErrors))
	copy(errors, s.syncErrors)
	
	return DataSyncStatus{
		IsRunning:      s.isRunning,
		LastSync:       s.lastDataSync,
		NextSync:       nextSync,
		TotalStocks:    totalStocks,
		ProcessedToday: processedToday,
		Errors:         errors,
	}
}

// addError adds an error to the error list with timestamp
func (s *SchedulerService) addError(errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	timestampedError := time.Now().Format("2006-01-02 15:04:05") + ": " + errorMsg
	s.syncErrors = append(s.syncErrors, timestampedError)
	
	// Keep only last 20 errors
	if len(s.syncErrors) > 20 {
		s.syncErrors = s.syncErrors[1:]
	}
	
	log.Printf("Sync error: %s", errorMsg)
}

// TriggerManualSync triggers a manual data sync for a specific stock
func (s *SchedulerService) TriggerManualSync(symbol string) error {
	canMake, err := s.alphaVantageClient.CanMakeRequest()
	if err != nil {
		return err
	}
	
	if !canMake {
		return fmt.Errorf("rate limit exceeded, cannot perform manual sync")
	}
	
	log.Printf("Manual sync triggered for %s", symbol)
	
	data, err := s.alphaVantageClient.FetchDailyData(symbol)
	if err != nil {
		return err
	}
	
	err = s.alphaVantageClient.SaveHistoricalData(symbol, data)
	if err != nil {
		return err
	}
	
	// Invalidate all caches immediately when new data arrives (manual sync)
	if s.cache != nil {
		err = s.cache.InvalidateAll()
		if err != nil {
			log.Printf("Warning: Failed to invalidate cache after manual sync: %v", err)
		} else {
			log.Printf("🔄 Cache invalidated after manual sync for %s", symbol)
		}
	}
	
	err = s.updateStockSyncTime(symbol)
	if err != nil {
		log.Printf("Failed to update sync time for %s: %v", symbol, err)
	}
	
	s.mu.Lock()
	s.lastDataSync = time.Now()
	s.mu.Unlock()
	
	return nil
}