# Archive.is CLI Brief

## API Identity

- **Service:** archive.today (aka archive.is, archive.ph, archive.md, archive.fo, archive.li, archive.vn — all the same backend, different TLDs to dodge ISP blocks)
- **Domain:** Web page archiving service, primary use case is paywall bypass
- **Users:** Journalists, researchers, fact-checkers, paywalled-news readers, Wikipedia editors (until Feb 2026), Bogleheads and Threads power users sharing articles
- **Data profile:** ~500M archived pages, ~700TB as of 2021. The service captures the rendered DOM of a page, strips JS, and serves the result as a static HTML snapshot. This is why it bypasses paywalls — many paywalls are JS overlays that the archiver skips.

## Reachability Risk

**Medium.** Not a blocker, but not free.

- No Cloudflare CAPTCHA on fresh IPs using the OS DNS resolver
- The well-known "archive.is CAPTCHA loop" affects users on Cloudflare DNS (1.1.1.1) only — resolver issue, not HTTP-level. Our CLI will use the OS resolver.
- Rate limit is IP-behavior based, ~429 HTTP after heavy submit activity, with ~1 hour cooldown
- `ArchiveTeam` wiki confirms: "IP quota unknown, server stops responding after excessive archiving"
- Must use a realistic User-Agent (not Go's default `Go-http-client/1.1`)
- No CAPTCHA/JS challenge on timegate or timemap endpoints — those are cheap lookups, safe to use freely
- Evidence: `palewire/archiveis` issue #32 (429, 2021 still open), `oduwsdl/archivenow` issue #53 (captcha for academic batch use)

## CRITICAL: Reputation Warning (Feb 2026)

Wikipedia formally blacklisted archive.today on Feb 21, 2026 after an RfC with 200+ participants. Triggers:
- Jan 2026: archive.today caught embedding DDoS JS targeting security blogger Jani Patokallio
- Feb 2026: archive.today caught tampering with third-party snapshots to modify content
- Coverage: Tom's Hardware, TechCrunch, TechRadar, Boing Boing, TechSpot

**Implications for this CLI:**
- Paywall reading (primary use case): unaffected. Article text still accurate for 99%+ of captures.
- Evidence preservation / legal citation: dead. Use Wayback Machine instead.
- README must explicitly disclaim that this is for personal paywall reading, NOT legal evidence.
- We mitigate by building Wayback Machine fallback from day one.

## Top Workflows (ranked by paywall-reader demand)

1. **"I hit a paywall — give me the readable version"** — the 80% use case. Command: `archive-is read <url>`. Implementation: timegate lookup first (fast, no rate limit), fall back to submit only on miss. Copy result to clipboard for pasting.

2. **"Get the full article text, not just the URL"** — feeds directly into LLMs, notes, Claude context windows. Command: `archive-is get <url>`. Fetches the memento HTML and extracts clean markdown. No existing CLI does this well.

3. **"When was this URL first/last archived?"** — researchers tracking article edits and takedowns. Command: `archive-is history <url>`. Uses the timemap endpoint which returns Memento link-format with all snapshots.

4. **"Force a fresh capture"** — when the existing snapshot is stale or you want to preserve the current version. Command: `archive-is save <url>` with `--anyway=1` flag to bypass the dedup window.

5. **"Bulk-save a list of URLs"** — archive a reading list, research sources, or a link dump. Command: `archive-is bulk <file>` with built-in 10-second gaps to avoid 429.

6. **"Search my local archive history"** — once we've cached lookups, search by keyword. Command: `archive-is search <query>`.

## Table Stakes (competitor features to absorb)

From `palewire/archiveis` (Python, canonical reference):
- Submit URL to create archive
- Return the memento URL

From `HRDepartment/archivetoday` (Node.js, closest competitor):
- `snapshot(url)` — submit
- `timemap(url)` — list all snapshots
- `--renew` / `anyway=1` flag for forced re-capture
- `--quiet` for URL-only output
- `--incomplete` for non-blocking submit

From `oduwsdl/archivenow` (Python, multi-archive):
- Push to multiple archives at once
- Selenium-based browser automation (we skip this; net/http is enough)

From `internetarchive/wayback` availability API:
- `archive.org/wayback/available?url=...&timestamp=...`
- Returns JSON with closest snapshot
- No auth required

## Data Layer

Primary entities (ALL belong in SQLite):

| Table | Purpose | Key fields |
|-------|---------|------------|
| `archives` | Lookup cache + history | original_url (canonical), memento_url, captured_at, fetched_at, status, mirror, backend |
| `bodies` | Extracted article text cache (opt-in) | archive_id, title, author, published_at, markdown_content |
| `domains` | Per-domain metadata | domain, first_seen, last_seen, archive_count |

Canonical URL normalization: strip UTM params, sort query string, lowercase host. This prevents duplicate lookups for "same" URL.

FTS5 search across `bodies.markdown_content` enables: "find that NYT article about Fed rate cuts I archived last month."

## User Vision

**"I want to be able to see any article in the world that is behind a paywall."**

This is the single organizing principle. Every feature earns its place by serving this goal. Secondary features (history, search, bulk) are enrichment, not the core. The `read` command is the hero.

## Product Thesis

**Name:** `archive-is-pp-cli` (binary), published as `archive-is` in the library

**Why it should exist:**

1. No Go implementation exists. palewire/archiveis is Python (runtime-bound, stale). HRDepartment/archivetoday is Node (runtime-bound). A single-binary Go CLI installs with one command on any platform.

2. No existing CLI does lookup-before-submit. Every competitor submits on every call, wasting 60+ seconds per hit and triggering 429s. Our `read` command hits timegate first (500ms), only submits if there's no recent snapshot. This is the killer UX.

3. No existing CLI extracts clean article text. Every competitor returns the archive URL and stops. Our `get` command fetches the memento HTML and returns markdown, ready to pipe into `claude`, `llm`, `pandoc`, or a note app.

4. No MCP server exists for archive.is as of April 2026. We ship one. Claude and other agents get native paywall bypass.

5. No competitor has mirror fallback. We auto-fail over between archive.ph → archive.md → archive.is → archive.fo when one is blocked.

6. No competitor has a Wayback backend. Given the Feb 2026 reputation risk, a dual-backend CLI hedges the user's bet.

## Build Priorities

1. **Data layer + archives table** — foundational, all commands write here
2. **`read <url>`** — timegate-first find-or-create, the hero command
3. **`get <url>`** — fetch memento HTML, extract clean markdown text
4. **`history <url>`** — parse timemap link-format, return chronological snapshot list
5. **`save <url>`** — force fresh capture with `--anyway=1`
6. **`bulk <file>`** — rate-limited batch, reads stdin or file
7. **`search <query>`** — FTS5 over local archives cache
8. **Mirror fallback** — transport wrapper that retries on .ph → .md → .is → .fo
9. **Wayback backend** — `--backend wayback` flag for Internet Archive fallback
10. **MCP server** — expose `read`, `get`, `history` as MCP tools

## Sources & References

- palewire/archiveis: https://github.com/palewire/archiveis
- HRDepartment/archivetoday: https://github.com/HRDepartment/archivetoday
- oduwsdl/archivenow: https://github.com/oduwsdl/archivenow
- ArchiveTeam wiki: https://wiki.archiveteam.org/index.php/Archive.today
- Wayback availability API: https://archive.org/wayback/available?url=...
- Memento Protocol RFC 7089: https://datatracker.ietf.org/doc/html/rfc7089
- Feb 2026 Wikipedia blacklist: Tom's Hardware, TechCrunch coverage
- Live verification: timegate and timemap endpoints tested against archive.ph during research
