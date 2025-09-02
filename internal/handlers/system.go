package handlers

import (
	"net/http"
	"strconv"
	"time"

	"stock-intelligence-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	alphaVantageClient *services.AlphaVantageClient
	schedulerService   *services.SchedulerService
}

func NewSystemHandler(alphaVantageClient *services.AlphaVantageClient, schedulerService *services.SchedulerService) *SystemHandler {
	return &SystemHandler{
		alphaVantageClient: alphaVantageClient,
		schedulerService:   schedulerService,
	}
}

// GetAPIStatus returns the current Alpha Vantage API status and rate limits
func (h *SystemHandler) GetAPIStatus(c *gin.Context) {
	rateLimit, err := h.alphaVantageClient.GetRateLimit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get API rate limit status",
			"details": err.Error(),
		})
		return
	}

	// Get API call stats for last 7 days
	stats, err := h.alphaVantageClient.GetAPICallStats(7)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get API call statistics",
			"details": err.Error(),
		})
		return
	}

	response := gin.H{
		"service": "alphavantage",
		"status":  "active",
		"rate_limit": gin.H{
			"daily_limit":          rateLimit.DailyLimit,
			"daily_used":           rateLimit.CurrentDailyCount,
			"daily_remaining":      rateLimit.RemainingDaily(),
			"hourly_limit":         rateLimit.HourlyLimit,
			"hourly_used":          rateLimit.CurrentHourlyCount,
			"hourly_remaining":     rateLimit.RemainingHourly(),
			"can_make_request":     rateLimit.CanMakeRequest(),
			"last_reset_date":      rateLimit.LastResetDate.Format("2006-01-02"),
			"last_reset_hour":      rateLimit.LastResetHour,
		},
		"statistics": stats,
		"updated_at": time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetDataSyncStatus returns the status of the background data synchronization
func (h *SystemHandler) GetDataSyncStatus(c *gin.Context) {
	status := h.schedulerService.GetStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"sync_status": status,
		"updated_at":  time.Now(),
	})
}

// TriggerManualSync triggers a manual sync for a specific stock
func (h *SystemHandler) TriggerManualSync(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Stock symbol is required",
		})
		return
	}

	err := h.schedulerService.TriggerManualSync(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to trigger manual sync",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Manual sync triggered successfully",
		"symbol":  symbol,
		"timestamp": time.Now(),
	})
}

// GetSystemHealth returns overall system health status
func (h *SystemHandler) GetSystemHealth(c *gin.Context) {
	// Get data sync status
	syncStatus := h.schedulerService.GetStatus()
	
	// Get API rate limit status
	rateLimit, err := h.alphaVantageClient.GetRateLimit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system health",
			"details": err.Error(),
		})
		return
	}

	// Determine overall health
	health := "healthy"
	if !syncStatus.IsRunning {
		health = "degraded"
	}
	if !rateLimit.CanMakeRequest() && syncStatus.ProcessedToday == 0 {
		health = "unhealthy"
	}

	response := gin.H{
		"status": health,
		"components": gin.H{
			"database": gin.H{
				"status": "healthy", // We assume DB is healthy if we can query it
			},
			"scheduler": gin.H{
				"status": map[bool]string{true: "healthy", false: "unhealthy"}[syncStatus.IsRunning],
				"details": gin.H{
					"last_sync":        syncStatus.LastSync,
					"next_sync":        syncStatus.NextSync,
					"processed_today":  syncStatus.ProcessedToday,
					"total_stocks":     syncStatus.TotalStocks,
					"recent_errors":    len(syncStatus.Errors),
				},
			},
			"api": gin.H{
				"status": map[bool]string{true: "healthy", false: "rate_limited"}[rateLimit.CanMakeRequest()],
				"details": gin.H{
					"daily_remaining":  rateLimit.RemainingDaily(),
					"hourly_remaining": rateLimit.RemainingHourly(),
				},
			},
		},
		"timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetAPICallHistory returns detailed API call history
func (h *SystemHandler) GetAPICallHistory(c *gin.Context) {
	// Get days parameter from query string, default to 7
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 30 {
		days = 7
	}

	stats, err := h.alphaVantageClient.GetAPICallStats(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get API call history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": stats,
		"period_days": days,
		"updated_at": time.Now(),
	})
}