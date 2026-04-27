# Hacker News CLI Brief (Regen, v2.3.9)

## Context: This Is a Reprint

Prior CLI generated 2026-04-11 with printing-press v1.3.3, scored 87/100 (Grade A). User requests full regeneration on v2.3.9 to leverage current machine capabilities. Prior research re-validated against the current machine; what changed:

- **cliutil package** — `FanoutRun` for parallel item fetches; `CleanText` for HTML extraction. Replaces ad-hoc concurrent fetching code.
- **MCP intents** — multi-step composed MCP tools (e.g., "explore_topic" = search + top items). Prior MCP was one-tool-per-endpoint; intents reduce agent token cost.
- **Scoring rubrics** — 50/50 weighted Tier 1 (16 dims, infrastructure) + Tier 2 (7 dims, domain correctness). Prior 87 was on older 16-dim rubric; need to ensure all dimensions are covered.
- **Verify-friendly RunE** — hand-written commands must validate inside RunE, fall through to `cmd.Help()` when no args, short-circuit on `dryRunOK`. Prior commands may not have followed.
- **`cliutil.IsVerifyEnv` short-circuit** — side-effect commands (`open`) must check this. Prior `open` likely doesn't.
- **Agent build checklist (9 principles)** — typed exit codes, `--select` paired with `--agent`, etc. Need to verify all are honored.
- **`pp:novel-static-reference` directive** — for curated content (none for HN).

The dual-API constraint (F3 from prior retro) **remains**: spec format supports one `base_url`. Algolia search must still be hand-built. Approach: keep spec Firebase-only; hand-build Algolia helper with `cliutil`-style guarantees.

## API Identity
- **Domain:** Developer news, discussions, job postings, community content
- **Users:** Software developers, tech leads, startup founders, anyone reading HN daily; agents that need to monitor tech trends
- **Data profile:** Dual-API — **Firebase** (real-time items, stories, users, no rate limit) + **Algolia** (full-text search, date filtering, tags, public)

## Reachability Risk
**None.** Both APIs are public, free, no auth, no rate limits (Firebase) / generous (Algolia). Same reachability as prior run, no GitHub issues reporting blocks since 2026-04.

## Top Workflows
1. **Morning scan** — `top`, `best` lists; agents pulling daily HN signal
2. **Topic pulse** — "what is HN saying about Rust this week?" (Algolia date+tag filter)
3. **Search past discussions** — "has X been posted before?" / "what happened when Stripe shipped Y?"
4. **Who's Hiring scan** — monthly thread, regex filter for Go/remote/SF
5. **Deep thread reading** — flatten or tree the comments without web UI nesting

## Data Sources

### Firebase (real-time, primary, in spec)
- Base: `https://hacker-news.firebaseio.com/v0`
- `/topstories.json` (500 IDs), `/newstories.json` (500), `/beststories.json` (200), `/askstories.json` (200), `/showstories.json` (200), `/jobstories.json` (200)
- `/item/{id}.json` — story/comment/job/poll, recursive via `kids`
- `/user/{id}.json` — karma, about, submitted IDs
- `/maxitem.json`, `/updates.json`

### Algolia (search, hand-built helper)
- Base: `https://hn.algolia.com/api/v1`
- `/search` — relevance-sorted; `/search_by_date` — date-sorted
- Tags: `story`, `comment`, `ask_hn`, `show_hn`, `job`, `poll`, `author_<name>`
- numericFilters: `created_at_i`, `points`, `num_comments`
- `/items/{id}` — full comment tree fetched in one shot (faster than recursive Firebase walks)

### Why both?
- Firebase: fresh ranks; Algolia is delayed by minutes
- Algolia: search/filter; Firebase doesn't expose query

## Codebase Intelligence
- Source: prior research + DeepWiki not re-run (same APIs as v1.3.3)
- Auth: none; no headers needed
- Data model: `Item.kids` is a tree; `Item.parent`, `Item.text`, `Item.dead`, `Item.deleted`. Algolia returns flat hits with `_tags` for type, `created_at_i` for sort.
- Rate limiting: Firebase none; Algolia ~1000 req/hour (more than enough for CLI)
- Architecture: Firebase = read-mostly REST; Algolia = SaaS search index updated from Firebase

## Data Layer
- **Primary entities:** stories, comments, users, jobs (all the same `Item` type with different `type` field)
- **Sync:** `top/best/new` story IDs on `sync`; per-item write-through cache for `item`/`comments`
- **FTS:** SQLite FTS5 on title + comment text + user about. Compound: "show me all comments by user X mentioning Y" runs offline.
- **Cursor:** maxitem ID + per-list refresh timestamp

## Product Thesis
- **Name:** `hackernews-pp-cli` (binary)
- **Why it should exist:** Every HN CLI is either TUI-only (circumflex — beautiful, no scripting), unmaintained Python (haxor-news, 2015), or a one-tool-per-endpoint MCP that forces agents to stitch primitives. None combine Firebase real-time + Algolia search, none have agent-native `--json --select` output, none have local SQLite FTS, none expose compound MCP intents (e.g., "explore topic" = search + ranked items + comment summary in one call). A modern Go CLI with all of these would be the first HN tool that's genuinely useful to both humans piping into `jq` and agents reaching for an HN tool.

## Build Priorities
1. **Foundation:** SQLite store, Firebase client, hand-built Algolia helper (replayable HTTP only — no browser), cliutil already emitted
2. **Absorb (P1):** All competing CLI/MCP features — top/new/best/ask/show/jobs lists, item, comments tree, user, search, hiring, freelance, open, updates, bookmarks
3. **Transcend (P2):** Front-page diff (`since`), topic pulse, hiring stats, story velocity, controversial sort, repost finder, posting timing, thread tldr, my-submissions tracker, karma trend
4. **MCP intents:** `hn_explore_topic`, `hn_thread_summary`, `hn_hiring_match` (compound tools, not endpoint mirrors)

## Source Priority
N/A — single source (Hacker News).
