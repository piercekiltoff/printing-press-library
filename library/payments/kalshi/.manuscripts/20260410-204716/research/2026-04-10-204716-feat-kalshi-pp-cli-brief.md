# Kalshi CLI Brief

## API Identity
- Domain: CFTC-regulated prediction market exchange (binary contracts on real-world events)
- Users: Traders, quants, automated market makers, AI agents, researchers, political/sports/economics enthusiasts
- Data profile: 91 REST endpoints (OpenAPI 3.x), RSA-PSS signature auth, demo sandbox available
- Base URL: https://api.elections.kalshi.com/trade-api/v2
- Spec: https://docs.kalshi.com/openapi.yaml
- Hierarchy: Series → Events → Markets (binary yes/no contracts)
- Categories: politics, economics, climate, technology, entertainment, sports, crypto

## Reachability Risk
- None — official documented API with OpenAPI spec, auto-generated SDKs, active community

## Auth Profile
- Type: RSA-PSS signature (not simple bearer token)
- Headers: KALSHI-ACCESS-KEY (API key UUID), KALSHI-ACCESS-TIMESTAMP (ms), KALSHI-ACCESS-SIGNATURE (base64)
- Signing: SHA256 with PSS padding, message = timestamp + method + path (no query params)
- Key management: Private key generated once at kalshi.com/api, cannot be recovered
- Demo env: https://demo-api.kalshi.co/trade-api/v2

## Top Workflows
1. Market discovery & research — browse events by category, search by keyword, check odds/prices
2. Portfolio monitoring — balance, positions, P&L tracking, settlement history
3. Order management — place limit/market orders, amend, cancel, batch operations, order groups
4. Historical analysis — candlesticks, trade history, price trends, volume tracking
5. Market making — orderbook depth analysis, spread tracking, competitive quoting

## Table Stakes (from competitors)
- RSA-PSS signature auth (all competitors)
- Market search by keyword/category/ticker (austron24, JThomasDevs)
- Orderbook depth visualization (austron24)
- Portfolio positions with return % (JThomasDevs)
- Human-readable prices ($0.68 / 68%) and time-to-expiry (JThomasDevs)
- Trade history and fills (austron24)
- JSON output for scripting (austron24)
- Market type detection (binary/range/multi-outcome) (JThomasDevs)
- OpenAPI documentation browser (austron24)
- AI-powered probability estimation (OctagonAI)
- SQLite caching with TTL (OctagonAI)
- TUI dashboard (newyorkcompute)

## Data Layer
- Primary entities: series, events, markets, orderbooks, trades, positions, fills, settlements, orders, candlesticks
- Sync cursor: cursor-based pagination on most list endpoints
- FTS/search: market titles, event tickers, series descriptions, tags, categories
- Historical: candlestick OHLC data, trade history, fill history, settlement history
- Live data: milestones, game stats, structured targets

## User Vision
- User has API key + RSA private key ready
- Be careful with live testing — real monetary data
- No mutations during smoke testing

## Product Thesis
- Name: kalshi-pp-cli
- Why it should exist: No existing Kalshi CLI has a local data layer. The best competitor (OctagonAI, 168 stars) focuses on AI trading, not general-purpose market intelligence. austron24 (14 stars) is the most complete general CLI but has zero offline capability. A CLI with SQLite persistence enables historical tracking, cross-market analysis, portfolio attribution, and event correlation that no tool currently offers — all offline, all composable, all agent-native.

## Build Priorities
1. Data layer: series, events, markets, orderbooks, trades, positions, fills, settlements, candlesticks
2. Full sync with cursor pagination + incremental updates
3. Every read endpoint from the 91-endpoint spec (market data, portfolio, historical, search)
4. Transcendence: cross-market correlation, portfolio attribution, historical odds tracking, event lifecycle analysis
5. Agent-native: --json, --select, --csv, --dry-run, typed exit codes, --compact
