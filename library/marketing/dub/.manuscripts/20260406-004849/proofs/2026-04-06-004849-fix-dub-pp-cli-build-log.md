# dub-pp-cli Build Log

## Priority 0: Foundation
- SQLite store with tables: links, tags, folders, domains, customers, partners, commissions, payouts, events, bounties, submissions, analytics, track, qr, tokens
- FTS5 indexes on: resources_fts, links_fts, folders_fts, partners_fts
- Sync workflow with incremental/full modes
- Full-text search across all synced data

## Priority 1: Absorbed Features (54)
All 54 absorbed features generated from OpenAPI spec:
- Links: list, create, get-info, get-count, update, delete, bulk-create, bulk-update, bulk-delete, upsert
- Analytics: retrieve with 10+ groupBy dimensions
- Events: list with filters
- Tags: list, create, update, delete
- Folders: list, create, update, delete
- Domains: list, create, update, delete, register, check-status
- Track: lead, sale
- Customers: list, get, update, delete
- Partners: list, create, retrieve-links, create-link, upsert-link, retrieve-analytics, ban, deactivate
- Commissions: list, update, bulk-update
- Payouts: list
- Bounties: list-submissions, approve, reject
- QR: generate
- Embed Tokens: create
- Sync, search, SQL, export, import, tail, workflow

## Priority 2: Transcendence Features (8)
1. `campaigns` — Campaign performance dashboard (tag-grouped analytics)
2. `funnel` — Attribution funnel (click→lead→sale conversion rates)
3. `tags analytics` — Tag analytics rollup
4. `customers journey` — Customer journey timeline
5. `links stale` — Stale link detector
6. `partners leaderboard` — Partner ranking
7. `domains report` — Domain utilization report
8. `links duplicates` — Duplicate link detector

## Fixes Applied
- Removed duplicate `newAnalyticsCmd` registration in root.go (would have caused panic)
- Rewrote CLI description from "Manage dub resources via the dub API" to "Manage links, analytics, domains, and partner programs via the Dub API"

## Skipped Complex Body Fields
- `geo` (object with country-level targeting) — complex nested object
- `testVariants` (A/B test config) — array of objects
- `linkProps` (bulk operations link properties) — complex nested object
- `partner` (partner creation details) — complex nested object
- `metadata` (custom key-value pairs) — dynamic object

## Generator Limitations Found
- None blocking
