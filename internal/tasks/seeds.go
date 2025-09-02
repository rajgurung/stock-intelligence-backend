package tasks

// StockSeed represents seed data for stocks
type StockSeed struct {
	Symbol      string
	CompanyName string
	Sector      string
	Industry    string
	Exchange    string
	MarketCap   *int64
	IsActive    bool
}

// getStockSeeds returns a curated list of major stocks for seeding
// This includes S&P 500 companies across different sectors
func getStockSeeds() []StockSeed {
	// Helper function to create int64 pointer
	marketCap := func(value int64) *int64 {
		return &value
	}

	return []StockSeed{
		// Technology - Large Cap
		{"AAPL", "Apple Inc.", "Technology", "Consumer Electronics", "NASDAQ", marketCap(3000000000000), true},
		{"MSFT", "Microsoft Corporation", "Technology", "Software", "NASDAQ", marketCap(2800000000000), true},
		{"GOOGL", "Alphabet Inc.", "Technology", "Internet Services", "NASDAQ", marketCap(1600000000000), true},
		{"AMZN", "Amazon.com Inc.", "Consumer Discretionary", "E-commerce", "NASDAQ", marketCap(1500000000000), true},
		{"META", "Meta Platforms Inc.", "Technology", "Social Media", "NASDAQ", marketCap(800000000000), true},
		{"NVDA", "NVIDIA Corporation", "Technology", "Semiconductors", "NASDAQ", marketCap(1100000000000), true},
		{"TSLA", "Tesla Inc.", "Consumer Discretionary", "Electric Vehicles", "NASDAQ", marketCap(800000000000), true},
		{"NFLX", "Netflix Inc.", "Communication Services", "Streaming", "NASDAQ", marketCap(170000000000), true},
		
		// Financial Services
		{"JPM", "JPMorgan Chase & Co.", "Financial Services", "Banking", "NYSE", marketCap(420000000000), true},
		{"BAC", "Bank of America Corporation", "Financial Services", "Banking", "NYSE", marketCap(280000000000), true},
		{"WFC", "Wells Fargo & Company", "Financial Services", "Banking", "NYSE", marketCap(180000000000), true},
		{"GS", "The Goldman Sachs Group Inc.", "Financial Services", "Investment Banking", "NYSE", marketCap(120000000000), true},
		{"MS", "Morgan Stanley", "Financial Services", "Investment Banking", "NYSE", marketCap(140000000000), true},
		{"AXP", "American Express Company", "Financial Services", "Credit Services", "NYSE", marketCap(150000000000), true},
		{"V", "Visa Inc.", "Financial Services", "Payment Processing", "NYSE", marketCap(520000000000), true},
		{"MA", "Mastercard Incorporated", "Financial Services", "Payment Processing", "NYSE", marketCap(390000000000), true},
		
		// Healthcare
		{"UNH", "UnitedHealth Group Inc.", "Healthcare", "Health Insurance", "NYSE", marketCap(480000000000), true},
		{"JNJ", "Johnson & Johnson", "Healthcare", "Pharmaceuticals", "NYSE", marketCap(420000000000), true},
		{"PFE", "Pfizer Inc.", "Healthcare", "Pharmaceuticals", "NYSE", marketCap(220000000000), true},
		{"ABBV", "AbbVie Inc.", "Healthcare", "Pharmaceuticals", "NYSE", marketCap(290000000000), true},
		{"MRK", "Merck & Co. Inc.", "Healthcare", "Pharmaceuticals", "NYSE", marketCap(280000000000), true},
		{"TMO", "Thermo Fisher Scientific Inc.", "Healthcare", "Life Sciences Tools", "NYSE", marketCap(210000000000), true},
		{"ABT", "Abbott Laboratories", "Healthcare", "Medical Devices", "NYSE", marketCap(180000000000), true},
		
		// Consumer & Retail
		{"WMT", "Walmart Inc.", "Consumer Staples", "Retail", "NYSE", marketCap(530000000000), true},
		{"PG", "Procter & Gamble Company", "Consumer Staples", "Consumer Products", "NYSE", marketCap(380000000000), true},
		{"KO", "The Coca-Cola Company", "Consumer Staples", "Beverages", "NYSE", marketCap(260000000000), true},
		{"PEP", "PepsiCo Inc.", "Consumer Staples", "Beverages", "NASDAQ", marketCap(240000000000), true},
		{"COST", "Costco Wholesale Corporation", "Consumer Staples", "Retail", "NASDAQ", marketCap(320000000000), true},
		{"HD", "The Home Depot Inc.", "Consumer Discretionary", "Home Improvement", "NYSE", marketCap(380000000000), true},
		{"MCD", "McDonald's Corporation", "Consumer Discretionary", "Restaurants", "NYSE", marketCap(200000000000), true},
		{"NKE", "NIKE Inc.", "Consumer Discretionary", "Apparel", "NYSE", marketCap(180000000000), true},
		{"SBUX", "Starbucks Corporation", "Consumer Discretionary", "Restaurants", "NASDAQ", marketCap(110000000000), true},
		
		// Industrial & Manufacturing
		{"BA", "The Boeing Company", "Industrials", "Aerospace", "NYSE", marketCap(150000000000), true},
		{"CAT", "Caterpillar Inc.", "Industrials", "Construction Equipment", "NYSE", marketCap(160000000000), true},
		{"GE", "General Electric Company", "Industrials", "Conglomerate", "NYSE", marketCap(180000000000), true},
		{"MMM", "3M Company", "Industrials", "Industrial Conglomerate", "NYSE", marketCap(70000000000), true},
		{"HON", "Honeywell International Inc.", "Industrials", "Conglomerate", "NASDAQ", marketCap(140000000000), true},
		{"UPS", "United Parcel Service Inc.", "Industrials", "Logistics", "NYSE", marketCap(130000000000), true},
		{"RTX", "Raytheon Technologies Corporation", "Industrials", "Aerospace & Defense", "NYSE", marketCap(140000000000), true},
		
		// Energy & Utilities
		{"XOM", "Exxon Mobil Corporation", "Energy", "Oil & Gas", "NYSE", marketCap(450000000000), true},
		{"CVX", "Chevron Corporation", "Energy", "Oil & Gas", "NYSE", marketCap(280000000000), true},
		{"COP", "ConocoPhillips", "Energy", "Oil & Gas", "NYSE", marketCap(140000000000), true},
		{"SLB", "Schlumberger Limited", "Energy", "Oil Services", "NYSE", marketCap(60000000000), true},
		{"NEE", "NextEra Energy Inc.", "Utilities", "Electric Utilities", "NYSE", marketCap(150000000000), true},
		{"DUK", "Duke Energy Corporation", "Utilities", "Electric Utilities", "NYSE", marketCap(80000000000), true},
		
		// Telecommunications & Media
		{"VZ", "Verizon Communications Inc.", "Communication Services", "Telecommunications", "NYSE", marketCap(170000000000), true},
		{"T", "AT&T Inc.", "Communication Services", "Telecommunications", "NYSE", marketCap(120000000000), true},
		{"CMCSA", "Comcast Corporation", "Communication Services", "Media", "NASDAQ", marketCap(180000000000), true},
		{"DIS", "The Walt Disney Company", "Communication Services", "Entertainment", "NYSE", marketCap(200000000000), true},
		
		// Real Estate & REITs
		{"AMT", "American Tower Corporation", "Real Estate", "REITs", "NYSE", marketCap(90000000000), true},
		{"PLD", "Prologis Inc.", "Real Estate", "REITs", "NYSE", marketCap(100000000000), true},
		{"CCI", "Crown Castle Inc.", "Real Estate", "REITs", "NYSE", marketCap(60000000000), true},
		
		// Materials & Chemicals
		{"LIN", "Linde plc", "Materials", "Chemicals", "NYSE", marketCap(200000000000), true},
		{"APD", "Air Products and Chemicals Inc.", "Materials", "Chemicals", "NYSE", marketCap(60000000000), true},
		{"DOW", "Dow Inc.", "Materials", "Chemicals", "NYSE", marketCap(40000000000), true},
		{"DD", "DuPont de Nemours Inc.", "Materials", "Chemicals", "NYSE", marketCap(30000000000), true},
		
		// Additional Tech & Growth Stocks
		{"CRM", "Salesforce Inc.", "Technology", "Software", "NYSE", marketCap(220000000000), true},
		{"ORCL", "Oracle Corporation", "Technology", "Software", "NYSE", marketCap(320000000000), true},
		{"IBM", "International Business Machines Corporation", "Technology", "Software", "NYSE", marketCap(130000000000), true},
		{"INTC", "Intel Corporation", "Technology", "Semiconductors", "NASDAQ", marketCap(200000000000), true},
		{"AMD", "Advanced Micro Devices Inc.", "Technology", "Semiconductors", "NASDAQ", marketCap(240000000000), true},
		{"CSCO", "Cisco Systems Inc.", "Technology", "Networking", "NASDAQ", marketCap(200000000000), true},
		{"ADBE", "Adobe Inc.", "Technology", "Software", "NASDAQ", marketCap(240000000000), true},
		{"NOW", "ServiceNow Inc.", "Technology", "Software", "NYSE", marketCap(140000000000), true},
		{"UBER", "Uber Technologies Inc.", "Technology", "Transportation", "NYSE", marketCap(120000000000), true},
		{"SPOT", "Spotify Technology S.A.", "Communication Services", "Music Streaming", "NYSE", marketCap(50000000000), true},
		
		// Biotech & Pharma
		{"GILD", "Gilead Sciences Inc.", "Healthcare", "Biotechnology", "NASDAQ", marketCap(80000000000), true},
		{"AMGN", "Amgen Inc.", "Healthcare", "Biotechnology", "NASDAQ", marketCap(140000000000), true},
		{"BIIB", "Biogen Inc.", "Healthcare", "Biotechnology", "NASDAQ", marketCap(40000000000), true},
		{"REGN", "Regeneron Pharmaceuticals Inc.", "Healthcare", "Biotechnology", "NASDAQ", marketCap(90000000000), true},
		
		// Semiconductor Equipment & Materials
		{"ASML", "ASML Holding N.V.", "Technology", "Semiconductor Equipment", "NASDAQ", marketCap(300000000000), true},
		{"TSM", "Taiwan Semiconductor Manufacturing Company", "Technology", "Semiconductors", "NYSE", marketCap(500000000000), true},
		{"AVGO", "Broadcom Inc.", "Technology", "Semiconductors", "NASDAQ", marketCap(600000000000), true},
		{"QCOM", "QUALCOMM Incorporated", "Technology", "Semiconductors", "NASDAQ", marketCap(190000000000), true},
		{"TXN", "Texas Instruments Incorporated", "Technology", "Semiconductors", "NASDAQ", marketCap(170000000000), true},
		
		// E-commerce & Digital Services
		{"BABA", "Alibaba Group Holding Limited", "Consumer Discretionary", "E-commerce", "NYSE", marketCap(200000000000), true},
		{"SHOP", "Shopify Inc.", "Technology", "E-commerce Software", "NYSE", marketCap(80000000000), true},
		{"SQ", "Block Inc.", "Technology", "Financial Technology", "NYSE", marketCap(40000000000), true},
		{"PYPL", "PayPal Holdings Inc.", "Financial Services", "Payment Processing", "NASDAQ", marketCap(80000000000), true},
		
		// Automotive
		{"F", "Ford Motor Company", "Consumer Discretionary", "Automotive", "NYSE", marketCap(50000000000), true},
		{"GM", "General Motors Company", "Consumer Discretionary", "Automotive", "NYSE", marketCap(60000000000), true},
		{"RIVN", "Rivian Automotive Inc.", "Consumer Discretionary", "Electric Vehicles", "NASDAQ", marketCap(20000000000), true},
		
		// Airlines & Travel
		{"AAL", "American Airlines Group Inc.", "Industrials", "Airlines", "NASDAQ", marketCap(10000000000), true},
		{"DAL", "Delta Air Lines Inc.", "Industrials", "Airlines", "NYSE", marketCap(30000000000), true},
		{"UAL", "United Airlines Holdings Inc.", "Industrials", "Airlines", "NASDAQ", marketCap(25000000000), true},
		
		// Gaming & Entertainment
		{"EA", "Electronic Arts Inc.", "Communication Services", "Gaming", "NASDAQ", marketCap(40000000000), true},
		{"ATVI", "Activision Blizzard Inc.", "Communication Services", "Gaming", "NASDAQ", marketCap(60000000000), true},
		{"TTWO", "Take-Two Interactive Software Inc.", "Communication Services", "Gaming", "NASDAQ", marketCap(25000000000), true},
	}
}