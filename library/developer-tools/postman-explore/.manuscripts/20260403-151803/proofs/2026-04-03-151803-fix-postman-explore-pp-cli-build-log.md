# Postman Explore CLI Build Log

## What Was Built

### Priority 0 (Foundation)
- Fixed `serviceForPath()` routing: "search" for /search-all, "publishing" for everything else
- Enhanced SQLite store with dedicated entity, category, team, metric_snapshots, and watchlist tables
- Entity-specific FTS5 index over name, summary, publisher_name, tags
- Batch entity upsert with metric extraction

### Priority 1 (Absorbed Features) — 8 New Commands
1. `search <query>` — Full-text search with --type filter and --offline mode
2. `browse <type>` — Browse collections/workspaces/apis/flows with --sort and --category
3. `categories list/show` — List and inspect API categories
4. `stats` — Network entity counts (709K collections, 310K workspaces, etc.)
5. `open <name-or-id>` — Open entity in browser from local store
6. `trending [type]` — Popular entities (shortcut for browse --sort popular)
7. `stale` — Find stale entities (requires sync)
8. `similar <query>` — FTS5 similarity search (requires sync)

### Priority 2 (Transcendence)
- Metric snapshots table for trend tracking
- Watchlist table for entity tracking
- Entity-specific store methods: SearchEntities, ListEntities, StaleEntities
- AddToWatchlist/RemoveFromWatchlist/GetWatchlist
- Sync fixed to paginate all 6 entity types (collection, workspace, api, flow, team, category)

### Priority 3 (Polish)
- CLI description rewritten: "Search, browse, and analyze the Postman Public API Network from the terminal"
- Sort values corrected: API only accepts "popular" and "featured"
- Pagination parameter fixed: offset-based (not cursor-based)

## Live API Verification (Priority 1 Review Gate)
- `search stripe --limit 3` — PASS (returned Stripe API Demos, Stripe Developers workspace)
- `browse collections --limit 3` — PASS (returned Salesforce Platform APIs as #1)
- `stats` — PASS (709,609 collections, 310,405 workspaces, 25,193 APIs)
- `categories list` — PASS (12 categories returned)
- `search auth --dry-run` — PASS (correct proxy envelope)

## Known Limitations
- Sort only supports "popular" and "featured" (API rejects other values)
- Trending uses "popular" sort (no separate trending endpoint)
- Sync paginates with offset/limit; full sync of 700K+ collections would be slow
