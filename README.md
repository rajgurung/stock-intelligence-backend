# Stock Intelligence Backend

A high-performance Go-based REST API server for real-time stock market data analysis and intelligence. Built with Gin framework, PostgreSQL, and WebSocket support for live data streaming.

## 🚀 Features

- **Real-time Stock Data**: Integration with Alpha Vantage API for live market data
- **RESTful API**: Comprehensive endpoints for stock data, market analysis, and system monitoring
- **WebSocket Support**: Real-time price updates and live data streaming
- **PostgreSQL Integration**: Robust data persistence with migration system
- **Background Services**: Automated data synchronization and cleanup jobs
- **Rate Limiting**: Built-in API rate limiting and error handling
- **Docker Support**: Containerized deployment ready

## 🏗️ Architecture

```
backend/
├── cmd/                    # Command-line tools and utilities
│   ├── data-fetcher/      # Alpha Vantage data fetcher
│   ├── migrate/           # Database migration tool
│   ├── scheduler/         # Background job scheduler
│   └── seed/             # Database seeding utility
├── internal/
│   ├── database/         # Database connection and migrations
│   ├── handlers/         # HTTP request handlers
│   ├── models/           # Data models and structures
│   ├── services/         # Business logic and external APIs
│   └── tasks/           # Background task management
├── migrations/           # SQL migration files
└── main.go              # Application entry point
```

## 🛠️ Prerequisites

- Go 1.21 or higher
- PostgreSQL 13 or higher
- Redis (optional, for caching)
- Alpha Vantage API key

## ⚡ Quick Start

### 1. Environment Setup

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
DATABASE_URL=postgresql://user:password@localhost/stock_intelligence
REDIS_URL=redis://localhost:6379
ALPHA_VANTAGE_API_KEY=your_api_key_here
JWT_SECRET=your_jwt_secret
PORT=8080
GIN_MODE=debug
```

### 2. Database Setup

```bash
# Install dependencies
go mod tidy

# Run migrations
go run cmd/migrate/main.go up

# Seed initial data (optional)
go run cmd/seed/main.go
```

### 3. Start Development Server

```bash
# Standard run
go run main.go

# With race detection
go run -race main.go

# With hot reload (requires air)
go install github.com/cosmtrek/air@latest
air
```

The server will start on `http://localhost:8080`

## 📡 API Endpoints

### Stock Data
- `GET /api/v1/stocks` - Get all stocks with pagination
- `GET /api/v1/stocks/:symbol` - Get specific stock data
- `GET /api/v1/stocks/:symbol/performance` - Get historical performance
- `GET /api/v1/stocks/price-range` - Filter stocks by price range

### Market Data
- `GET /api/v1/market/overview` - Market overview and statistics
- `GET /api/v1/market/performance` - Market performance data
- `GET /api/v1/market/sectors` - Sector analysis data

### System Monitoring
- `GET /health` - Health check endpoint
- `GET /api/v1/system/health` - Detailed system health
- `GET /api/v1/system/api-status` - Alpha Vantage API status
- `GET /api/v1/sync/status` - Data synchronization status

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/handlers

# Run benchmarks
go test -bench=. ./...
```

## 🐳 Docker Deployment

```bash
# Build image
docker build -t stock-intelligence-backend .

# Run container
docker run -p 8080:8080 --env-file .env stock-intelligence-backend
```

## 📊 Data Sources

- **Alpha Vantage**: Primary data source for stock prices and market data
- **PostgreSQL**: Local data storage and caching
- **Rate Limiting**: 5 calls/minute, 500 calls/day (free tier)

## 🛡️ Security Features

- Input validation and sanitization
- Rate limiting protection
- CORS configuration
- Environment variable security
- SQL injection protection via ORM

## 🔧 Development Tools

```bash
# Database migrations
go run cmd/migrate/main.go up
go run cmd/migrate/main.go down

# Data fetching
go run cmd/data-fetcher/main.go

# Background scheduler
go run cmd/scheduler/main.go

# Manual data sync
go run cmd/trigger-sync/main.go --symbol AAPL
```

## 📈 Performance

- **Concurrent Processing**: Goroutines for parallel API calls
- **Connection Pooling**: PostgreSQL connection optimization  
- **Caching Strategy**: Redis integration for frequently accessed data
- **Background Jobs**: Non-blocking data synchronization

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- [Stock Intelligence Frontend](https://github.com/rajgurung/stock-intelligence-frontend) - React/Next.js dashboard