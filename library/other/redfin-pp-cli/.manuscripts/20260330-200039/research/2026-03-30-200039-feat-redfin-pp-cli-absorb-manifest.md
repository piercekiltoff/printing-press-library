# Redfin CLI Absorb Manifest

## Ecosystem Tools Found

1. **reteps/redfin** (Python, 140 stars) - Wrapper around unofficial Stingray API
2. **gredfin** (Go) - Server/worker scraping system with PostGIS
3. **crib** (Go, 3 stars) - Property valuation CLI (Redfin + Zillow)
4. **redfin-scraper** (Python, PyPI) - Bulk scraper using Stingray API
5. **HomeHarvest** (Python, 658 stars) - Multi-platform scraper (Redfin, Zillow, Realtor)
6. **RedfinScraper** (Python, 97 stars) - Redfin data scraper
7. **Apify Redfin MCP** - MCP server for property scraping
8. **Apify Redfin Scraper** - Actor-based scraper with agent details
9. **go-redfin-archiver** (Go) - Download listing images
10. **dreed47/redfin** (Python) - Home Assistant sensor for property estimates
11. **Domapus** (TypeScript) - Housing market heatmap by ZIP
12. **RedfinPlus** - Browser extension with API docs
13. **RapidAPI Unofficial Redfin** - Hosted API proxy

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Search by address | reteps/redfin `search()` | `search <query>` | FTS5 offline, SQLite cached, --json |
| 2 | Property initial info | reteps/redfin `initial_info()` | `property info <url>` | Offline cache, --json, --select |
| 3 | Above the fold details | reteps/redfin `above_the_fold()` | `property details <id>` | Combined with below-fold, complete view |
| 4 | Below the fold details | reteps/redfin `below_the_fold()` | `property details <id>` | SQLite persisted, searchable |
| 5 | AVM valuation | reteps/redfin `avm_details()` | `property value <id>` | Historical tracking in SQLite |
| 6 | AVM history | reteps/redfin `avm_historical()` | `property value-history <id>` | Chart-ready output, trend analysis |
| 7 | Neighborhood stats | reteps/redfin `neighborhood_stats()` | `property neighborhood <id>` | Walk/bike/transit scores cached |
| 8 | Similar listings | reteps/redfin `similar_listings()` | `property comps <id>` | Side-by-side comparison, --json |
| 9 | Similar sold | reteps/redfin `similar_sold()` | `property comps --sold <id>` | Historical comp tracking |
| 10 | Nearby homes | reteps/redfin `nearby_homes()` | `property nearby <id>` | Radius-based, map-aware |
| 11 | Owner estimate | reteps/redfin `owner_estimate()` | `property estimate <id>` | Track estimate changes over time |
| 12 | Cost of ownership | reteps/redfin `cost_of_home_ownership()` | `property costs <id>` | Monthly/annual breakdown |
| 13 | Property comments | reteps/redfin `property_comments()` | `property comments <id>` | FTS searchable |
| 14 | Building details | reteps/redfin `building_details_page()` | `property building <id>` | Offline cached |
| 15 | Tour insights | reteps/redfin `tour_insights()` | `property tours <id>` | Scheduling info |
| 16 | Floor plans | reteps/redfin `floor_plans()` | `rental plans <id>` | Unit availability + pricing |
| 17 | Descriptive paragraph | reteps/redfin `descriptive_paragraph()` | `property describe <id>` | FTS indexed |
| 18 | Hood photos | reteps/redfin `hood_photos()` | `property photos --hood <id>` | Download/cache |
| 19 | Page tags | reteps/redfin `page_tags()` | Internal metadata | SEO/metadata extraction |
| 20 | Property stats | reteps/redfin `stats()` | `property stats <id>` | Regional context |
| 21 | GIS polygon search | RedfinPlus, Stingray API | `search --polygon <coords>` | SQLite cached results |
| 22 | CSV bulk download | Stingray gis-csv | `search --csv <region>` | Auto-import to SQLite |
| 23 | Bulk zip scraping | redfin-scraper | `sync --zips <zip1,zip2>` | Incremental sync, offline |
| 24 | Multi-platform search | HomeHarvest | `search` (Redfin-focused but superior) | SQLite + FTS5, offline-first |
| 25 | Image download | go-redfin-archiver | `property photos <id> --download` | Organized by listing |
| 26 | Property valuation | crib | `property value <addr>` | Historical tracking, not just snapshot |
| 27 | Server/worker scraping | gredfin | `sync` command | No server needed, local SQLite |
| 28 | Zipcode-by-zipcode search | gredfin workers | `sync --zips` | Single binary, no PostGIS needed |
| 29 | Agent details | Apify Redfin MCP | `agent <name>` | Agent history, listing count |
| 30 | Home Assistant sensor | dreed47/redfin | `watch <addr>` | Price alerts via CLI |
| 31 | Market heatmap data | Domapus | `trends heatmap --zip` | Terminal-friendly, --json |
| 32 | New listings feed | Sitemap XML | `feed new` | Auto-sync, alerts |
| 33 | Updated listings feed | Sitemap XML | `feed updates` | Change detection |
| 34 | Region trends | Stingray API | `trends <region>` | SQLite historical |
| 35 | Aggregate trends | Stingray API | `trends aggregate <region>` | Cross-region comparison |
| 36 | Data Center download | Redfin Data Center | `data download <dataset>` | Import weekly/monthly TSV |
| 37 | RHPI index | Redfin Data Center | `data rhpi` | Home price index tracking |
| 38 | Investor data | Redfin Data Center | `data investors` | Investment activity |
| 39 | Rental market data | Redfin Data Center | `data rentals` | Rental trends |
| 40 | Commute info | Stingray API | `property commute <id>` | Commute times cached |

### Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 1 | Price trajectory tracking | `track <addr>` | Requires historical AVM snapshots stored locally over time | 8/10 | reteps/redfin avm_historical exists but no persistence; Home Assistant sensor tracks single property |
| 2 | Comp intelligence | `comp-analysis <addr>` | Requires join across property details + similar_listings + similar_sold + AVM data | 9/10 | No tool combines comps with valuation trends; gredfin stores but requires PostGIS server |
| 3 | Market pulse | `pulse <region>` | Requires local aggregate of trends + new listings feed + inventory changes | 8/10 | Data Center has raw data but no CLI synthesizes across feeds |
| 4 | Deal finder | `deals --region <id>` | Requires local join: listing price vs AVM estimate vs neighborhood median vs days-on-market | 9/10 | Common investor workflow, no existing tool automates; redfin-scraper + manual analysis |
| 5 | Neighborhood comparison | `compare-hoods <hood1> <hood2>` | Requires join across neighborhood stats + trends + property data for two areas | 7/10 | Domapus has heatmap but no comparison; RedfinPlus shows one neighborhood at a time |
| 6 | Investment calculator | `invest <addr>` | Requires join: cost_of_ownership + AVM + rental estimates + trend trajectory | 8/10 | crib gives one-shot valuation; no tool combines ownership cost with rental yield |
| 7 | Stale listing radar | `stale --days 30 --region <id>` | Requires local tracking of days-on-market + price changes over sync cycles | 7/10 | gredfin workers track but need server; common buyer workflow |
| 8 | Price alert watchlist | `watch add <addr> --below 500000` | Requires persistent watchlist + periodic sync + threshold comparison | 8/10 | Home Assistant sensor does this for one property; no CLI does it at scale |
| 9 | Market report generator | `report <region>` | Requires synthesis of trends + inventory + RHPI + new listings into one document | 7/10 | Data Center has raw datasets but no CLI generates reports |
| 10 | Cross-zip analysis | `analyze-zips <zip1> <zip2> <zip3>` | Requires local store of multiple ZIP trends + property data for comparison | 7/10 | redfin-scraper scrapes by zip but no comparison; Domapus shows heatmap only |


### Brainstorm Features (user-requested)

| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 11 | Portfolio tracker | `portfolio` / `portfolio add <addr>` | Track multiple owned/watched properties with consolidated value trends, equity changes, and market position in SQLite | 10/10 | User's #1 requested feature; no existing tool does this from CLI |
| 12 | Smart scoring | `score <property-id> --profile <name>` | Score properties against custom criteria profiles (commute, schools, price-to-AVM ratio) using local data joins | 9/10 | User workflow: home shopping + investment analysis |
| 13 | Export engine | `export <query> --format csv/json/xlsx` | Export any search/analysis to file, breaking data out of browser | 8/10 | User pain: can't export/save data easily from Redfin |
| 14 | Listing change history | `history <property-id>` | Track every price/status change over sync cycles in SQLite | 9/10 | User pain: no historical tracking on Redfin |
| 15 | Mortgage calculator | `mortgage <price> --down 20% --rate 6.5` | Monthly payments, DTI, scenario comparison + AVM + cost data | 8/10 | User workflow: investment analysis |
| 16 | School district analysis | `schools <property-id>` / `schools compare <h1> <h2>` | Join neighborhood + school data across properties for scoring | 8/10 | User workflow: home shopping with school criteria |
