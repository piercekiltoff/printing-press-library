# Hacker News CLI Absorb Manifest (Regen)

## Sources Analyzed (re-validated from prior run)
1. **circumflex** (Go, TUI) — story browsing, comment reading, reader mode, favorites, vim nav
2. **haxor-news** (Python) — story lists, search, who's hiring regex, comment filtering, user profiles
3. **hnterminal** (Python) — browsing, login, upvote, commenting (auth-only, skipping)
4. **mcp-hacker-news** (TypeScript MCP) — 11 tools: top/best/new/ask/show/job, item, user, comments, maxitem, updates
5. **hackernews-mcp** (Python MCP) — article content extraction + discussions
6. **hn-mcp** — Firebase + Algolia integration
7. **hackernews-api** (npm) — Firebase wrapper
8. **hacker-news-api** (npm) — Algolia wrapper
9. **hnclient** (PyPI) — cached API calls

## Absorbed (P1: match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | Top stories | All | `hackernews stories top` | --limit, --json, --select, parallel item fetch via cliutil.FanoutRun |
| 2 | New stories | All | `hackernews stories new` | Same flags |
| 3 | Best stories | circumflex, MCPs | `hackernews stories best` | Same flags |
| 4 | Ask HN | haxor-news, MCPs | `hackernews ask list` | Same flags |
| 5 | Show HN | haxor-news, MCPs | `hackernews show list` | Same flags |
| 6 | Job stories | haxor-news, MCPs | `hackernews jobs list` | Same flags |
| 7 | Item details | All | `hackernews stories get <id>` | Full details, parent chain, --json, --select |
| 8 | Read comments tree | circumflex, haxor-news | `hackernews comments <id>` | Threaded display, --flat, --depth, --json. Uses Algolia `/items/{id}` for one-shot fetch |
| 9 | Search stories | haxor-news (Algolia) | `hackernews search <query>` | Full Algolia params: --tag, --since, --until, --min-points, --sort, --json |
| 10 | User profile | haxor-news, MCPs | `hackernews users get <name>` | Karma, about, submission preview, --json |
| 11 | Who's Hiring | haxor-news | `hackernews hiring <regex>` | Filter latest monthly thread by tech/location/remote |
| 12 | Freelance thread | haxor-news | `hackernews freelance <regex>` | Same pattern |
| 13 | Open in browser | haxor-news | `hackernews open <id>` | Side-effect convention: print by default, --launch to open. cliutil.IsVerifyEnv guard. |
| 14 | Recently updated | MCP servers | `hackernews updates` | Changed items + profiles, --json |
| 15 | Bookmarks (local) | circumflex | `hackernews bookmark add/list/rm` | Local SQLite, not tied to HN account |
| 16 | Comment filtering | haxor-news | `--author`, `--since`, `--match` flags on `comments` | Regex + time + author |

## Transcendence (P2: only possible with our approach — reprint reconciliation)

Each prior feature re-scored on current rubric (1–10) against current personas: agent users, daily-HN power users, hiring scanners.

| # | Feature | Command | Score | Action | Notes |
|---|---------|---------|-------|--------|-------|
| 1 | Front page diff | `hackernews since` | 9/10 | **KEEP** | Stories that appeared/disappeared/moved since last check. Requires SQLite snapshots. Unique to local-store CLIs. |
| 2 | Thread tldr | `hackernews tldr <id>` | 8/10 | **REFRAME** | Was "summarize comments". Reframe: structured digest (top voices by author, root vs reply ratio, thread heat metric). Avoid AI-summary placeholder; ship deterministic stats. |
| 3 | Submission tracker | `hackernews my <username>` | 8/10 | **KEEP** | Per-user submission history with score buckets, best-time analysis. |
| 4 | Topic pulse | `hackernews pulse <topic>` | 9/10 | **KEEP** | Algolia date+tag aggregation. "What's HN saying about X this week?" Strong agent fit. |
| 5 | Who's Hiring stats | `hackernews hiring-stats` | 8/10 | **KEEP** | Aggregate across N months: top languages, remote %, company freq. Uses Algolia author tag. |
| 6 | Story velocity | `hackernews velocity <id>` | 6/10 | **REFRAME** | Was points/hr live tracker. Reframe: shows current rank trajectory from local snapshots taken via `since`. No polling required. |
| 7 | User karma graph | `hackernews karma <username>` | 6/10 | **DROP** | Karma is a single number on the user record; "trend" requires per-day snapshots that we don't take. Synthesizing it from submission scores is misleading (karma includes comment upvotes too). Anti-pattern: re-implementing the API. |
| 8 | Controversial | `hackernews controversial` | 7/10 | **KEEP** | Sort synced stories by comments/points ratio. Pure local SQL. |
| 9 | Repost finder | `hackernews repost <url>` | 8/10 | **KEEP** | Algolia URL search; show prior submissions with scores. Useful before posting. |
| 10 | Posting timing | `hackernews timing` | 5/10 | **DROP** | "Best time to post" by day-of-week × hour aggregation. Requires bulk Algolia scrape that's not in the workflows; cute but rarely actionable. Replaced by simpler heuristics in `my`. |

### New (introduced for this regen, scored against current rubric)

| # | Feature | Command | Score | Why |
|---|---------|---------|-------|------|
| 11 | Sync command | `hackernews sync` | 9/10 | Foundation for all transcendence. Pulls top/best/new + write-through items. Required for offline FTS. |
| 12 | FTS search across local store | `hackernews local-search <query>` | 8/10 | Offline search across synced stories AND comments. Algolia stops at ~7 days; local store keeps everything you've touched. |
| 13 | Doctor | `hackernews doctor` | 6/10 | Standard health check (Firebase reachable, Algolia reachable, DB writable). |

### Reprint reconciliation summary
- **Kept verbatim:** since, my, pulse, hiring-stats, controversial, repost (6)
- **Reframed:** tldr (deterministic stats, not AI), velocity (snapshot-based, not polling) (2)
- **Dropped:** karma (re-implementation risk), timing (low-actionability) (2)
- **New:** sync, local-search, doctor (3)

Net P2: 11 transcendence commands. Down from prior 10 named transcendence + 2 dropped + 3 new.

## MCP Intents (new for this regen)

The current machine supports composed MCP intent tools. These reduce agent token cost vs. one-tool-per-endpoint.

| # | Intent | Composes | Returns |
|---|--------|----------|---------|
| 1 | `hn_explore_topic` | search → top 5 hits → fetch each item with comment count | Story list + scoring metadata |
| 2 | `hn_who_is_hiring` | latest "Ask HN: Who is hiring" thread → filter regex → top matches | Filtered job posts |

Endpoint-mirror tools stay visible (small API). MCP transport: stdio + http (for cloud agents).

## Spec Strategy

Spec contains **Firebase only**. Search is hand-built (Algolia base URL is different — single-base-URL spec constraint, same as prior run). Search becomes a transcendence/novel-feature command, not a spec-driven endpoint. This avoids the prior bug where the spec generated `/search` against Firebase base URL.

`internal/algolia/` package houses the Algolia client. Keeps it isolated from generated code.

## Stub Disclosure

No features will ship as stubs. Everything in this manifest is shipping scope.

## Anti-reimplementation kill check

Per `skills/printing-press/references/absorb-scoring.md`:
- ✓ Every command above either calls Firebase, calls Algolia, reads from SQLite store, or composes both
- ✓ No command returns hardcoded data, canned responses, or in-process aggregations of constants
- ✓ `controversial`, `since`, `my` read from local store populated by `sync` (legitimate carve-out)
- ✓ `local-search` reads from FTS5 (legitimate carve-out)
- ✓ No `pp:novel-static-reference` directives needed (no curated content)
