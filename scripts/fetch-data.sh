#!/bin/bash

# Stock Data Fetcher Script
# This script runs the data fetcher to populate missing stock prices

set -e  # Exit on any error

echo "🚀 Stock Data Fetcher"
echo "==================="

# Change to backend directory
cd "$(dirname "$0")/.."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found"
    echo "Please create a .env file with DATABASE_URL and ALPHA_VANTAGE_API_KEY"
    exit 1
fi

# Run the data fetcher
echo "📊 Starting data fetch process..."
echo ""

go run cmd/data-fetcher/main.go

echo ""
echo "✅ Data fetch completed!"
echo ""
echo "💡 Tips:"
echo "   - Run this script daily to populate missing stock data"
echo "   - Alpha Vantage free tier allows 25 calls/day"
echo "   - Use 'go run cmd/scheduler/main.go' for automatic scheduling"