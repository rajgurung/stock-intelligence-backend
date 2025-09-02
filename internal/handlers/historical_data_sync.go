package handlers

import (
	"net/http"
	"strconv"

	"stock-intelligence-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// HistoricalDataSyncHandler handles historical data synchronization endpoints
type HistoricalDataSyncHandler struct {
	syncService *services.HistoricalDataSyncService
}

// NewHistoricalDataSyncHandler creates a new historical data sync handler
func NewHistoricalDataSyncHandler(syncService *services.HistoricalDataSyncService) *HistoricalDataSyncHandler {
	return &HistoricalDataSyncHandler{
		syncService: syncService,
	}
}

// TriggerBatchSync triggers a batch synchronization of historical data
func (h *HistoricalDataSyncHandler) TriggerBatchSync(c *gin.Context) {
	// Get limit from query parameter (default 24)
	limitStr := c.DefaultQuery("limit", "24")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid limit parameter",
		})
		return
	}

	// Maximum safety limit
	if limit > 25 {
		limit = 25
	}

	// Trigger the batch sync
	result, err := h.syncService.SyncBatch(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": result.Message,
	})
}

// GetSyncStatus returns the current synchronization status
func (h *HistoricalDataSyncHandler) GetSyncStatus(c *gin.Context) {
	status, err := h.syncService.GetSyncStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// GetPendingStocks returns stocks that need historical data sync
func (h *HistoricalDataSyncHandler) GetPendingStocks(c *gin.Context) {
	// Get limit from query parameter (default 25)
	limitStr := c.DefaultQuery("limit", "25")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 25
	}

	// Create SP500 service instance
	sp500Service := services.NewSP500PriorityService(h.syncService.GetDB())
	
	pendingStocks, err := sp500Service.GetPendingStocksForSync(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pendingStocks,
		"count":   len(pendingStocks),
	})
}