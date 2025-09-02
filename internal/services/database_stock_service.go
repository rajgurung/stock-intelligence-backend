package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"stock-intelligence-backend/internal/cache"
	"stock-intelligence-backend/internal/models"
)

type DatabaseStockService struct {
	db    *sql.DB
	cache *cache.RedisCache
}

func NewDatabaseStockService(db *sql.DB, redisCache *cache.RedisCache) *DatabaseStockService {
	return &DatabaseStockService{
		db:    db,
		cache: redisCache,
	}
}

// GetAllStocks returns all stocks from the database with caching
func (d *DatabaseStockService) GetAllStocks() []models.Stock {
	// Try to get from cache first
	if d.cache != nil {
		var cachedStocks []models.Stock
		err := d.cache.GetStocksList(&cachedStocks)
		if err == nil && len(cachedStocks) > 0 {
			log.Printf("Loaded %d stocks from cache", len(cachedStocks))
			return cachedStocks
		}
	}

	// Cache miss - fetch from database
	stocks := d.fetchAllStocksFromDatabase()

	// Cache the results for 55 minutes (until next hourly update + safety margin)
	if d.cache != nil && len(stocks) > 0 {
		err := d.cache.SetStocksList(stocks, 55*time.Minute)
		if err != nil {
			log.Printf("Warning: Failed to cache stocks list: %v", err)
		}
	}

	return stocks
}

// fetchAllStocksFromDatabase performs the actual database query
func (d *DatabaseStockService) fetchAllStocksFromDatabase() []models.Stock {
	query := `
		SELECT s.id, s.symbol, s.company_name, s.sector, s.industry, s.market_cap, 
		       s.price_range, s.exchange, s.is_active, s.created_at, s.updated_at,
		       COALESCE(latest.close_price, 0) as current_price,
		       COALESCE(latest.close_price - previous.close_price, 0) as daily_change,
		       COALESCE(
		           CASE WHEN previous.close_price > 0 THEN
		               ((latest.close_price - previous.close_price) / previous.close_price * 100)
		           ELSE 0 END, 0
		       ) as change_percent,
		       COALESCE(latest.volume, 0) as volume,
		       COALESCE(latest.date, s.updated_at) as last_updated
		FROM stocks s
		LEFT JOIN LATERAL (
		    SELECT close_price, volume, date 
		    FROM daily_prices 
		    WHERE stock_id = s.id 
		    ORDER BY date DESC 
		    LIMIT 1
		) latest ON true
		LEFT JOIN LATERAL (
		    SELECT close_price 
		    FROM daily_prices 
		    WHERE stock_id = s.id AND date < latest.date
		    ORDER BY date DESC 
		    LIMIT 1
		) previous ON true
		WHERE s.is_active = true
		ORDER BY s.symbol
	`
	
	rows, err := d.db.Query(query)
	if err != nil {
		log.Printf("Error fetching stocks: %v", err)
		return []models.Stock{}
	}
	defer rows.Close()
	
	var stocks []models.Stock
	for rows.Next() {
		var stock models.Stock
		var currentPrice sql.NullFloat64
		var dailyChange sql.NullFloat64
		var changePercent sql.NullFloat64
		var volume sql.NullInt64
		var lastUpdated time.Time
		var priceRange sql.NullString
		
		err := rows.Scan(
			&stock.ID, &stock.Symbol, &stock.CompanyName, &stock.Sector, 
			&stock.Industry, &stock.MarketCap, &priceRange, &stock.Exchange,
			&stock.IsActive, &stock.CreatedAt, &stock.UpdatedAt,
			&currentPrice, &dailyChange, &changePercent, &volume, &lastUpdated,
		)
		if err != nil {
			log.Printf("Error scanning stock: %v", err)
			continue
		}
		
		// Set price range from database or use fallback
		if priceRange.Valid {
			stock.PriceRange = priceRange.String
		} else {
			stock.PriceRange = ""
		}
		
		// Set computed fields from database data only
		if currentPrice.Valid && currentPrice.Float64 > 0 {
			stock.CurrentPrice = currentPrice.Float64
			// Only set change values if they are valid (not null from database)
			if dailyChange.Valid {
				stock.DailyChange = dailyChange.Float64
			}
			if changePercent.Valid {
				stock.ChangePercent = changePercent.Float64
			}
			stock.Volume = volume.Int64
		} else {
			// Set default values for stocks without price data
			stock.CurrentPrice = 0.0
			stock.DailyChange = 0.0
			stock.ChangePercent = 0.0
			stock.Volume = 0
		}
		
		stock.LastUpdated = lastUpdated
		
		// Ensure price range is set
		if stock.PriceRange == "" {
			stock.PriceRange = stock.GetPriceRange()
		}
		
		stocks = append(stocks, stock)
	}
	
	log.Printf("Loaded %d stocks from database", len(stocks))
	return stocks
}

// GetAllStocksPaginated returns stocks with pagination support
func (d *DatabaseStockService) GetAllStocksPaginated(limit, offset int) ([]models.Stock, int) {
	// First get total count
	var totalCount int
	countQuery := `
		SELECT COUNT(DISTINCT s.id)
		FROM stocks s
		LEFT JOIN LATERAL (
		    SELECT close_price, volume, date 
		    FROM daily_prices 
		    WHERE stock_id = s.id 
		    ORDER BY date DESC 
		    LIMIT 1
		) latest ON true
		LEFT JOIN LATERAL (
		    SELECT close_price 
		    FROM daily_prices 
		    WHERE stock_id = s.id AND date < latest.date
		    ORDER BY date DESC 
		    LIMIT 1
		) previous ON true
		WHERE s.is_active = true
	`
	
	err := d.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		log.Printf("Error getting stock count: %v", err)
		return []models.Stock{}, 0
	}
	
	// Now get paginated results using the same query structure as GetAllStocks
	query := `
		SELECT s.id, s.symbol, s.company_name, s.sector, s.industry, s.market_cap, 
		       s.price_range, s.exchange, s.is_active, s.created_at, s.updated_at,
		       COALESCE(latest.close_price, 0) as current_price,
		       COALESCE(latest.close_price - previous.close_price, 0) as daily_change,
		       COALESCE(
		           CASE WHEN previous.close_price > 0 THEN
		               ((latest.close_price - previous.close_price) / previous.close_price * 100)
		           ELSE 0 END, 0
		       ) as change_percent,
		       COALESCE(latest.volume, 0) as volume,
		       COALESCE(latest.date, s.updated_at) as last_updated
		FROM stocks s
		LEFT JOIN LATERAL (
		    SELECT close_price, volume, date 
		    FROM daily_prices 
		    WHERE stock_id = s.id 
		    ORDER BY date DESC 
		    LIMIT 1
		) latest ON true
		LEFT JOIN LATERAL (
		    SELECT close_price 
		    FROM daily_prices 
		    WHERE stock_id = s.id AND date < latest.date
		    ORDER BY date DESC 
		    LIMIT 1
		) previous ON true
		WHERE s.is_active = true
		ORDER BY s.market_cap DESC, s.symbol
		LIMIT $1 OFFSET $2
	`
	
	rows, err := d.db.Query(query, limit, offset)
	if err != nil {
		log.Printf("Error fetching paginated stocks: %v", err)
		return []models.Stock{}, totalCount
	}
	defer rows.Close()
	
	var stocks []models.Stock
	for rows.Next() {
		var stock models.Stock
		var currentPrice sql.NullFloat64
		var dailyChange sql.NullFloat64
		var changePercent sql.NullFloat64
		var volume sql.NullInt64
		var lastUpdated time.Time
		var priceRange sql.NullString
		
		err := rows.Scan(
			&stock.ID, &stock.Symbol, &stock.CompanyName, &stock.Sector, 
			&stock.Industry, &stock.MarketCap, &priceRange, &stock.Exchange,
			&stock.IsActive, &stock.CreatedAt, &stock.UpdatedAt,
			&currentPrice, &dailyChange, &changePercent, &volume, &lastUpdated,
		)
		if err != nil {
			log.Printf("Error scanning stock: %v", err)
			continue
		}
		
		// Set price range from database or use fallback
		if priceRange.Valid {
			stock.PriceRange = priceRange.String
		} else {
			stock.PriceRange = ""
		}
		
		// Set computed fields from database data only
		if currentPrice.Valid && currentPrice.Float64 > 0 {
			stock.CurrentPrice = currentPrice.Float64
			// Only set change values if they are valid (not null from database)
			if dailyChange.Valid {
				stock.DailyChange = dailyChange.Float64
			}
			if changePercent.Valid {
				stock.ChangePercent = changePercent.Float64
			}
			stock.Volume = volume.Int64
		} else {
			// Set default values for stocks without price data
			stock.CurrentPrice = 0.0
			stock.DailyChange = 0.0
			stock.ChangePercent = 0.0
			stock.Volume = 0
		}
		
		stock.LastUpdated = lastUpdated
		
		// Ensure price range is set
		if stock.PriceRange == "" {
			stock.PriceRange = stock.GetPriceRange()
		}
		
		stocks = append(stocks, stock)
	}
	
	log.Printf("Loaded %d stocks from database (page %d, limit %d)", len(stocks), offset/limit+1, limit)
	return stocks, totalCount
}


// GetStockBySymbol returns a specific stock by symbol
func (d *DatabaseStockService) GetStockBySymbol(symbol string) (*models.Stock, error) {
	query := `
		SELECT s.id, s.symbol, s.company_name, s.sector, s.industry, s.market_cap,
		       s.price_range, s.exchange, s.is_active, s.created_at, s.updated_at
		FROM stocks s
		WHERE s.symbol = $1 AND s.is_active = true
	`
	
	var stock models.Stock
	var priceRange sql.NullString
	err := d.db.QueryRow(query, symbol).Scan(
		&stock.ID, &stock.Symbol, &stock.CompanyName, &stock.Sector,
		&stock.Industry, &stock.MarketCap, &priceRange, &stock.Exchange,
		&stock.IsActive, &stock.CreatedAt, &stock.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("stock not found: %s", symbol)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	// Set price range from database or use fallback
	if priceRange.Valid {
		stock.PriceRange = priceRange.String
	} else {
		stock.PriceRange = ""
	}
	
	// Get latest price data
	priceQuery := `
		SELECT close_price, volume, date,
		       close_price - LAG(close_price) OVER (ORDER BY date) as daily_change,
		       ((close_price - LAG(close_price) OVER (ORDER BY date)) / 
		        LAG(close_price) OVER (ORDER BY date) * 100) as change_percent
		FROM daily_prices 
		WHERE stock_id = $1 
		ORDER BY date DESC 
		LIMIT 1
	`
	
	var currentPrice, dailyChange, changePercent sql.NullFloat64
	var volume sql.NullInt64
	var lastUpdated time.Time
	
	err = d.db.QueryRow(priceQuery, stock.ID).Scan(
		&currentPrice, &volume, &lastUpdated, &dailyChange, &changePercent,
	)
	
	if err == nil && currentPrice.Valid {
		stock.CurrentPrice = currentPrice.Float64
		stock.DailyChange = dailyChange.Float64
		stock.ChangePercent = changePercent.Float64
		stock.Volume = volume.Int64
		stock.LastUpdated = lastUpdated
	} else {
		// Return error if no price data available - database-only mode
		return nil, fmt.Errorf("no price data available for stock: %s", symbol)
	}
	
	return &stock, nil
}

// GetStocksBySector returns stocks filtered by sector with caching
func (d *DatabaseStockService) GetStocksBySector(sector string) []models.Stock {
	// Try to get from cache first
	if d.cache != nil {
		var cachedStocks []models.Stock
		err := d.cache.GetSectorData(sector, &cachedStocks)
		if err == nil && len(cachedStocks) > 0 {
			log.Printf("Loaded %d stocks for sector '%s' from cache", len(cachedStocks), sector)
			return cachedStocks
		}
	}

	// Cache miss - filter from all stocks
	allStocks := d.GetAllStocks()
	var filtered []models.Stock
	
	for _, stock := range allStocks {
		if stock.Sector == sector {
			filtered = append(filtered, stock)
		}
	}

	// Cache the sector results for 55 minutes (until next hourly update + safety margin)
	if d.cache != nil && len(filtered) > 0 {
		err := d.cache.SetSectorData(sector, filtered, 55*time.Minute)
		if err != nil {
			log.Printf("Warning: Failed to cache sector data for '%s': %v", sector, err)
		}
	}
	
	return filtered
}

// GetStocksByPriceRange returns stocks filtered by price range
func (d *DatabaseStockService) GetStocksByPriceRange(priceRange string) []models.Stock {
	allStocks := d.GetAllStocks()
	var filtered []models.Stock
	
	for _, stock := range allStocks {
		if stock.PriceRange == priceRange {
			filtered = append(filtered, stock)
		}
	}
	
	return filtered
}

// GetDB returns the database connection for direct queries
func (d *DatabaseStockService) GetDB() *sql.DB {
	return d.db
}