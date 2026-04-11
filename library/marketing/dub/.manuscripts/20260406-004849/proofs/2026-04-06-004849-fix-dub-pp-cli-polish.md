# dub-pp-cli Polish Report

## Verify Delta
- Before: 100% (22/22 PASS)
- After: 100% (22/22 PASS)
- No regression

## Scorecard Delta
- Before: 96/100 Grade A
- After: 93/100 Grade A
- Data Pipeline Integrity dropped 10→7 (likely due to store modifications)

## Fixes Applied During Live Testing
1. **retryAfter cap** — Retry-After header with far-future HTTP date caused ~1.2M hour wait. Capped to 60s max.
2. **FTS search quoting** — Terms with hyphens (pp-test) caused FTS5 parse errors. All search methods now use ftsQuote().
3. **UpsertBatch FTS population** — UpsertBatch (used by sync) wasn't updating resources_fts index. Added FTS insert after each upsert.
4. **Search routing** — Search command was querying per-table FTS indexes (links_fts, folders_fts) that were never populated. Changed to use resources_fts which IS populated during sync.

## Printing Press Issues (for retro)
1. UpsertBatch in store template should populate resources_fts — same pattern as Upsert()
2. Search command template should default to resources_fts, not per-table FTS indexes that may not be populated

## Ship Recommendation: ship
