# linear-pp-cli Build Log

## What Was Built

### P0: Foundation
- GraphQL client layer (internal/client/graphql.go) - query, mutate, paginated fetch
- GraphQL query constants (internal/client/queries.go) - 15+ queries/mutations
- Enhanced SQLite store (internal/store/store.go) - entity-specific tables, upsert methods, FTS5
- Sync command - pulls issues, projects, teams, cycles, users, labels, workflow states
- SQL command - arbitrary read-only queries

### P1: Absorbed Features (via generator + enhancements)
- 76 generated commands from GraphQL schema (issues, projects, teams, cycles, etc.)
- Issues: list, get, create, update via promoted commands
- Projects: list, get via promoted commands
- Teams: list, get via promoted commands
- Cycles: list, get via promoted commands
- Doctor: health check with auth validation
- Auth: API key configuration
- Export/Import: JSONL data transfer
- Me: current user info via GraphQL

### P2: Transcendence
- today - my issues across all teams, sorted by priority
- stale - issues not updated in N days
- bottleneck - overloaded team members analysis
- similar - FTS5 duplicate detection
- workload - issue/estimate distribution per member
- velocity - sprint completion trends across cycles

## Codex Delegation
- Store enhancement delegated to Codex gpt-5.4 (success, 0 failures)

## Intentionally Deferred
- Watch mode (real-time polling) - lower priority
- Webhooks management - lower priority
- Git branch integration - requires system git access
- Triage inbox - requires webhook setup
- Cycle comparison command - lower priority transcendence feature

## Generator Limitations Found
- GraphQL SDL parser doesn't strip pagination args from entity type fields (causes duplicate struct fields)
- Generated promoted commands reference undefined usageErr function
- Both fixed with post-generation patches
