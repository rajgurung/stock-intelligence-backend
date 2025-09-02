package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"stock-intelligence-backend/internal/models"
	"stock-intelligence-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, check the origin properly
		return true
	},
	// Add connection limits and timeouts
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// WebSocketHandler handles WebSocket connections for real-time data
type WebSocketHandler struct {
	stockService *services.HybridStockService
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast    chan []byte
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(stockService *services.HybridStockService) *WebSocketHandler {
	handler := &WebSocketHandler{
		stockService: stockService,
		clients:      make(map[*websocket.Conn]bool),
		broadcast:    make(chan []byte),
	}

	// Start the broadcast goroutine
	go handler.handleBroadcast()
	
	// Start the price update goroutine
	go handler.broadcastPriceUpdates()

	return handler
}

const maxConnections = 3 // Reasonable limit for a single user session

// HandleWebSocket handles WebSocket upgrade and connection
func (wsh *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Check connection limit first
	wsh.clientsMutex.RLock()
	currentConnections := len(wsh.clients)
	wsh.clientsMutex.RUnlock()
	
	if currentConnections >= maxConnections {
		log.Printf("WebSocket connection limit reached (%d/%d). Rejecting new connection from %s", 
			currentConnections, maxConnections, c.ClientIP())
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Too many connections",
			"limit": maxConnections,
			"current": currentConnections,
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	wsh.clientsMutex.Lock()
	wsh.clients[conn] = true
	clientCount := len(wsh.clients)
	wsh.clientsMutex.Unlock()

	log.Printf("WebSocket client connected. Total clients: %d/%d", clientCount, maxConnections)

	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	// Set up ping/pong handlers for connection health
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial data
	wsh.sendInitialData(conn)

	// Start ping ticker for this connection
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Handle incoming messages and pings
	go func() {
		for {
			select {
			case <-pingTicker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("WebSocket ping error: %v", err)
					return
				}
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket unexpected close error: %v", err)
			} else {
				log.Printf("WebSocket connection closed: %v", err)
			}
			break
		}
		// Reset read deadline on successful message
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	// Unregister client
	wsh.clientsMutex.Lock()
	delete(wsh.clients, conn)
	clientCount = len(wsh.clients)
	wsh.clientsMutex.Unlock()

	log.Printf("WebSocket client disconnected. Total clients: %d/%d", clientCount, maxConnections)
}

// sendInitialData sends initial stock data to a newly connected client
func (wsh *WebSocketHandler) sendInitialData(conn *websocket.Conn) {
	stocks := wsh.stockService.GetAllStocks()
	performance := wsh.stockService.GetPerformanceData()
	overview := wsh.stockService.GetMarketOverview()

	initialData := map[string]interface{}{
		"type": "initial",
		"data": map[string]interface{}{
			"stocks":      stocks,
			"performance": performance,
			"overview":    overview,
			"timestamp":   time.Now().Unix(),
		},
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(initialData); err != nil {
		log.Printf("Error sending initial data: %v", err)
	}
}

// handleBroadcast handles broadcasting messages to all clients
func (wsh *WebSocketHandler) handleBroadcast() {
	for {
		message := <-wsh.broadcast
		
		wsh.clientsMutex.Lock()
		var clientsToRemove []*websocket.Conn
		for client := range wsh.clients {
			client.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				clientsToRemove = append(clientsToRemove, client)
			}
		}
		
		// Remove failed clients after iteration
		for _, client := range clientsToRemove {
			delete(wsh.clients, client)
		}
		wsh.clientsMutex.Unlock()
	}
}

// broadcastPriceUpdates simulates real-time price updates
func (wsh *WebSocketHandler) broadcastPriceUpdates() {
	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get updated stock data
			stocks := wsh.stockService.GetAllStocks()
			
			// Simulate price changes for demo purposes
			updatedStocks := wsh.simulatepriceChanges(stocks)
			
			// Create update message
			updateMessage := map[string]interface{}{
				"type": "price_update",
				"data": map[string]interface{}{
					"stocks":    updatedStocks,
					"timestamp": time.Now().Unix(),
				},
			}

			// Broadcast to all connected clients
			wsh.broadcastToClients(updateMessage)
		}
	}
}

// simulatepriceChanges adds small random changes to stock prices for demo
func (wsh *WebSocketHandler) simulatepriceChanges(stocks []models.Stock) []models.Stock {
	// For demo purposes, we'll make small random changes to prices
	// In production, this would come from real market data feeds
	
	updatedStocks := make([]models.Stock, len(stocks))
	copy(updatedStocks, stocks)
	
	for i := range updatedStocks {
		// Random price change between -0.5% and +0.5%
		changePercent := (float64(time.Now().Unix()%1000) - 500) / 100000 // Simple pseudo-random
		priceChange := updatedStocks[i].CurrentPrice * changePercent
		
		updatedStocks[i].CurrentPrice += priceChange
		updatedStocks[i].DailyChange += priceChange
		updatedStocks[i].ChangePercent += changePercent
		updatedStocks[i].LastUpdated = time.Now()
	}
	
	return updatedStocks
}

// broadcastToClients sends a message to all connected WebSocket clients
func (wsh *WebSocketHandler) broadcastToClients(message interface{}) {
	wsh.clientsMutex.Lock()
	defer wsh.clientsMutex.Unlock()
	
	if len(wsh.clients) == 0 {
		return // No clients to broadcast to
	}

	var clientsToRemove []*websocket.Conn
	for client := range wsh.clients {
		client.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := client.WriteJSON(message); err != nil {
			log.Printf("WebSocket broadcast error: %v", err)
			client.Close()
			clientsToRemove = append(clientsToRemove, client)
		}
	}
	
	// Remove failed clients after iteration
	for _, client := range clientsToRemove {
		delete(wsh.clients, client)
	}
}

// GetConnectedClients returns the number of connected WebSocket clients
func (wsh *WebSocketHandler) GetConnectedClients() int {
	wsh.clientsMutex.RLock()
	defer wsh.clientsMutex.RUnlock()
	return len(wsh.clients)
}