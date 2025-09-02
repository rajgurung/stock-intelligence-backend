package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"stock-intelligence-backend/internal/models"
	"stock-intelligence-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// DatabaseStockHandler handles stock-related HTTP requests using database
type DatabaseStockHandler struct {
	stockService *services.DatabaseStockService
}

// NewDatabaseStockHandler creates a new database stock handler
func NewDatabaseStockHandler(stockService *services.DatabaseStockService) *DatabaseStockHandler {
	return &DatabaseStockHandler{
		stockService: stockService,
	}
}

// GetAllStocks returns all stocks from database with pagination support
func (h *DatabaseStockHandler) GetAllStocks(c *gin.Context) {
	// Query parameters for filtering and pagination
	sector := c.Query("sector")
	priceRange := c.Query("price_range")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	
	// Parse pagination parameters
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50 // Default page size
	}
	if limit > 200 {
		limit = 200 // Maximum page size
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}
	
	var stocks []models.Stock
	var totalCount int
	
	// Apply filters
	if sector != "" {
		stocks = h.stockService.GetStocksBySector(sector)
		totalCount = len(stocks)
		// Apply pagination to filtered results
		end := offset + limit
		if offset >= len(stocks) {
			stocks = []models.Stock{}
		} else {
			if end > len(stocks) {
				end = len(stocks)
			}
			stocks = stocks[offset:end]
		}
	} else if priceRange != "" {
		stocks = h.stockService.GetStocksByPriceRange(priceRange)
		totalCount = len(stocks)
		// Apply pagination to filtered results
		end := offset + limit
		if offset >= len(stocks) {
			stocks = []models.Stock{}
		} else {
			if end > len(stocks) {
				end = len(stocks)
			}
			stocks = stocks[offset:end]
		}
	} else {
		// Use new paginated method
		stocks, totalCount = h.stockService.GetAllStocksPaginated(limit, offset)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"data":        stocks,
		"count":       len(stocks),
		"total":       totalCount,
		"offset":      offset,
		"limit":       limit,
		"has_more":    offset+len(stocks) < totalCount,
	})
}

// GetStockBySymbol returns a specific stock by symbol
func (h *DatabaseStockHandler) GetStockBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Symbol parameter is required",
		})
		return
	}
	
	stock, err := h.stockService.GetStockBySymbol(symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Stock not found",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stock,
	})
}

// GetStocksByPriceRange returns stocks filtered by price range
func (h *DatabaseStockHandler) GetStocksByPriceRange(c *gin.Context) {
	priceRange := c.Query("range")
	if priceRange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Price range parameter is required",
		})
		return
	}
	
	stocks := h.stockService.GetStocksByPriceRange(priceRange)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stocks,
		"count":   len(stocks),
	})
}

// GetSectors returns all unique sectors
func (h *DatabaseStockHandler) GetSectors(c *gin.Context) {
	stocks := h.stockService.GetAllStocks()
	sectorMap := make(map[string]int)
	
	for _, stock := range stocks {
		if stock.Sector != "" {
			sectorMap[stock.Sector]++
		}
	}
	
	type SectorInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	
	var sectors []SectorInfo
	for sector, count := range sectorMap {
		sectors = append(sectors, SectorInfo{
			Name:  sector,
			Count: count,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sectors,
		"count":   len(sectors),
	})
}

// GetMarketOverview returns market overview statistics
func (h *DatabaseStockHandler) GetMarketOverview(c *gin.Context) {
	stocks := h.stockService.GetAllStocks()
	
	totalStocks := len(stocks)
	advancing := 0
	declining := 0
	unchanged := 0
	totalChange := 0.0
	
	for _, stock := range stocks {
		if stock.ChangePercent > 0.01 {
			advancing++
		} else if stock.ChangePercent < -0.01 {
			declining++
		} else {
			unchanged++
		}
		totalChange += stock.ChangePercent
	}
	
	avgChange := 0.0
	if totalStocks > 0 {
		avgChange = totalChange / float64(totalStocks)
	}
	
	overview := map[string]interface{}{
		"total_stocks":    totalStocks,
		"advancing_count": advancing,
		"declining_count": declining,
		"unchanged_count": unchanged,
		"avg_change":      avgChange,
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// GetPerformanceData returns performance categories
func (h *DatabaseStockHandler) GetPerformanceData(c *gin.Context) {
	stocks := h.stockService.GetAllStocks()
	
	if len(stocks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"top_gainers": []models.Stock{},
				"top_losers":  []models.Stock{},
				"most_active": []models.Stock{},
			},
		})
		return
	}
	
	// Sort for top gainers (highest change percent)
	topGainers := make([]models.Stock, len(stocks))
	copy(topGainers, stocks)
	for i := 0; i < len(topGainers)-1; i++ {
		for j := i + 1; j < len(topGainers); j++ {
			if topGainers[j].ChangePercent > topGainers[i].ChangePercent {
				topGainers[i], topGainers[j] = topGainers[j], topGainers[i]
			}
		}
	}
	if len(topGainers) > 10 {
		topGainers = topGainers[:10]
	}
	
	// Sort for top losers (lowest change percent)
	topLosers := make([]models.Stock, len(stocks))
	copy(topLosers, stocks)
	for i := 0; i < len(topLosers)-1; i++ {
		for j := i + 1; j < len(topLosers); j++ {
			if topLosers[j].ChangePercent < topLosers[i].ChangePercent {
				topLosers[i], topLosers[j] = topLosers[j], topLosers[i]
			}
		}
	}
	if len(topLosers) > 10 {
		topLosers = topLosers[:10]
	}
	
	// Sort for most active (highest volume)
	mostActive := make([]models.Stock, len(stocks))
	copy(mostActive, stocks)
	for i := 0; i < len(mostActive)-1; i++ {
		for j := i + 1; j < len(mostActive); j++ {
			if mostActive[j].Volume > mostActive[i].Volume {
				mostActive[i], mostActive[j] = mostActive[j], mostActive[i]
			}
		}
	}
	if len(mostActive) > 10 {
		mostActive = mostActive[:10]
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"top_gainers": topGainers,
			"top_losers":  topLosers,
			"most_active": mostActive,
		},
	})
}

// GetDataSourceInfo returns information about data sources
func (h *DatabaseStockHandler) GetDataSourceInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"primary_source":   "Local Database",
			"fallback_source":  "Generated Data",
			"last_updated":     "Real-time",
			"total_stocks":     len(h.stockService.GetAllStocks()),
			"data_freshness":   "Live",
			"api_integration": []string{"Alpha Vantage (Historical)", "Local Generation (Real-time)"},
		},
	})
}

// GetStockHistoricalPerformance returns historical performance data for a specific stock
func (h *DatabaseStockHandler) GetStockHistoricalPerformance(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Symbol parameter is required",
		})
		return
	}
	
	// Get days parameter (default to 30 for mini charts)
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365 // Maximum 1 year
	}
	
	// Query recent daily prices from database
	query := `
		SELECT dp.date, dp.close_price, dp.volume
		FROM daily_prices dp
		JOIN stocks s ON dp.stock_id = s.id
		WHERE s.symbol = $1
		ORDER BY dp.date DESC
		LIMIT $2
	`
	
	rows, err := h.stockService.GetDB().Query(query, symbol, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch historical data",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()
	
	type DataPoint struct {
		Date   string  `json:"date"`
		Price  float64 `json:"price"`
		Volume int64   `json:"volume"`
	}
	
	var dataPoints []DataPoint
	for rows.Next() {
		var date time.Time
		var price float64
		var volume int64
		
		err := rows.Scan(&date, &price, &volume)
		if err != nil {
			continue // Skip invalid rows
		}
		
		dataPoints = append(dataPoints, DataPoint{
			Date:   date.Format("2006-01-02"),
			Price:  price,
			Volume: volume,
		})
	}
	
	// Reverse to get chronological order (oldest first)
	for i := 0; i < len(dataPoints)/2; i++ {
		j := len(dataPoints) - 1 - i
		dataPoints[i], dataPoints[j] = dataPoints[j], dataPoints[i]
	}
	
	// Calculate performance metrics if we have data
	totalReturn := 0.0
	if len(dataPoints) > 1 {
		startPrice := dataPoints[0].Price
		endPrice := dataPoints[len(dataPoints)-1].Price
		if startPrice > 0 {
			totalReturn = ((endPrice - startPrice) / startPrice) * 100
		}
	}
	
	performance := map[string]interface{}{
		"symbol":      symbol,
		"timeframe":   fmt.Sprintf("%dD", days),
		"data_points": dataPoints,
		"count":       len(dataPoints),
		"performance_metrics": gin.H{
			"total_return": totalReturn,
			"data_quality": "real", // Indicate this is real data
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    performance,
	})
}