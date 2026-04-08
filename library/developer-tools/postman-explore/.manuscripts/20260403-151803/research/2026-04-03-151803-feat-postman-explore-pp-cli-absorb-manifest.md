# Postman Explore CLI — Absorb Manifest

## Sources Analyzed
1. **Postman Explore website** (postman.com/explore) — The primary source, 6 proxy API endpoints
2. **Postman MCP Server** (postmanlabs/postman-mcp-server) — Official 100+ tool MCP, manages personal resources
3. **Postman Claude Skill** (SterlingChin/postman-claude-skill) — 7-phase API lifecycle skill
4. **APIs.guru** (APIs-guru/openapi-directory) — Wikipedia for Web APIs, REST API + npm package
5. **RapidAPI Hub** — 35K+ API marketplace, web-only
6. **postman-cli** (midnqp/postman-cli) — Stale minimalist collection CLI
7. **postman-sync** (thomascube/postman-sync) — Workspace-to-local sync

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Full-text search across entity types | Postman Explore search bar | `search <query> --type collection,workspace,api,flow,team` | Offline FTS5, regex, type filtering, --json |
| 2 | Browse collections by popularity | Postman Explore Browse page | `browse collections --sort popular --limit 20` | Offline cache, piped output, --compact |
| 3 | Browse workspaces | Postman Explore Workspaces page | `browse workspaces --sort popular` | Same agent-native treatment |
| 4 | Browse APIs | Postman Explore APIs page | `browse apis --sort new` | Same |
| 5 | Browse flows | Postman Explore Flows page | `browse flows --sort featured` | Same |
| 6 | Filter by category | Postman Explore category filter | `browse collections --category artificial-intelligence` | Category slug + ID resolution |
| 7 | Sort by trending (week) | Postman Explore trending page | `browse collections --sort week` | Historical trend data via sync |
| 8 | Sort by all-time popular | Postman Explore alltime sort | `browse collections --sort alltime` | Same |
| 9 | View entity details | Postman Explore entity pages | `inspect <entity-id>` | Full metrics, publisher info, tags, --json |
| 10 | View entity metrics | Postman Explore metric badges | `inspect <entity-id> --metrics` | Fork/view/watch counts, month trends |
| 11 | Publisher/team profiles | Postman Explore team pages | `teams list --sort popular` | Team metrics, collection counts |
| 12 | List all categories | Postman Explore categories page | `categories list` | Slugs, IDs, descriptions |
| 13 | Category details | Postman Explore category page | `categories show <slug>` | Hero image URL, public URL |
| 14 | Network statistics | Postman Explore (hidden) | `stats` | Collection/workspace/API/flow/team counts |
| 15 | Featured/spotlighted content | Postman Explore Spotlight | `browse collections --sort featured` | Same as sort filter |
| 16 | Search by provider | APIs.guru /providers.json | `search <query> --type team` | Publisher search with metrics |
| 17 | API directory metrics | APIs.guru /metrics.json | `stats --detail` | Richer than APIs.guru (per-type counts) |
| 18 | Open in browser | Postman Explore links | `open <entity-id>` | Opens postman.com URL in default browser |
| 19 | Pagination | Postman Explore infinite scroll | `browse --offset 40 --limit 20` | Explicit offset/limit control |
| 20 | Verified publisher badge | Postman Explore verified icon | `search --verified-only` | Filter to verified publishers |

## Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This |
|---|---------|---------|------------------------|
| 1 | Sync & offline search | `sync` / `search --offline` | SQLite FTS5 over all synced entities. Search 700K+ collections without network. |
| 2 | Trend tracking | `trending --days 7` | Periodic metric snapshots in SQLite. Shows fork/view velocity, not just absolute counts. |
| 3 | Category intelligence | `categories stats` | Cross-entity analysis: which categories are growing, where new APIs concentrate. |
| 4 | Publisher leaderboard | `teams rank --by forks` | Rank publishers across ALL their collections by aggregate metrics. |
| 5 | Similar API finder | `similar <entity-id>` | FTS5 "more like this" query across name+summary+description+tags. |
| 6 | Stale detection | `stale --days 180` | Find collections not updated in N days. Requires local updatedAt tracking. |
| 7 | Watch list & diff | `watch add <id>` / `watch diff` | Track specific entities, alert on metric changes between syncs. |
| 8 | Landscape report | `landscape --category payments` | Category-level aggregate: top publishers, avg fork rate, growth trajectory. |
| 9 | Fork velocity analysis | `velocity <entity-id>` | Month-over-month fork rate change as a leading popularity indicator. |
| 10 | Cross-category search | `search "graphql" --all-categories` | Find APIs matching a query that span multiple categories. Only works with local data. |

