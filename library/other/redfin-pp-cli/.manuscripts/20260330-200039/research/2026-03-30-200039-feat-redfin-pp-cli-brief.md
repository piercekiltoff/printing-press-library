# Redfin CLI Brief

## API Identity
- Domain: Real estate listings, property data, market trends, valuations
- Users: Homebuyers, investors, agents, data analysts, real estate researchers
- Data profile: No official public API. Unofficial "Stingray" API reverse-engineered from redfin.com. No auth required for most endpoints. Also has a public Data Center with downloadable market datasets (TSV/CSV).

## API Endpoints (Stingray - Unofficial)

### Search & GIS
- `/stingray/api/gis` - Property search by polygon/region with filters
- `/stingray/api/gis-csv` - CSV download of search results (up to 350)
- `/stingray/do/gis-search` - Alternative search endpoint

### Property Details
- `/stingray/api/home/details/initialInfo` - Get listingId from URL path
- `/stingray/api/home/details/aboveTheFold` - Images, price, top details
- `/stingray/api/home/details/belowTheFold` - Amenities, history, characteristics
- `/stingray/api/home/details/mainHouseInfoPanelInfo` - Info panel
- `/stingray/api/home/details/neighborhoodStats/statsInfo` - Walk/bike/transit scores
- `/stingray/api/home/details/avmHistoricalData` - AVM valuation history
- `/stingray/api/home/details/commute/commuteInfo` - Commute data

### Property Sub-resources
- `/stingray/api/home/details/owner-estimate` - Owner valuation
- `/stingray/api/home/details/similar-listings` - Comparable active listings
- `/stingray/api/home/details/similar-sold` - Recently sold comparables
- `/stingray/api/home/details/nearby-homes` - Adjacent properties
- `/stingray/api/home/details/property-comments` - User comments
- `/stingray/api/home/details/building-details` - Building info
- `/stingray/api/home/details/cost-of-home-ownership` - Cost breakdown
- `/stingray/api/home/details/descriptive-paragraph` - Description
- `/stingray/api/home/details/tour-insights` - Tour info
- `/stingray/api/home/details/stats` - Property statistics

### Rentals
- `/stingray/api/v1/rentals/{rentalId}/floorPlans` - Floor plans, units, pricing

### Region/Market
- `/stingray/api/region/{type}/{regionId}/{code}/aggregate-trends` - Market trends
- `/stingray/api/region/{type}/{regionId}/{code}/trends` - Detailed trends

### Feeds
- `/newest_listings.xml` - Sitemap of newest listings
- `/sitemap_com_latest_updates.xml` - Recently updated listings

### Data Center (Public Downloads)
- Weekly housing market data (TSV, gzip)
- Monthly housing market data by geo level
- Existing home sales data
- Redfin Home Price Index (RHPI)
- Investor data
- Rental market data
- Buyer vs seller dynamics

## Top Workflows
1. Search properties by location with filters (price, beds, baths, sqft, etc.)
2. Get detailed property info (value estimates, comparables, neighborhood scores)
3. Track market trends by region (median prices, inventory, days on market)
4. Monitor new listings via sitemap feeds
5. Download bulk market data for analysis

## Table Stakes
- Property search with geo/filter parameters
- Property detail lookup with all sub-resources
- Comparable properties (similar listed + similar sold)
- Market trend analysis by region
- Property valuation estimates (AVM)
- Neighborhood scores (walk, bike, transit)

## Data Layer
- Primary entities: Properties, Regions, Trends, Valuations, Comparables
- Sync cursor: Sitemap lastmod timestamps, Data Center weekly updates
- FTS/search: Full-text search across property descriptions, addresses, neighborhoods

## Product Thesis
- Name: redfin-pp-cli
- Why it should exist: No CLI tool provides offline-first, SQLite-backed Redfin property search with cross-entity queries. Existing tools are either Python-only scrapers (reteps/redfin, redfin-scraper), server-based systems (gredfin), or single-purpose valuators (crib). None offer agent-native output, offline search, trend analysis, or composable real estate intelligence from the terminal.

## Build Priorities
1. Property search with full filter support + SQLite persistence
2. Property detail retrieval with all sub-resources
3. Market trend queries by region
4. Comparables engine (similar listings + sold)
5. Valuation tracking (AVM history)
6. Bulk data center download + import
7. New listing monitoring via feeds
8. Cross-entity intelligence commands
