package services

import (
	"log"

	"stock-intelligence-backend/internal/models"
)

// HybridStockService provides a simple interface for stock data
// Uses database service as the only data source
type HybridStockService struct {
	databaseService *DatabaseStockService
}

// NewHybridStockService creates a new hybrid stock service
func NewHybridStockService(databaseService *DatabaseStockService) *HybridStockService {
	service := &HybridStockService{
		databaseService: databaseService,
	}

	log.Println("Stock service initialized with database backend")
	return service
}

// GetAllStocks returns all stocks from database
func (h *HybridStockService) GetAllStocks() []models.Stock {
	return h.databaseService.GetAllStocks()
}

// refreshCache is no longer needed as we use database service directly
func (h *HybridStockService) refreshCache() {
	// Database service handles its own caching and refresh logic
	return
}

// GetStockBySymbol returns a specific stock by symbol
func (h *HybridStockService) GetStockBySymbol(symbol string) *models.Stock {
	stock, err := h.databaseService.GetStockBySymbol(symbol)
	if err != nil {
		return nil
	}
	return stock
}

// GetStocksByPriceRange filters stocks by price range
func (h *HybridStockService) GetStocksByPriceRange(priceRange string) []models.Stock {
	return h.databaseService.GetStocksByPriceRange(priceRange)
}

// GetStocksBySector filters stocks by sector
func (h *HybridStockService) GetStocksBySector(sector string) []models.Stock {
	return h.databaseService.GetStocksBySector(sector)
}

// GetPerformanceData returns categorized performance data
func (h *HybridStockService) GetPerformanceData() models.StockPerformance {
	stocks := h.GetAllStocks()

	var gainers, losers, mostActive []models.Stock

	// Separate stocks by performance
	for _, stock := range stocks {
		if stock.ChangePercent > 0 {
			gainers = append(gainers, stock)
		} else if stock.ChangePercent < 0 {
			losers = append(losers, stock)
		}
		mostActive = append(mostActive, stock)
	}

	// Sort gainers by change percent (descending)
	for i := 0; i < len(gainers)-1; i++ {
		for j := i + 1; j < len(gainers); j++ {
			if gainers[i].ChangePercent < gainers[j].ChangePercent {
				gainers[i], gainers[j] = gainers[j], gainers[i]
			}
		}
	}

	// Sort losers by change percent (ascending)
	for i := 0; i < len(losers)-1; i++ {
		for j := i + 1; j < len(losers); j++ {
			if losers[i].ChangePercent > losers[j].ChangePercent {
				losers[i], losers[j] = losers[j], losers[i]
			}
		}
	}

	// Sort by volume for most active
	for i := 0; i < len(mostActive)-1; i++ {
		for j := i + 1; j < len(mostActive); j++ {
			if mostActive[i].Volume < mostActive[j].Volume {
				mostActive[i], mostActive[j] = mostActive[j], mostActive[i]
			}
		}
	}

	// Limit to top 10 each
	if len(gainers) > 10 {
		gainers = gainers[:10]
	}
	if len(losers) > 10 {
		losers = losers[:10]
	}
	if len(mostActive) > 10 {
		mostActive = mostActive[:10]
	}

	return models.StockPerformance{
		TopGainers: gainers,
		TopLosers:  losers,
		MostActive: mostActive,
	}
}

// GetMarketOverview returns overall market statistics
func (h *HybridStockService) GetMarketOverview() models.MarketOverview {
	allStocks := h.GetAllStocks()

	var advancing, declining, unchanged int
	var totalChange float64

	for _, stock := range allStocks {
		if stock.ChangePercent > 0 {
			advancing++
		} else if stock.ChangePercent < 0 {
			declining++
		} else {
			unchanged++
		}
		totalChange += stock.ChangePercent
	}

	avgChange := 0.0
	if len(allStocks) > 0 {
		avgChange = totalChange / float64(len(allStocks))
	}

	return models.MarketOverview{
		TotalStocks:    len(allStocks),
		AdvancingCount: advancing,
		DecliningCount: declining,
		UnchangedCount: unchanged,
		AvgChange:      avgChange,
	}
}

// GetDataSource returns information about current data source
func (h *HybridStockService) GetDataSource() map[string]interface{} {
	totalStocks := len(h.GetAllStocks())

	return map[string]interface{}{
		"using_real_data":   true,
		"total_stocks":      totalStocks,
		"data_sources": map[string]int{
			"database": totalStocks,
		},
		"api_status": "Database-only mode - Alpha Vantage API data stored in database",
	}
}

// GetHistoricalPerformance returns historical performance data for a stock symbol
func (h *HybridStockService) GetHistoricalPerformance(symbol string, days int) *models.HistoricalPerformance {
	return nil // No historical data available - database-only mode uses real data through handlers
}


