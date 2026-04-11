# Hacker News CLI Brief

## API Identity
- Domain: Developer news, discussions, job postings, community content
- Users: Software developers, tech enthusiasts, startup founders, anyone who reads HN daily
- Data profile: Dual-API — Firebase (real-time items, stories, users) + Algolia (full-text search, date filtering, tags)

## User Vision
High bar for novel/transcendent features. Developer audience will scrutinize every command. Must be genuinely more useful than just opening news.ycombinator.com.

## Data Sources

### Firebase API (real-time, primary)
- Base URL: https://hacker-news.firebaseio.com/v0
- Auth: None. No rate limits.
- Endpoints:
  - /topstories.json — up to 500 IDs
  - /newstories.json — up to 500 IDs
  - /beststories.json — up to 200 IDs
  - /askstories.json — up to 200 IDs
  - /showstories.json — up to 200 IDs
  - /jobstories.json — up to 200 IDs
  - /item/{id}.json — story, comment, job, poll details
  - /user/{id}.json — user profile (karma, submitted items)
  - /maxitem.json — current max item ID
  - /updates.json — recently changed items and profiles

### Algolia Search API (search + filtering)
- Base URL: https://hn.algolia.com/api/v1
- Auth: None (public API key embedded).
- Endpoints:
  - /search?query=...&tags=...&numericFilters=... — relevance-sorted
  - /search_by_date?query=...&tags=... — date-sorted
  - /items/{id} — item details with full comment tree
  - /users/{username} — user details
- Tags: story, comment, ask_hn, show_hn, job, poll, author_{username}
- Filters: created_at_i (timestamp), points, num_comments

## Reachability Risk
- **None** — Both APIs public, free, no auth, no rate limits (Firebase), generous limits (Algolia).

## Top Workflows
1. **Morning scan**: Check top/best stories, open interesting ones — the daily HN ritual
2. **Search past discussions**: "Was this already discussed on HN?" — search by keyword, date range
3. **Who's Hiring**: Browse monthly job threads, filter by tech/location/remote
4. **Deep thread reading**: Read comment threads without the web UI's nesting pain
5. **Track a user**: See someone's submissions and karma over time

## Table Stakes (from competitors)
- Top/new/best/ask/show/job story lists (all CLIs, all MCP servers)
- View item details (all CLIs)
- Read comment threads (circumflex, haxor-news, hnterminal)
- Search stories (haxor-news via Algolia)
- Who's Hiring filter with regex (haxor-news)
- User profile lookup (haxor-news, MCP servers)
- Favorites/bookmarks (circumflex)
- Reader mode for articles (circumflex)
- Open in browser (haxor-news)
- Category filtering — top/best/new/ask/show/jobs (all CLIs)
- Comment filtering — unseen, recent, regex (haxor-news)
- Article content extraction (hackernews-mcp via Firecrawl)
- Recently updated items (MCP servers)
- --json output (none of the existing CLIs have this!)

## Data Layer
- Primary entities: stories, comments, users, jobs
- Sync: top/best/new stories on `sync`, individual items cached via write-through
- FTS: on story titles, comment text, user names for instant offline search

## Product Thesis
- Name: hackernews-pp-cli (hn-pp-cli binary)
- Why it should exist: Every HN CLI is either a TUI (circumflex — beautiful but no scripting/piping) or a Python tool from 2015 (haxor-news — unmaintained). None have --json output, agent-native flags, or local SQLite persistence. None combine Firebase real-time data with Algolia search. None do compound analytics ("show me the most discussed topics this week" or "which of my submissions got traction?"). A modern Go CLI with both APIs, local search, and agent-native output would be the first HN tool that's both human-friendly AND machine-friendly.

## Build Priorities
1. Dual-API integration — Firebase for story lists/items, Algolia for search/filtering
2. Core commands — top, new, best, ask, show, jobs, item, user, search
3. Who's Hiring with smart filtering
4. Local SQLite with write-through — every lookup grows the searchable corpus
5. Transcendence features — thread analytics, topic trends, submission tracker
