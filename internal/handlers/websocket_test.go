package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stock-intelligence-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock HybridStockService for testing
type MockHybridStockService struct {
	mock.Mock
}

func (m *MockHybridStockService) GetAllStocks() []models.Stock {
	args := m.Called()
	return args.Get(0).([]models.Stock)
}

func (m *MockHybridStockService) GetPerformanceData() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockHybridStockService) GetMarketOverview() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockHybridStockService) GetStockBySymbol(symbol string) (*models.Stock, error) {
	args := m.Called(symbol)
	return args.Get(0).(*models.Stock), args.Error(1)
}

func TestNewWebSocketHandler(t *testing.T) {
	mockService := &MockHybridStockService{}
	
	handler := NewWebSocketHandler(mockService)
	
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.clients)
	assert.NotNil(t, handler.broadcast)
	assert.Equal(t, mockService, handler.stockService)
	
	// Give goroutines a moment to start
	time.Sleep(10 * time.Millisecond)
}

func TestWebSocketHandler_GetConnectedClients(t *testing.T) {
	mockService := &MockHybridStockService{}
	handler := NewWebSocketHandler(mockService)
	
	// Initially no clients
	count := handler.GetConnectedClients()
	assert.Equal(t, 0, count)
}

func TestWebSocketHandler_HandleWebSocket_ConnectionLimit(t *testing.T) {
	mockService := &MockHybridStockService{}
	handler := NewWebSocketHandler(mockService)
	
	// Fill up to the connection limit
	for i := 0; i < maxConnections; i++ {
		conn := &websocket.Conn{}
		handler.clientsMutex.Lock()
		handler.clients[conn] = true
		handler.clientsMutex.Unlock()
	}
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)
	
	// Create request with WebSocket upgrade headers
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should reject connection due to limit
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Too many connections", response["error"])
	assert.Equal(t, float64(maxConnections), response["limit"])
}

func TestWebSocketHandler_SimulatePriceChanges(t *testing.T) {
	mockService := &MockHybridStockService{}
	handler := NewWebSocketHandler(mockService)
	
	originalStocks := []models.Stock{
		{
			ID:           1,
			Symbol:       "AAPL",
			CurrentPrice: 150.0,
			DailyChange:  2.5,
			ChangePercent: 1.69,
		},
		{
			ID:           2,
			Symbol:       "MSFT",
			CurrentPrice: 380.0,
			DailyChange:  -1.2,
			ChangePercent: -0.31,
		},
	}
	
	updatedStocks := handler.simulatepriceChanges(originalStocks)
	
	assert.Len(t, updatedStocks, 2)
	assert.Equal(t, "AAPL", updatedStocks[0].Symbol)
	assert.Equal(t, "MSFT", updatedStocks[1].Symbol)
	
	// Prices should be slightly different due to simulation
	// (exact values depend on time-based pseudo-random generation)
	assert.NotEqual(t, originalStocks[0].CurrentPrice, updatedStocks[0].CurrentPrice)
	assert.NotEqual(t, originalStocks[1].CurrentPrice, updatedStocks[1].CurrentPrice)
}

func TestWebSocketHandler_BroadcastToClients_NoClients(t *testing.T) {
	mockService := &MockHybridStockService{}
	handler := NewWebSocketHandler(mockService)
	
	message := map[string]interface{}{
		"type": "test",
		"data": "test message",
	}
	
	// Should not panic when no clients
	handler.broadcastToClients(message)
	
	// Verify no clients
	assert.Equal(t, 0, handler.GetConnectedClients())
}

func TestWebSocketUpgrader_CheckOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name:   "any origin allowed in development",
			origin: "http://localhost:3000",
			want:   true,
		},
		{
			name:   "any origin allowed",
			origin: "http://example.com",
			want:   true,
		},
		{
			name:   "empty origin allowed",
			origin: "",
			want:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			
			result := upgrader.CheckOrigin(req)
			assert.Equal(t, tt.want, result)
		})
	}
}

// Integration test helper to create a WebSocket connection
func createTestWebSocketConnection(t *testing.T, handler *WebSocketHandler) (*websocket.Conn, *httptest.Server) {
	mockService := &MockHybridStockService{}
	
	// Mock the service calls that happen during connection
	mockService.On("GetAllStocks").Return([]models.Stock{
		{
			ID:           1,
			Symbol:       "AAPL",
			CompanyName:  "Apple Inc.",
			CurrentPrice: 150.0,
		},
	})
	mockService.On("GetPerformanceData").Return(map[string]interface{}{
		"top_gainers": []interface{}{},
		"top_losers":  []interface{}{},
	})
	mockService.On("GetMarketOverview").Return(map[string]interface{}{
		"total_stocks": 1,
	})
	
	handler.stockService = mockService
	
	// Create test server
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)
	
	server := httptest.NewServer(router)
	
	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	
	// Create WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	
	return conn, server
}

func TestWebSocketHandler_ConnectionLifecycle(t *testing.T) {
	handler := NewWebSocketHandler(&MockHybridStockService{})
	
	// Initial state
	assert.Equal(t, 0, handler.GetConnectedClients())
	
	// Create connection
	conn, server := createTestWebSocketConnection(t, handler)
	defer server.Close()
	defer conn.Close()
	
	// Give connection time to establish
	time.Sleep(50 * time.Millisecond)
	
	// Should have 1 client
	assert.Equal(t, 1, handler.GetConnectedClients())
	
	// Close connection
	conn.Close()
	
	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocketHandler_MessageHandling(t *testing.T) {
	handler := NewWebSocketHandler(&MockHybridStockService{})
	conn, server := createTestWebSocketConnection(t, handler)
	defer server.Close()
	defer conn.Close()
	
	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	
	// Should receive initial data message
	var initialMessage map[string]interface{}
	err := conn.ReadJSON(&initialMessage)
	require.NoError(t, err)
	
	assert.Equal(t, "initial", initialMessage["type"])
	assert.NotNil(t, initialMessage["data"])
	
	// Should receive price updates
	var updateMessage map[string]interface{}
	err = conn.ReadJSON(&updateMessage)
	require.NoError(t, err)
	
	assert.Equal(t, "price_update", updateMessage["type"])
	assert.NotNil(t, updateMessage["data"])
}

// Benchmark tests for WebSocket performance
func BenchmarkWebSocketHandler_SimulatePriceChanges(b *testing.B) {
	handler := NewWebSocketHandler(&MockHybridStockService{})
	
	// Create test stocks
	stocks := make([]models.Stock, 100)
	for i := 0; i < 100; i++ {
		stocks[i] = models.Stock{
			ID:           uint(i + 1),
			Symbol:       "SYM" + string(rune(i)),
			CurrentPrice: 100.0,
			DailyChange:  1.0,
			ChangePercent: 1.0,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.simulatepriceChanges(stocks)
	}
}

func BenchmarkWebSocketHandler_BroadcastToClients(b *testing.B) {
	handler := NewWebSocketHandler(&MockHybridStockService{})
	
	message := map[string]interface{}{
		"type": "benchmark",
		"data": "test data",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.broadcastToClients(message)
	}
}