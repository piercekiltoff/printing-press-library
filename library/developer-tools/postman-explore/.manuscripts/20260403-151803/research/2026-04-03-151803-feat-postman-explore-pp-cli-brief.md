# Postman Explore CLI Brief

## API Identity
- **Domain:** API discovery and directory browsing
- **Users:** Developers, DevRel teams, API architects, and AI agents searching for APIs to integrate
- **Data profile:** 200K+ public workspaces, 500K+ collections, organized by 20+ categories. Entities have metrics (views, forks, watches), publisher info, tags, and category associations.

## Reachability Risk
- **Low** — The undocumented proxy API at `_api/ws/proxy` has been stable across multiple runs (March 29 and March 30). No auth required for public browsing. No community reports of blocking. Rate limiting exists but is generous for normal usage.

## Top Workflows
1. **Find an API for a project** — Search by keyword ("payment", "weather", "auth"), browse by category, sort by popularity. The #1 use case.
2. **Discover trending APIs** — What's hot this week? What APIs are gaining forks and views? DevRel and tech leads use this to stay current.
3. **Evaluate an API publisher** — Look at a team's collections, see how many APIs they publish, how active they are, whether they're verified.
4. **Compare APIs in a category** — "Show me all payment APIs sorted by popularity" — category-filtered browsing with metrics.
5. **Monitor the API landscape** — Track new APIs appearing in categories over time, catch early signals of emerging platforms.

## Table Stakes
- Full-text search across collections, workspaces, APIs, flows, and teams
- Browse by entity type with sorting (popular, recent, featured, new, week, alltime)
- Filter by category
- View entity details: name, summary, description, metrics (views, forks, watches)
- View publisher/team information
- List all categories
- Pagination for all list/search endpoints

## Data Layer
- **Primary entities:** NetworkEntity (collections, workspaces, APIs, flows), Category, Team, SearchResult
- **Sync cursor:** offset-based pagination for browse; cursor-based for search
- **FTS/search:** SQLite FTS5 over entity names, summaries, descriptions, publisher names, and tags
- **Metrics snapshots:** Periodic sync captures view/fork/watch counts, enabling trend analysis over time

## Product Thesis
- **Name:** postman-explore-pp-cli
- **Why it should exist:** The world's largest API directory (200K+ workspaces, 500K+ collections) has NO programmatic access. Postman's own API can manage YOUR collections but cannot search the public API network. This CLI is the only way to search, browse, filter, and analyze the Postman API Network from the terminal or an AI agent. It turns a website-only experience into a scriptable, offline-capable, agent-native tool.

## Build Priorities
1. **Search** — Full-text search across all entity types with type filtering, pagination, and relevance scoring
2. **Browse** — Browse by entity type, sort by popularity/recency, filter by category
3. **Categories** — List and inspect categories for discovery-driven workflows
4. **Teams/Publishers** — View publisher profiles and their published APIs
5. **Stats** — Network-wide entity counts and health overview
6. **Local data layer** — SQLite sync for offline search, trend tracking, and compound queries
