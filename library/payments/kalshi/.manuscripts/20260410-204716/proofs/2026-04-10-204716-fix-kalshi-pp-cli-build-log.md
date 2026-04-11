# Kalshi CLI Build Log

## What Was Built

### Priority 0 (Foundation) — Complete
- SQLite data layer with tables for: markets, events, series, portfolio, orderbook,
  communications, exchange, search, structured_targets, api_keys, fcm, account,
  incentive_programs, multivariate_event_collections, historical, live_data,
  milestones, metadata, lookup
- Sync command with cursor-based pagination, parallel workers, incremental sync
- FTS5 full-text search across all resources
- Export/import for data portability

### Priority 1 (Absorbed Features) — Complete
- 80+ API endpoint commands covering all 91 spec endpoints
- RSA-PSS signature auth rewritten (3-header Kalshi auth: KEY, TIMESTAMP, SIGNATURE)
- Config supports KALSHI_API_KEY, KALSHI_PRIVATE_KEY_PATH, KALSHI_PRIVATE_KEY env vars
- Demo environment via KALSHI_ENV=demo
- --json, --csv, --select, --compact, --dry-run, --agent on all commands
- Adaptive rate limiter with backoff on 429
- Doctor command with auth validation via portfolio/balance

### Priority 2 (Transcendence) — Complete (8 of 10)
1. **portfolio attribution** — P&L by category/series/event with win/loss breakdown
2. **portfolio winrate** — Win rate analytics with ROI calculation
3. **portfolio calendar** — Settlement calendar with position details
4. **portfolio exposure** — Risk analysis by category with concentration warnings
5. **portfolio stale** — Expiring positions finder
6. **markets movers** — Biggest price changes since last sync
7. **markets heatmap** — Category activity visualization with bar charts
8. **markets correlate** — Cross-market comparison
9. **events lifecycle** — Event tracking from creation to settlement

### Deferred
- Odds history tracker (requires periodic snapshot infrastructure beyond basic sync)
- Cross-market correlation with historical price series (current version compares current state)

### Auth Changes
- Rewrote config.go: removed OAuth token flow, added RSA private key loading (PKCS1 + PKCS8)
- Rewrote client.go: implemented RSA-PSS signing with SHA256, 3-header auth
- Rewrote auth.go: setup guide instead of token save (Kalshi uses key pairs)
- Rewrote doctor.go: validates credentials via portfolio/balance API call

### Generator Limitations Found
- Generator derived slug "kalshi-trade-manual" from spec title instead of using --name flag
- Generator created OAuth-style token auth instead of Kalshi's RSA-PSS signature auth
- Config env vars were overly verbose (KALSHI_TRADE_MANUAL_KALSHI_ACCESS_KEY)
