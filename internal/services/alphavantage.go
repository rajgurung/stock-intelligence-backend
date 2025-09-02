package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"stock-intelligence-backend/internal/models"
)

type AlphaVantageClient struct {
	apiKey   string
	baseURL  string
	db       *sql.DB
	client   *http.Client
}

type AlphaVantageResponse struct {
	MetaData   MetaData                     `json:"Meta Data"`
	TimeSeries map[string]TimeSeriesEntry   `json:"Time Series (Daily)"`
}

type MetaData struct {
	Information   string `json:"1. Information"`
	Symbol        string `json:"2. Symbol"`
	LastRefreshed string `json:"3. Last Refreshed"`
	OutputSize    string `json:"4. Output Size"`
	TimeZone      string `json:"5. Time Zone"`
}

type TimeSeriesEntry struct {
	Open   string `json:"1. open"`
	High   string `json:"2. high"`
	Low    string `json:"3. low"`
	Close  string `json:"4. close"`
	Volume string `json:"5. volume"`
}

func NewAlphaVantageClient(apiKey string, db *sql.DB) *AlphaVantageClient {
	return &AlphaVantageClient{
		apiKey:  apiKey,
		baseURL: "https://www.alphavantage.co/query",
		db:      db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CanMakeRequest checks if we can make an API call based on rate limits
func (a *AlphaVantageClient) CanMakeRequest() (bool, error) {
	var rateLimit models.APIRateLimit
	
	query := `
		SELECT id, service_name, daily_limit, hourly_limit, current_daily_count, 
		       current_hourly_count, last_reset_date, last_reset_hour
		FROM api_rate_limits 
		WHERE service_name = 'alphavantage'
	`
	
	err := a.db.QueryRow(query).Scan(
		&rateLimit.ID, &rateLimit.ServiceName, &rateLimit.DailyLimit,
		&rateLimit.HourlyLimit, &rateLimit.CurrentDailyCount,
		&rateLimit.CurrentHourlyCount, &rateLimit.LastResetDate,
		&rateLimit.LastResetHour,
	)
	
	if err != nil {
		return false, fmt.Errorf("failed to get rate limit: %w", err)
	}
	
	return rateLimit.CanMakeRequest(), nil
}

// LogAPICall logs an API call to the database
func (a *AlphaVantageClient) LogAPICall(endpoint string, params map[string]string, 
	status int, responseBody, errorMsg string, processingTime time.Duration) error {
	
	paramsJSON, _ := json.Marshal(params)
	
	query := `
		INSERT INTO api_calls (service_name, endpoint, request_params, response_status, 
		                      response_body, error_message, processing_time_ms)
		VALUES ('alphavantage', $1, $2, $3, $4, $5, $6)
	`
	
	_, err := a.db.Exec(query, endpoint, paramsJSON, status, responseBody, errorMsg, 
		int(processingTime.Milliseconds()))
	
	if err != nil {
		log.Printf("Failed to log API call: %v", err)
		return err
	}
	
	// Update rate limit counters
	return a.updateRateLimit()
}

// updateRateLimit increments the rate limit counters
func (a *AlphaVantageClient) updateRateLimit() error {
	query := `
		UPDATE api_rate_limits 
		SET current_daily_count = current_daily_count + 1,
		    current_hourly_count = current_hourly_count + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE service_name = 'alphavantage'
	`
	
	_, err := a.db.Exec(query)
	return err
}

// FetchDailyData fetches daily time series data for a stock
func (a *AlphaVantageClient) FetchDailyData(symbol string) (*AlphaVantageResponse, error) {
	canMake, err := a.CanMakeRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	
	if !canMake {
		return nil, fmt.Errorf("rate limit exceeded for Alpha Vantage API")
	}
	
	params := map[string]string{
		"function":   "TIME_SERIES_DAILY",
		"symbol":     symbol,
		"outputsize": "full",
		"apikey":     a.apiKey,
	}
	
	start := time.Now()
	response, err := a.makeRequest(params)
	processingTime := time.Since(start)
	
	var responseBody string
	var status int
	var errorMsg string
	
	if err != nil {
		status = 0
		errorMsg = err.Error()
		log.Printf("Alpha Vantage API error for %s: %v", symbol, err)
	} else {
		status = 200
		responseBody = string(response)
	}
	
	// Log the API call
	logErr := a.LogAPICall("TIME_SERIES_DAILY", params, status, responseBody, errorMsg, processingTime)
	if logErr != nil {
		log.Printf("Failed to log API call: %v", logErr)
	}
	
	if err != nil {
		return nil, err
	}
	
	var avResponse AlphaVantageResponse
	if err := json.Unmarshal(response, &avResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Alpha Vantage response: %w", err)
	}
	
	// Check for API error responses
	if avResponse.TimeSeries == nil || len(avResponse.TimeSeries) == 0 {
		// Check if it's an error response
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(response, &errorResponse); err == nil {
			if errorMsg, exists := errorResponse["Error Message"]; exists {
				return nil, fmt.Errorf("Alpha Vantage API error: %v", errorMsg)
			}
			if note, exists := errorResponse["Note"]; exists {
				return nil, fmt.Errorf("Alpha Vantage API note: %v", note)
			}
		}
		return nil, fmt.Errorf("no time series data returned for symbol %s", symbol)
	}
	
	log.Printf("Successfully fetched %d days of data for %s", len(avResponse.TimeSeries), symbol)
	return &avResponse, nil
}

// makeRequest makes HTTP request to Alpha Vantage API
func (a *AlphaVantageClient) makeRequest(params map[string]string) ([]byte, error) {
	reqURL, err := url.Parse(a.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	
	query := reqURL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	reqURL.RawQuery = query.Encode()
	
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Stock-Intelligence-Backend/1.0")
	
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	return body, nil
}

// SaveHistoricalData saves Alpha Vantage data to database
func (a *AlphaVantageClient) SaveHistoricalData(symbol string, data *AlphaVantageResponse) error {
	// Get stock ID
	var stockID int
	err := a.db.QueryRow("SELECT id FROM stocks WHERE symbol = $1", symbol).Scan(&stockID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("stock with symbol %s not found", symbol)
		}
		return fmt.Errorf("failed to get stock ID: %w", err)
	}
	
	// Prepare insert statement with ON CONFLICT handling
	insertQuery := `
		INSERT INTO daily_prices (stock_id, date, open_price, high_price, low_price, 
		                         close_price, adjusted_close, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stock_id, date) 
		DO UPDATE SET 
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			adjusted_close = EXCLUDED.adjusted_close,
			volume = EXCLUDED.volume,
			created_at = CURRENT_TIMESTAMP
	`
	
	stmt, err := a.db.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()
	
	inserted := 0
	updated := 0
	
	for dateStr, entry := range data.TimeSeries {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Failed to parse date %s: %v", dateStr, err)
			continue
		}
		
		open, _ := strconv.ParseFloat(entry.Open, 64)
		high, _ := strconv.ParseFloat(entry.High, 64)
		low, _ := strconv.ParseFloat(entry.Low, 64)
		close, _ := strconv.ParseFloat(entry.Close, 64)
		adjustedClose := close // TIME_SERIES_DAILY doesn't have adjusted close, use regular close
		volume, _ := strconv.ParseInt(entry.Volume, 10, 64)
		
		result, err := stmt.Exec(stockID, date, open, high, low, close, adjustedClose, volume)
		if err != nil {
			log.Printf("Failed to insert data for %s on %s: %v", symbol, dateStr, err)
			continue
		}
		
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			inserted++
		} else {
			updated++
		}
	}
	
	log.Printf("Saved data for %s: %d inserted, %d updated", symbol, inserted, updated)
	return nil
}

// GetRateLimit returns current rate limit status
func (a *AlphaVantageClient) GetRateLimit() (*models.APIRateLimit, error) {
	var rateLimit models.APIRateLimit
	
	query := `
		SELECT id, service_name, daily_limit, hourly_limit, current_daily_count, 
		       current_hourly_count, last_reset_date, last_reset_hour, created_at, updated_at
		FROM api_rate_limits 
		WHERE service_name = 'alphavantage'
	`
	
	err := a.db.QueryRow(query).Scan(
		&rateLimit.ID, &rateLimit.ServiceName, &rateLimit.DailyLimit,
		&rateLimit.HourlyLimit, &rateLimit.CurrentDailyCount,
		&rateLimit.CurrentHourlyCount, &rateLimit.LastResetDate,
		&rateLimit.LastResetHour, &rateLimit.CreatedAt, &rateLimit.UpdatedAt,
	)
	
	return &rateLimit, err
}

// GetAPICallStats returns API call statistics
func (a *AlphaVantageClient) GetAPICallStats(days int) ([]models.APICallStats, error) {
	query := `
		SELECT service_name, endpoint, total_calls, successful_calls, failed_calls,
		       avg_processing_time_ms, last_call_at, call_date
		FROM api_call_stats 
		WHERE service_name = 'alphavantage' 
		  AND call_date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY call_date DESC, endpoint
	`
	
	rows, err := a.db.Query(fmt.Sprintf(query, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []models.APICallStats
	for rows.Next() {
		var stat models.APICallStats
		err := rows.Scan(&stat.ServiceName, &stat.Endpoint, &stat.TotalCalls,
			&stat.SuccessfulCalls, &stat.FailedCalls, &stat.AvgProcessingTimeMs,
			&stat.LastCallAt, &stat.CallDate)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	
	return stats, rows.Err()
}