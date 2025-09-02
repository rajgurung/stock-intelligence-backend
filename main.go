package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"stock-intelligence-backend/internal/cache"
	"stock-intelligence-backend/internal/database"
	"stock-intelligence-backend/internal/handlers"
	"stock-intelligence-backend/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize database
	db, err := database.InitializeDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis cache
	redisURL := os.Getenv("REDIS_URL")
	redisCache, err := cache.NewRedisCache(redisURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Continuing without cache...")
		redisCache = nil
	} else {
		defer redisCache.Close()
	}
	
	// Initialize services
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	
	// Create Alpha Vantage client
	alphaVantageClient := services.NewAlphaVantageClient(apiKey, db)
	
	// Create scheduler service with cache for invalidation
	schedulerService := services.NewSchedulerService(db, alphaVantageClient, redisCache)
	
	// Start scheduler if API key is configured
	if apiKey != "" && apiKey != "your_api_key_here" {
		if err := schedulerService.Start(); err != nil {
			log.Printf("Failed to start scheduler: %v", err)
		} else {
			log.Println("Data synchronization scheduler started")
		}
	}
	
	// Initialize database stock service with Redis cache
	databaseStockService := services.NewDatabaseStockService(db, redisCache)
	
	// Initialize historical data sync service
	historicalDataSyncService := services.NewHistoricalDataSyncService(db, alphaVantageClient)
	
	// Initialize handlers
	databaseStockHandler := handlers.NewDatabaseStockHandler(databaseStockService)
	wsHandler := handlers.NewWebSocketHandler(services.NewHybridStockService(databaseStockService))
	systemHandler := handlers.NewSystemHandler(alphaVantageClient, schedulerService)
	syncHandler := handlers.NewHistoricalDataSyncHandler(historicalDataSyncService)

	// Initialize router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":           "ok",
			"service":          "stock-intelligence-backend",
			"websocket_clients": wsHandler.GetConnectedClients(),
		})
	})

	// WebSocket endpoint
	r.GET("/ws", wsHandler.HandleWebSocket)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Stock endpoints
		stocks := v1.Group("/stocks")
		{
			stocks.GET("", databaseStockHandler.GetAllStocks)
			stocks.GET("/:symbol", databaseStockHandler.GetStockBySymbol)
			stocks.GET("/:symbol/performance", databaseStockHandler.GetStockHistoricalPerformance)
			stocks.GET("/price-range", databaseStockHandler.GetStocksByPriceRange)
		}

		// Market data endpoints
		market := v1.Group("/market")
		{
			market.GET("/performance", databaseStockHandler.GetPerformanceData)
			market.GET("/overview", databaseStockHandler.GetMarketOverview)
			market.GET("/sectors", databaseStockHandler.GetSectors)
			market.GET("/data-source", databaseStockHandler.GetDataSourceInfo)
		}
		
		// System monitoring endpoints
		system := v1.Group("/system")
		{
			system.GET("/health", systemHandler.GetSystemHealth)
			system.GET("/api-status", systemHandler.GetAPIStatus)
			system.GET("/sync-status", systemHandler.GetDataSyncStatus)
			system.GET("/api-history", systemHandler.GetAPICallHistory)
			system.POST("/sync/:symbol", systemHandler.TriggerManualSync)
		}
		
		// Historical data sync endpoints
		sync := v1.Group("/sync")
		{
			sync.POST("/batch", syncHandler.TriggerBatchSync)
			sync.GET("/status", syncHandler.GetSyncStatus)
			sync.GET("/pending", syncHandler.GetPendingStocks)
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	log.Println("Database-only mode: Using database as primary data source")
	log.Printf("Stock data service ready with %d stocks", len(databaseStockService.GetAllStocks()))
	
	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		schedulerService.Stop()
		db.Close()
		os.Exit(0)
	}()
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}