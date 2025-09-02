package services

import (
	"database/sql"
	"fmt"
	"log"
)

// SP500Stock represents a stock with priority information
type SP500Stock struct {
	Symbol      string
	CompanyName string
	Priority    int
	MarketCap   int64
	HasData     bool
}

// SP500PriorityService manages S&P 500 stock priorities for historical data fetching
type SP500PriorityService struct {
	db *sql.DB
}

// NewSP500PriorityService creates a new S&P 500 priority service
func NewSP500PriorityService(db *sql.DB) *SP500PriorityService {
	return &SP500PriorityService{
		db: db,
	}
}

// GetTop500SP500Stocks returns the top S&P 500 stocks ordered by market cap priority
func (s *SP500PriorityService) GetTop500SP500Stocks() []SP500Stock {
	// Top S&P 500 stocks by market cap (as of 2024)
	// Priority 1 = Highest priority (largest market cap)
	return []SP500Stock{
		// Mega Cap Tech (Priority 1-10)
		{Symbol: "AAPL", CompanyName: "Apple Inc.", Priority: 1, MarketCap: 3000000000000},
		{Symbol: "MSFT", CompanyName: "Microsoft Corporation", Priority: 2, MarketCap: 2800000000000},
		{Symbol: "GOOGL", CompanyName: "Alphabet Inc.", Priority: 3, MarketCap: 1700000000000},
		{Symbol: "AMZN", CompanyName: "Amazon.com Inc.", Priority: 4, MarketCap: 1500000000000},
		{Symbol: "NVDA", CompanyName: "NVIDIA Corporation", Priority: 5, MarketCap: 1400000000000},
		{Symbol: "TSLA", CompanyName: "Tesla Inc.", Priority: 6, MarketCap: 800000000000},
		{Symbol: "META", CompanyName: "Meta Platforms Inc.", Priority: 7, MarketCap: 750000000000},
		{Symbol: "NFLX", CompanyName: "Netflix Inc.", Priority: 8, MarketCap: 180000000000},
		{Symbol: "ADBE", CompanyName: "Adobe Inc.", Priority: 9, MarketCap: 220000000000},
		{Symbol: "CRM", CompanyName: "Salesforce Inc.", Priority: 10, MarketCap: 210000000000},
		
		// Large Cap Financial & Healthcare (Priority 11-25)
		{Symbol: "BRK.B", CompanyName: "Berkshire Hathaway Inc.", Priority: 11, MarketCap: 750000000000},
		{Symbol: "JPM", CompanyName: "JPMorgan Chase & Co.", Priority: 12, MarketCap: 450000000000},
		{Symbol: "JNJ", CompanyName: "Johnson & Johnson", Priority: 13, MarketCap: 420000000000},
		{Symbol: "V", CompanyName: "Visa Inc.", Priority: 14, MarketCap: 500000000000},
		{Symbol: "PG", CompanyName: "Procter & Gamble Co.", Priority: 15, MarketCap: 380000000000},
		{Symbol: "UNH", CompanyName: "UnitedHealth Group Inc.", Priority: 16, MarketCap: 480000000000},
		{Symbol: "HD", CompanyName: "Home Depot Inc.", Priority: 17, MarketCap: 350000000000},
		{Symbol: "MA", CompanyName: "Mastercard Inc.", Priority: 18, MarketCap: 380000000000},
		{Symbol: "BAC", CompanyName: "Bank of America Corp.", Priority: 19, MarketCap: 250000000000},
		{Symbol: "ABBV", CompanyName: "AbbVie Inc.", Priority: 20, MarketCap: 290000000000},
		{Symbol: "KO", CompanyName: "Coca-Cola Co.", Priority: 21, MarketCap: 260000000000},
		{Symbol: "WMT", CompanyName: "Walmart Inc.", Priority: 22, MarketCap: 520000000000},
		{Symbol: "PEP", CompanyName: "PepsiCo Inc.", Priority: 23, MarketCap: 240000000000},
		{Symbol: "COST", CompanyName: "Costco Wholesale Corp.", Priority: 24, MarketCap: 320000000000},
		{Symbol: "MRK", CompanyName: "Merck & Co. Inc.", Priority: 25, MarketCap: 280000000000},
		
		// Large Cap Industrials & Communication (Priority 26-50)
		{Symbol: "AVGO", CompanyName: "Broadcom Inc.", Priority: 26, MarketCap: 600000000000},
		{Symbol: "ORCL", CompanyName: "Oracle Corporation", Priority: 27, MarketCap: 320000000000},
		{Symbol: "TMO", CompanyName: "Thermo Fisher Scientific Inc.", Priority: 28, MarketCap: 210000000000},
		{Symbol: "ACN", CompanyName: "Accenture plc", Priority: 29, MarketCap: 220000000000},
		{Symbol: "CVX", CompanyName: "Chevron Corporation", Priority: 30, MarketCap: 290000000000},
		{Symbol: "LLY", CompanyName: "Eli Lilly and Co.", Priority: 31, MarketCap: 620000000000},
		{Symbol: "ABT", CompanyName: "Abbott Laboratories", Priority: 32, MarketCap: 180000000000},
		{Symbol: "QCOM", CompanyName: "QUALCOMM Inc.", Priority: 33, MarketCap: 180000000000},
		{Symbol: "TXN", CompanyName: "Texas Instruments Inc.", Priority: 34, MarketCap: 160000000000},
		{Symbol: "WFC", CompanyName: "Wells Fargo & Co.", Priority: 35, MarketCap: 180000000000},
		{Symbol: "NKE", CompanyName: "NIKE Inc.", Priority: 36, MarketCap: 160000000000},
		{Symbol: "INTC", CompanyName: "Intel Corporation", Priority: 37, MarketCap: 200000000000},
		{Symbol: "AMD", CompanyName: "Advanced Micro Devices Inc.", Priority: 38, MarketCap: 220000000000},
		{Symbol: "DHR", CompanyName: "Danaher Corporation", Priority: 39, MarketCap: 170000000000},
		{Symbol: "NEE", CompanyName: "NextEra Energy Inc.", Priority: 40, MarketCap: 150000000000},
		{Symbol: "CSCO", CompanyName: "Cisco Systems Inc.", Priority: 41, MarketCap: 200000000000},
		{Symbol: "PFE", CompanyName: "Pfizer Inc.", Priority: 42, MarketCap: 160000000000},
		{Symbol: "VZ", CompanyName: "Verizon Communications Inc.", Priority: 43, MarketCap: 170000000000},
		{Symbol: "CMCSA", CompanyName: "Comcast Corporation", Priority: 44, MarketCap: 160000000000},
		{Symbol: "DIS", CompanyName: "Walt Disney Co.", Priority: 45, MarketCap: 180000000000},
		{Symbol: "COP", CompanyName: "ConocoPhillips", Priority: 46, MarketCap: 140000000000},
		{Symbol: "MS", CompanyName: "Morgan Stanley", Priority: 47, MarketCap: 140000000000},
		{Symbol: "IBM", CompanyName: "International Business Machines Corp.", Priority: 48, MarketCap: 130000000000},
		{Symbol: "GS", CompanyName: "Goldman Sachs Group Inc.", Priority: 49, MarketCap: 130000000000},
		{Symbol: "CAT", CompanyName: "Caterpillar Inc.", Priority: 50, MarketCap: 140000000000},
	}
}

// GetPendingStocksForSync returns stocks that need historical data, ordered by priority
func (s *SP500PriorityService) GetPendingStocksForSync(limit int) ([]SP500Stock, error) {
	// First, get all stocks from database that need data
	query := `
		SELECT s.symbol, s.company_name, s.market_cap,
		       CASE WHEN COUNT(dp.date) >= 30 THEN true ELSE false END as has_data,
		       COUNT(dp.date) as price_count
		FROM stocks s
		LEFT JOIN daily_prices dp ON s.id = dp.stock_id
		WHERE s.is_active = true
		GROUP BY s.symbol, s.company_name, s.market_cap
		HAVING COUNT(dp.date) < 30
		ORDER BY s.market_cap DESC
		LIMIT $1
	`
	
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending stocks: %w", err)
	}
	defer rows.Close()
	
	var pendingStocks []SP500Stock
	sp500Map := make(map[string]SP500Stock)
	
	// Create a lookup map of S&P 500 stocks for priority assignment
	for _, stock := range s.GetTop500SP500Stocks() {
		sp500Map[stock.Symbol] = stock
	}
	
	for rows.Next() {
		var symbol, companyName string
		var marketCap int64
		var hasData bool
		var priceCount int
		
		err := rows.Scan(&symbol, &companyName, &marketCap, &hasData, &priceCount)
		if err != nil {
			log.Printf("Error scanning pending stock: %v", err)
			continue
		}
		
		// Create stock record
		stock := SP500Stock{
			Symbol:      symbol,
			CompanyName: companyName,
			MarketCap:   marketCap,
			HasData:     hasData,
		}
		
		// Assign priority if it's in our S&P 500 list, otherwise use market cap based priority
		if sp500Stock, exists := sp500Map[symbol]; exists {
			stock.Priority = sp500Stock.Priority
		} else {
			// Assign priority based on market cap for non-S&P 500 stocks
			stock.Priority = 500 + len(pendingStocks) // Lower priority
		}
		
		pendingStocks = append(pendingStocks, stock)
		
		log.Printf("Found pending stock: %s (priority %d, %d days of data)", 
			symbol, stock.Priority, priceCount)
	}
	
	return pendingStocks, rows.Err()
}

// GetStockPriority returns the priority of a given stock symbol
func (s *SP500PriorityService) GetStockPriority(symbol string) int {
	stocks := s.GetTop500SP500Stocks()
	for _, stock := range stocks {
		if stock.Symbol == symbol {
			return stock.Priority
		}
	}
	return 999 // Low priority if not in S&P 500
}

// UpdateStockWithPriority updates a stock record with S&P 500 priority information
func (s *SP500PriorityService) UpdateStockWithPriority(symbol string) error {
	stocks := s.GetTop500SP500Stocks()
	
	for _, stock := range stocks {
		if stock.Symbol == symbol {
			query := `
				UPDATE stocks 
				SET market_cap = $1, 
				    updated_at = CURRENT_TIMESTAMP
				WHERE symbol = $2
			`
			
			_, err := s.db.Exec(query, stock.MarketCap, symbol)
			if err != nil {
				return fmt.Errorf("failed to update stock priority for %s: %w", symbol, err)
			}
			
			log.Printf("Updated stock %s with priority %d and market cap %d", 
				symbol, stock.Priority, stock.MarketCap)
			return nil
		}
	}
	
	return fmt.Errorf("stock %s not found in S&P 500 list", symbol)
}