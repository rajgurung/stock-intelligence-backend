package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// HistoricalDataSyncService manages bulk historical data synchronization
type HistoricalDataSyncService struct {
	db                    *sql.DB
	alphaVantageClient    *AlphaVantageClient
	sp500PriorityService  *SP500PriorityService
}

// NewHistoricalDataSyncService creates a new historical data sync service
func NewHistoricalDataSyncService(db *sql.DB, alphaVantageClient *AlphaVantageClient) *HistoricalDataSyncService {
	return &HistoricalDataSyncService{
		db:                   db,
		alphaVantageClient:   alphaVantageClient,
		sp500PriorityService: NewSP500PriorityService(db),
	}
}

// SyncBatch synchronizes historical data for multiple stocks in batch
func (h *HistoricalDataSyncService) SyncBatch(maxStocks int) (*SyncResult, error) {
	log.Printf("Starting batch sync for up to %d stocks", maxStocks)
	
	// Check remaining API calls
	canMake, err := h.alphaVantageClient.CanMakeRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to check API availability: %w", err)
	}
	if !canMake {
		return nil, fmt.Errorf("no API calls remaining for today")
	}
	
	// Get current rate limit info
	rateLimit, err := h.alphaVantageClient.GetRateLimit()
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit info: %w", err)
	}
	
	remainingCalls := rateLimit.DailyLimit - rateLimit.CurrentDailyCount
	if remainingCalls <= 0 {
		return nil, fmt.Errorf("no API calls remaining today (%d/%d used)", 
			rateLimit.CurrentDailyCount, rateLimit.DailyLimit)
	}
	
	// Limit to available calls
	if maxStocks > remainingCalls {
		maxStocks = remainingCalls
		log.Printf("Limiting sync to %d stocks due to API rate limits", maxStocks)
	}
	
	// Get pending stocks ordered by priority
	pendingStocks, err := h.sp500PriorityService.GetPendingStocksForSync(maxStocks)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending stocks: %w", err)
	}
	
	log.Printf("Found %d pending stocks for sync", len(pendingStocks))
	
	if len(pendingStocks) == 0 {
		return &SyncResult{
			TotalAttempted: 0,
			Successful:     0,
			Failed:         0,
			Message:        "No pending stocks found - all priority stocks already have data",
		}, nil
	}
	
	// Sync each stock
	result := &SyncResult{
		StartTime: time.Now(),
		Stocks:    make([]StockSyncResult, 0),
	}
	
	for i, stock := range pendingStocks {
		log.Printf("Syncing stock %d/%d: %s (priority %d)", i+1, len(pendingStocks), stock.Symbol, stock.Priority)
		
		stockResult := h.syncSingleStock(stock)
		result.Stocks = append(result.Stocks, stockResult)
		result.TotalAttempted++
		
		if stockResult.Success {
			result.Successful++
		} else {
			result.Failed++
		}
		
		// Add small delay between API calls to be respectful
		if i < len(pendingStocks)-1 {
			time.Sleep(1 * time.Second)
		}
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Message = fmt.Sprintf("Batch sync completed: %d successful, %d failed out of %d attempted", 
		result.Successful, result.Failed, result.TotalAttempted)
	
	log.Printf("Batch sync completed in %v: %d successful, %d failed", 
		result.Duration, result.Successful, result.Failed)
	
	return result, nil
}

// syncSingleStock synchronizes historical data for a single stock
func (h *HistoricalDataSyncService) syncSingleStock(stock SP500Stock) StockSyncResult {
	start := time.Now()
	
	result := StockSyncResult{
		Symbol:    stock.Symbol,
		Priority:  stock.Priority,
		StartTime: start,
	}
	
	// Fetch historical data from Alpha Vantage
	data, err := h.alphaVantageClient.FetchDailyData(stock.Symbol)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		result.EndTime = time.Now()
		log.Printf("Failed to fetch data for %s: %v", stock.Symbol, err)
		return result
	}
	
	// Save to database
	err = h.alphaVantageClient.SaveHistoricalData(stock.Symbol, data)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Failed to save data: %v", err)
		result.EndTime = time.Now()
		log.Printf("Failed to save data for %s: %v", stock.Symbol, err)
		return result
	}
	
	// Update stock metadata with S&P 500 info
	err = h.sp500PriorityService.UpdateStockWithPriority(stock.Symbol)
	if err != nil {
		log.Printf("Warning: Failed to update priority for %s: %v", stock.Symbol, err)
	}
	
	// Update data completeness status
	err = h.updateStockDataStatus(stock.Symbol)
	if err != nil {
		log.Printf("Warning: Failed to update data status for %s: %v", stock.Symbol, err)
	}
	
	result.Success = true
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(start)
	result.RecordsAdded = len(data.TimeSeries)
	
	log.Printf("Successfully synced %s: %d records in %v", stock.Symbol, result.RecordsAdded, result.Duration)
	
	return result
}

// updateStockDataStatus updates the data completeness status for a stock
func (h *HistoricalDataSyncService) updateStockDataStatus(symbol string) error {
	query := `
		UPDATE stocks 
		SET has_sufficient_data = (
			SELECT COUNT(*) >= 30 
			FROM daily_prices dp 
			WHERE dp.stock_id = stocks.id
		),
		data_quality_score = (
			SELECT LEAST(100, COUNT(*)::INTEGER) 
			FROM daily_prices dp 
			WHERE dp.stock_id = stocks.id
		),
		last_data_sync = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP
		WHERE symbol = $1
	`
	
	_, err := h.db.Exec(query, symbol)
	return err
}

// GetSyncStatus returns the current synchronization status
func (h *HistoricalDataSyncService) GetSyncStatus() (*SyncStatus, error) {
	// Get S&P 500 stocks and their data status
	sp500Stocks := h.sp500PriorityService.GetTop500SP500Stocks()
	
	status := &SyncStatus{
		TotalSP500Stocks:     len(sp500Stocks),
		StocksWithData:       0,
		StocksNeedingData:    0,
		TopPriorityPending:   make([]string, 0),
		LastSyncTime:         time.Time{},
	}
	
	// Check each stock's data status
	for _, stock := range sp500Stocks {
		query := `
			SELECT 
				COALESCE(s.has_sufficient_data, false) as has_data,
				COUNT(dp.date) as price_count,
				MAX(dp.date) as latest_date,
				s.last_data_sync
			FROM stocks s
			LEFT JOIN daily_prices dp ON s.id = dp.stock_id
			WHERE s.symbol = $1 AND s.is_active = true
			GROUP BY s.id, s.has_sufficient_data, s.last_data_sync
		`
		
		var hasData bool
		var priceCount int
		var latestDate sql.NullTime
		var lastSync sql.NullTime
		
		err := h.db.QueryRow(query, stock.Symbol).Scan(&hasData, &priceCount, &latestDate, &lastSync)
		if err != nil && err != sql.ErrNoRows {
			continue
		}
		
		if hasData && priceCount >= 30 {
			status.StocksWithData++
		} else {
			status.StocksNeedingData++
			// Add to top priority pending (first 10)
			if len(status.TopPriorityPending) < 10 {
				status.TopPriorityPending = append(status.TopPriorityPending, stock.Symbol)
			}
		}
		
		// Track latest sync time
		if lastSync.Valid && lastSync.Time.After(status.LastSyncTime) {
			status.LastSyncTime = lastSync.Time
		}
	}
	
	// Get API rate limit info
	rateLimit, err := h.alphaVantageClient.GetRateLimit()
	if err == nil {
		status.APICallsUsed = rateLimit.CurrentDailyCount
		status.APICallsRemaining = rateLimit.DailyLimit - rateLimit.CurrentDailyCount
		status.DailyAPILimit = rateLimit.DailyLimit
	}
	
	status.PercentComplete = float64(status.StocksWithData) / float64(status.TotalSP500Stocks) * 100
	
	return status, nil
}

// GetDB returns the database connection for use in handlers
func (h *HistoricalDataSyncService) GetDB() *sql.DB {
	return h.db
}

// SyncResult represents the result of a batch synchronization
type SyncResult struct {
	TotalAttempted int                 `json:"total_attempted"`
	Successful     int                 `json:"successful"`
	Failed         int                 `json:"failed"`
	StartTime      time.Time           `json:"start_time"`
	EndTime        time.Time           `json:"end_time"`
	Duration       time.Duration       `json:"duration"`
	Message        string              `json:"message"`
	Stocks         []StockSyncResult   `json:"stocks"`
}

// StockSyncResult represents the result of syncing a single stock
type StockSyncResult struct {
	Symbol       string        `json:"symbol"`
	Priority     int           `json:"priority"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`
	RecordsAdded int           `json:"records_added"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
}

// SyncStatus represents the overall synchronization status
type SyncStatus struct {
	TotalSP500Stocks     int       `json:"total_sp500_stocks"`
	StocksWithData       int       `json:"stocks_with_data"`
	StocksNeedingData    int       `json:"stocks_needing_data"`
	PercentComplete      float64   `json:"percent_complete"`
	TopPriorityPending   []string  `json:"top_priority_pending"`
	APICallsUsed         int       `json:"api_calls_used"`
	APICallsRemaining    int       `json:"api_calls_remaining"`
	DailyAPILimit        int       `json:"daily_api_limit"`
	LastSyncTime         time.Time `json:"last_sync_time"`
}