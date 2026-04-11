# Kalshi CLI Absorb Manifest

## Sources Cataloged
1. austron24/kalshi-cli (Python, 14★) — most complete general CLI
2. JThomasDevs/kalshi-cli (Python, 1★) — smart search, interactive drill-down
3. OctagonAI/kalshi-trading-bot-cli (TypeScript, 168★) — AI research, Kelly sizing, SQLite
4. newyorkcompute/kalshi (TypeScript, 3★) — 14 MCP tools + TUI + market maker
5. yakub268/kalshi-mcp (TypeScript, 0★) — production-grade MCP, trending markets
6. fsctl/go-kalshi (Go, 2★) — Go client, semantic order types
7. austron24/kalshi-trader-plugin — Claude Code plugin for AI-assisted trading
8. kalshi-python (official SDK) — auto-generated from OpenAPI

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Market search by keyword | austron24 `find`, JThomasDevs `search` | `markets search <query>` + FTS5 | Works offline, regex, SQL composable |
| 2 | List markets with filters | austron24 `markets`, newyorkcompute `get_markets` | `markets list --status --category --series` | Offline + --json/--csv/--select |
| 3 | Get market details | austron24 `market`, yakub268 `get_market_details` | `markets get <ticker>` | Includes cached orderbook depth |
| 4 | Orderbook depth | austron24 `orderbook`, newyorkcompute `get_orderbook` | `markets orderbook <ticker>` | Snapshot stored in SQLite for spread tracking |
| 5 | Trade history | austron24 `trades`, newyorkcompute `get_trades` | `markets trades <ticker>` | Persisted for volume analysis |
| 6 | Market candlesticks | Kalshi API (historical) | `markets candles <ticker> --period` | Local OHLC data for charting |
| 7 | List events | newyorkcompute `get_events` | `events list --category --status` | FTS5 search, offline browse |
| 8 | Get event details | newyorkcompute `get_event` | `events get <ticker>` | Includes all child markets |
| 9 | List series | austron24 `series` | `series list` | FTS5 search |
| 10 | Get series details | austron24 `series` | `series get <ticker>` | Includes child events + markets |
| 11 | Portfolio balance | austron24 `balance`, yakub268 `get_portfolio` | `portfolio balance` | Historical tracking in SQLite |
| 12 | Portfolio positions | austron24 `positions`, JThomasDevs enriched | `portfolio positions` | Return %, market titles, P&L |
| 13 | Portfolio fills | austron24 `fills`, newyorkcompute `get_fills` | `portfolio fills` | Persisted for win rate analysis |
| 14 | Portfolio settlements | austron24 `settlements`, newyorkcompute `get_settlements` | `portfolio settlements` | P&L attribution by category |
| 15 | Portfolio summary | austron24 `summary` | `portfolio summary` | Total value, resting orders, exposure |
| 16 | Place order | austron24 `buy`/`sell`, newyorkcompute `create_order` | `orders create --side --type --price --count` | --dry-run, cost preview, --json |
| 17 | Cancel order | austron24 `cancel`, newyorkcompute `cancel_order` | `orders cancel <order_id>` | --dry-run confirmation |
| 18 | Batch cancel | austron24 `cancel-all`, newyorkcompute `batch_cancel_orders` | `orders cancel-all --market --side` | Scoped batch with preview |
| 19 | Amend order | Kalshi API | `orders amend <order_id> --price --count` | --dry-run |
| 20 | List orders | austron24 `orders`, newyorkcompute `get_orders` | `orders list --status --market` | Historical orders in SQLite |
| 21 | Order queue position | Kalshi API | `orders queue <order_id>` | Shows queue depth |
| 22 | Order groups | Kalshi API | `order-groups list/create/delete/trigger` | Full lifecycle management |
| 23 | Close position | austron24 `close`, fsctl semantic types | `positions close <ticker> --side` | Validates position exists first |
| 24 | Human-readable prices | JThomasDevs ($0.68 / 68%) | All price output: $0.68 (68%) | Default human, --json for raw |
| 25 | Human-readable expiry | JThomasDevs (8h 35m) | Relative time on all market output | Contextual: "8h 35m", "3 days" |
| 26 | Market type detection | JThomasDevs (binary/range/multi/parlay) | Auto-detected from event structure | Adaptive table display |
| 27 | Interactive drill-down | JThomasDevs series→events→markets | `browse` command with numbered selection | Navigate the hierarchy |
| 28 | Search alias expansion | JThomasDevs ("nfl"→"football") | Alias config in SQLite | User-extensible aliases |
| 29 | Trending markets | yakub268 `get_trending_markets` | `markets trending` | By volume, movers, new listings |
| 30 | Exchange status | Kalshi API | `exchange status` | Health check for trading hours |
| 31 | Exchange announcements | Kalshi API | `exchange announcements` | Latest platform news |
| 32 | Exchange schedule | Kalshi API | `exchange schedule` | Trading hours/holidays |
| 33 | API key management | Kalshi API | `api-keys list/create/delete` | Key lifecycle from CLI |
| 34 | Account limits | Kalshi API | `account limits` | Position/order limits |
| 35 | Historical markets | Kalshi API | `historical markets --status settled` | Browse settled markets |
| 36 | Historical fills/orders | Kalshi API | `historical fills/orders` | Full trade history |
| 37 | RFQ/Quotes | Kalshi API (communications) | `rfq list/create/delete`, `quotes list/create/accept` | Block trade negotiation |
| 38 | Subaccounts | Kalshi API | `subaccounts create/transfer/balances` | Multi-account management |
| 39 | Live data/milestones | Kalshi API | `live-data milestone <id>` | Real-time event data |
| 40 | Game stats | Kalshi API | `live-data game-stats <milestone_id>` | Sports data integration |
| 41 | Structured targets | Kalshi API | `targets list/get` | Target-based market sets |
| 42 | Tags/categories search | Kalshi API | `search tags --category` | Browse market taxonomy |
| 43 | Sport filters | Kalshi API | `search filters --sport` | Sport-specific market filters |
| 44 | Fee changes | Kalshi API | `series fee-changes` | Track fee schedule updates |
| 45 | Incentive programs | Kalshi API | `incentive-programs list` | Active reward programs |
| 46 | Multivariate events | Kalshi API | `multivariate list/get/create` | Complex event collections |
| 47 | Doctor/health check | Standard PP CLI | `doctor` | Auth, connectivity, env validation |
| 48 | JSON output | austron24 | `--json` on all commands | Valid JSON, pipes to jq |
| 49 | OpenAPI browser | austron24 `endpoints`/`show`/`schema` | `api endpoints/show/schema` | Built-in API reference |
| 50 | Demo environment | Kalshi API | `--demo` flag or `KALSHI_ENV=demo` | Switch to sandbox instantly |

## Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This | Score |
|---|---------|---------|------------------------|-------|
| 1 | Portfolio attribution | `portfolio attribution --by category --period 30d` | Joins fills + settlements + events + series data to compute P&L by category/series over time. No tool tracks this historically. | 9/10 |
| 2 | Odds history tracker | `markets history <ticker> --since 7d` | Records market prices at every sync. Shows how odds moved over time with price chart. Requires periodic snapshots stored in SQLite. | 8/10 |
| 3 | Win rate analytics | `portfolio winrate --by category` | Calculates W/L ratio, expected value, and ROI across all settled positions. Requires joining fills + settlements + market metadata. | 8/10 |
| 4 | Settlement calendar | `portfolio calendar` | Shows upcoming settlements with your positions, expected payouts, and category breakdown. Joins positions + events + market expiry data. | 8/10 |
| 5 | Market movers | `markets movers --period 24h` | Shows markets with biggest price changes since last sync. Requires prior price snapshots to compute deltas. | 7/10 |
| 6 | Cross-market correlation | `markets correlate <ticker1> <ticker2>` | Compares price history of two markets to find correlated events. Requires historical price data in SQLite for both markets. | 7/10 |
| 7 | Event lifecycle | `events lifecycle <ticker>` | Tracks an event from creation through settlement — price progression, volume, key moments. Requires temporal market data. | 7/10 |
| 8 | Category heatmap | `markets heatmap` | Aggregates volume, open interest, and price movement by category. Shows which categories are hot. Requires local market data aggregation. | 7/10 |
| 9 | Exposure analysis | `portfolio exposure` | Breaks down total exposure by category, correlation risk, and concentration. Requires joining positions + market metadata + category data. | 8/10 |
| 10 | Stale position finder | `portfolio stale --days 30` | Finds positions in markets approaching expiry where you haven't acted. Requires local position + market expiry joins. | 7/10 |

