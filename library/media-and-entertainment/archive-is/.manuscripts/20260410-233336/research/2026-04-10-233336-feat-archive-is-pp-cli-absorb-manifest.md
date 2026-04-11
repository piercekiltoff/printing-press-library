# Archive.is CLI — Absorb Manifest

## Source Tools Cataloged

| Tool | Language | Stars | What it does | Status |
|------|----------|-------|--------------|--------|
| palewire/archiveis | Python | ~150 | Reference archive.today client. Submit-only, uses stale submitid flow but still works. | Maintained but quiet |
| HRDepartment/archivetoday | Node.js | ~80 | Current npm package. snapshot + timemap, renew flag, quiet mode. | Active |
| oduwsdl/archivenow | Python | ~400 | Academic multi-archive pusher. Uses Selenium for archive.today. | Active, overkill |
| docs.rs/archiveis | Rust | ~10 | Rust port of palewire. Low usage. | Stale |
| Internet Archive wayback API | JSON API | n/a | Availability API + SPN2 capture API. Clean JSON, no auth. | Stable |
| Memento Protocol (RFC 7089) | Standard | n/a | timegate/timemap standard that archive.today implements | Standard |

**Total absorbable features: 18**

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Submit URL to create archive | palewire/archiveis | `archive-is save <url>` | Go binary, proper User-Agent, 180s timeout, typed exit codes |
| 2 | Force fresh capture (re-archive) | HRDepartment `--renew` | `archive-is save <url> --force` | `&anyway=1` param, clear messaging about dedup window |
| 3 | Lookup existing snapshot (timegate) | HRDepartment snapshot() | `archive-is read <url>` | Timegate-first, falls back to submit only on miss (saves 60+ sec) |
| 4 | List all snapshots (timemap) | HRDepartment timemap | `archive-is history <url>` | Parses Memento link-format, pretty-prints chronologically with --json |
| 5 | Get most recent snapshot | archive.ph/newest/ path | Folded into `read` command | One command instead of two |
| 6 | Quiet mode (URL only) | HRDepartment `--quiet` | `--quiet` flag on all commands | Standard, composable with pipes |
| 7 | Non-blocking submit | HRDepartment `--incomplete` | `archive-is save --async` | Returns submit URL immediately, doesn't wait for capture |
| 8 | JSON output | None of them | `--json` on every command | Agent-native, pipes to jq |
| 9 | Wayback fallback | archivenow multi-archive | `archive-is read --backend wayback` | Uses archive.org/wayback/available JSON API |
| 10 | Wayback submit | archivenow SPN2 | `archive-is save --backend wayback` | Uses Wayback SPN2 API when archive.is is down |
| 11 | Domain search | archive.ph/<domain>/ path | `archive-is history --domain` | Scrapes /`<domain>` page, returns all archives for that domain |
| 12 | Wildcard search | archive.ph/*.domain.com path | Folded into `history --domain` | Supports wildcard subdomain match |
| 13 | Global RSS feed | archive.ph/rss | `archive-is recent` | Parses RSS, shows recent archives globally |
| 14 | Search by query | archive.ph/search/?q= | `archive-is remote-search <query>` | HTML scrape the search page, extract URL+date |
| 15 | Typed exit codes | None of them | 0/2/3/4/5/7 in helpers.go | Standard Printing Press pattern |
| 16 | Doctor command | None of them | `archive-is doctor` | Checks all 6 mirrors, reports which are reachable |
| 17 | Local cache (SQLite) | None of them | `archive-is sync`, `sql`, `search` | Full data layer, offline lookup |
| 18 | MCP server | None of them | `archive-is-mcp` | First MCP for archive.is |

## Transcendence (only possible with our local data layer + dual-backend + compound reasoning)

| # | Feature | Command | Score | Why Only We Can Do This |
|---|---------|---------|-------|------------------------|
| 1 | **Read article as markdown** | `archive-is get <url>` | 10/10 | Fetches memento HTML, extracts clean markdown via readability heuristic. No competitor extracts text. This is the killer feature for LLM piping. |
| 2 | **Mirror auto-fallback** | transparent, triggered on 429/timeout | 9/10 | Retries across archive.ph → .md → .is → .fo → .li → .vn. No competitor does this. Survives ISP blocks and single-mirror rate limits. |
| 3 | **Lookup-before-submit** | `archive-is read` default behavior | 10/10 | Timegate first, submit only on miss. Saves 60+ seconds per hit and avoids 429. No competitor does this by default. |
| 4 | **Dual-backend fallback** | `--backend archive-is,wayback` | 8/10 | On archive.is failure, automatically tries Wayback. Hedges the Feb 2026 reputation risk and general availability. |
| 5 | **Local FTS5 search** | `archive-is search "fed rate cut"` | 7/10 | Search across extracted article text from all your past archives. Find that NYT article from months ago without remembering the URL. |
| 6 | **Bulk archive with rate-limit awareness** | `archive-is bulk <file>` | 7/10 | Reads URLs from file or stdin, 10-second gaps, exponential backoff on 429, writes results to SQLite. Survives rate limits without user babysitting. |
| 7 | **Copy-to-clipboard integration** | `archive-is read <url>` default behavior | 6/10 | Unix: pbcopy/xclip. Prints URL to stdout AND copies. The paywall-reader's muscle memory match. |
| 8 | **Reading queue** | `archive-is queue add <url>`, `queue read` | 6/10 | Add paywalled articles to a queue, batch-process them later. Useful for research workflows. |

**All 8 transcendence features will be built.**

## What to build first (Priority Order)

### Priority 0: Foundation
- SQLite data layer with `archives` and `bodies` tables
- HTTP client with mirror fallback transport
- Typed exit codes, doctor command
- Sync/search/sql machinery

### Priority 1: Absorbed features (all 18)
- submit, save, read, history, recent, remote-search, doctor, --json, --quiet, --async, Wayback fallback, etc.

### Priority 2: Transcendence (all 8 GOAT features)
- `get` (markdown extraction) — hero feature
- Mirror auto-fallback in transport layer
- Lookup-before-submit in `read` command
- Local search via FTS5
- Bulk with rate limiting
- Clipboard integration
- Reading queue

## Gap Analysis vs Best Competitor (HRDepartment/archivetoday)

HRDepartment has: snapshot, timemap, renew, quiet, incomplete = 5 features

Our CLI will have: 18 absorbed + 8 transcendence = **26 features**

**5.2x the feature count** of the best existing tool, with agent-native design, single-binary install, offline cache, dual backend, and mirror fallback. And it's the only Go implementation and the only MCP server.
