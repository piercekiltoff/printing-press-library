# Acceptance Report: kalshi-trade-manual

## Level: Full Dogfood

## Tests: 10/10 passed

| # | Test | Command | Result |
|---|------|---------|--------|
| 1 | Doctor (auth + connectivity) | `doctor` | PASS — Config OK, Auth RSA-PSS configured, API reachable, Credentials valid |
| 2 | Exchange status | `exchange get-status --json` | PASS — `exchange_active: true, trading_active: true` |
| 3 | List markets | `markets get --limit 5 --json` | PASS — Returns real market data with tickers, prices, event info |
| 4 | Portfolio balance | `portfolio get-balance --json` | PASS — Returns `balance: 0, portfolio_value: 0` (correct for new account) |
| 5 | Portfolio positions | `portfolio --json --limit 5` | PASS — Returns empty positions (correct for new account) |
| 6 | Sync markets | `sync --resources markets --full` | PASS — 20,000+ markets synced to SQLite |
| 7 | Events list | `events --limit 5 --json --compact` | PASS — Real events (Elon Mars, Pope, Climate) |
| 8 | Output modes (JSON) | `portfolio get-balance --json` | PASS — Valid JSON with meta.source envelope |
| 9 | Markets heatmap (transcendence) | `markets heatmap` | PASS — Shows categories from synced data |
| 10 | Markets movers (transcendence) | `markets movers --limit 5` | PASS — Shows market data from sync |

## Fixes Applied During Dogfood: 3
1. **Sync fix** (Printing Press issue) — Added Kalshi-specific wrapper keys to `extractPageItems` (markets, events, series, etc.)
2. **Sync fix** (Printing Press issue) — Added "ticker" to ID extraction in `UpsertBatch` (Kalshi uses ticker, not id)
3. **Query fix** (CLI fix) — Changed `status = 'open'` to `IN ('open', 'active')` in transcendence queries (Kalshi uses 'active')

## Printing Press Issues: 2
1. `extractPageItems` only tries generic wrapper keys (data, results, items) — should also try the resource name as a key
2. `UpsertBatch` only looks for "id" as primary key — should also try "ticker", "slug", etc.

## Gate: PASS
