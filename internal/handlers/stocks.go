package handlers

import (
	"net/http"
	"strconv"

	"stock-intelligence-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// StockHandler handles stock-related HTTP requests
type StockHandler struct {
	stockService *services.HybridStockService
}

// NewStockHandler creates a new stock handler
func NewStockHandler(stockService *services.HybridStockService) *StockHandler {
	return &StockHandler{
		stockService: stockService,
	}
}

// GetAllStocks returns all stocks
func (h *StockHandler) GetAllStocks(c *gin.Context) {
	// Query parameters for filtering
	sector := c.Query("sector")
	priceRange := c.Query("price_range")
	limit := c.Query("limit")
	
	stocks := h.stockService.GetAllStocks()
	
	// Apply filters
	if sector != "" {
		stocks = h.stockService.GetStocksBySector(sector)
	}
	
	if priceRange != "" {
		stocks = h.stockService.GetStocksByPriceRange(priceRange)
	}
	
	// Apply limit
	if limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 && limitInt < len(stocks) {
			stocks = stocks[:limitInt]
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stocks,
		"count":   len(stocks),
	})
}

// GetStockBySymbol returns a specific stock by symbol
func (h *StockHandler) GetStockBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	
	stock := h.stockService.GetStockBySymbol(symbol)
	if stock == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Stock not found",
			"symbol":  symbol,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stock,
	})
}

// GetPerformanceData returns categorized performance data
func (h *StockHandler) GetPerformanceData(c *gin.Context) {
	performance := h.stockService.GetPerformanceData()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    performance,
	})
}

// GetMarketOverview returns overall market statistics
func (h *StockHandler) GetMarketOverview(c *gin.Context) {
	overview := h.stockService.GetMarketOverview()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// GetStocksByPriceRange returns stocks filtered by price range
func (h *StockHandler) GetStocksByPriceRange(c *gin.Context) {
	priceRange := c.Query("range")
	if priceRange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "price range parameter is required",
		})
		return
	}
	
	stocks := h.stockService.GetStocksByPriceRange(priceRange)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stocks,
		"count":   len(stocks),
		"filter":  priceRange,
	})
}

// GetSectors returns available sectors
func (h *StockHandler) GetSectors(c *gin.Context) {
	stocks := h.stockService.GetAllStocks()
	sectorMap := make(map[string]int)
	
	for _, stock := range stocks {
		sectorMap[stock.Sector]++
	}
	
	sectors := make([]gin.H, 0, len(sectorMap))
	for sector, count := range sectorMap {
		sectors = append(sectors, gin.H{
			"sector": sector,
			"count":  count,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sectors,
	})
}

// GetDataSourceInfo returns information about current data sources
func (h *StockHandler) GetDataSourceInfo(c *gin.Context) {
	dataSourceInfo := h.stockService.GetDataSource()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dataSourceInfo,
	})
}

// GetStockHistoricalPerformance returns historical performance data for a specific stock
func (h *StockHandler) GetStockHistoricalPerformance(c *gin.Context) {
	symbol := c.Param("symbol")
	
	performance := h.stockService.GetHistoricalPerformance(symbol)
	if performance == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Historical performance data not found",
			"symbol":  symbol,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    performance,
	})
}