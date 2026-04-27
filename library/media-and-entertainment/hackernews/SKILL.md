---
name: pp-hackernews
description: "Hacker News from your terminal — with a local store, full-text search, and agent-native output no other HN tool has. Trigger phrases: `check hacker news`, `search hn`, `what is hn saying about`, `look up hn user`, `hn top stories`, `use hackernews`."
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata: '{"openclaw":{"requires":{"bins":["hackernews-pp-cli"]},"install":[{"id":"go","kind":"shell","command":"go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/hackernews/cmd/hackernews-pp-cli@latest","bins":["hackernews-pp-cli"],"label":"Install via go install"}]}}'
---

# Hacker News — Printing Press CLI

Combines the Firebase real-time API and the Algolia search API in one CLI. Sync once and run searches, diffs, and topic pulses against a local SQLite store — offline, scriptable, and agent-friendly. Every command supports --json and --select; mutations don't apply because Hacker News is read-only.

## When to Use This CLI

Reach for hackernews-pp-cli when you need to monitor or analyze HN signal programmatically: tracking topics, finding reposts before submitting, scanning Who's Hiring, or pulling thread digests into agent context. The local store makes follow-up queries cheap.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Local state that compounds
- **`since`** — Show what changed on the front page since last check — stories that appeared, disappeared, or moved.

  _Agents tracking HN signal need delta-mode, not full re-fetch._

  ```bash
  hackernews-pp-cli since --json
  ```
- **`controversial`** — Find stories with the highest comment-to-point ratio — the polarizing discussions.

  _Surfaces dissent, not just consensus, which the homepage hides._

  ```bash
  hackernews-pp-cli controversial --limit 10 --json
  ```
- **`velocity`** — Show a story's rank trajectory from local snapshots (climb, fall, stalled).

  _Agents asking 'is this gaining traction' get a trend, not a moment-in-time score._

  ```bash
  hackernews-pp-cli velocity 12345678 --json
  ```
- **`local-search`** — Offline FTS5 search across every story and comment you've touched.

  _Agents replaying past investigations don't re-hit Algolia._

  ```bash
  hackernews-pp-cli local-search "open source ai" --select title,url,score
  ```
- **`sync`** — Pull top/best/new lists into local SQLite for offline use and snapshot history.

  _First run makes the rest cheap; agents call once and read locally._

  ```bash
  hackernews-pp-cli sync --full
  ```

### Compound queries
- **`pulse`** — What HN is saying about a topic this week — score, comment, frequency by day.

  _One call replaces N Algolia paginations and the math an agent would otherwise do._

  ```bash
  hackernews-pp-cli pulse rust --days 7 --agent
  ```
- **`my`** — Track a user's submissions with score buckets, traction rate, and best posting time hints.

  _Replaces manual per-id fetches when an agent profiles a contributor._

  ```bash
  hackernews-pp-cli my pg --agent
  ```
- **`hiring-stats`** — Aggregate Who's Hiring across recent months: languages, remote ratio, top companies.

  _Agents matching jobs to a profile get the breakdown without scraping the threads themselves._

  ```bash
  hackernews-pp-cli hiring-stats --months 3 --agent --select languages
  ```
- **`repost`** — Has this URL been posted on HN before? Lists prior submissions with scores and dates.

  _Pre-flight check before posting; avoids dupe submissions._

  ```bash
  hackernews-pp-cli repost https://example.com/article
  ```

### Agent-native plumbing
- **`tldr`** — Deterministic thread digest: top authors by reply count, root vs reply ratio, comment heat metric.

  _Agents skimming a 500-comment thread get measurable signals, not opinion._

  ```bash
  hackernews-pp-cli tldr 12345678 --agent
  ```

## Command Reference

**ask** — Browse Ask HN questions

- `hackernews-pp-cli ask` — Get the latest Ask HN posts

**jobs** — Browse Hacker News job postings

- `hackernews-pp-cli jobs` — Get the latest Hacker News job postings

**maxitem** — Current maximum item ID

- `hackernews-pp-cli maxitem` — Returns the largest item ID currently assigned by Hacker News

**show** — Browse Show HN launches

- `hackernews-pp-cli show` — Get the latest Show HN posts

**stories** — Browse top, new, and best Hacker News stories

- `hackernews-pp-cli stories best` — Get the highest-voted stories on Hacker News
- `hackernews-pp-cli stories get` — Get details for a specific story, comment, job, or poll
- `hackernews-pp-cli stories new` — Get the newest stories on Hacker News
- `hackernews-pp-cli stories top` — Get the current top stories on Hacker News

**updates** — Recently changed items and profiles

- `hackernews-pp-cli updates` — Items and user profiles that have changed recently

**users** — Look up Hacker News user profiles

- `hackernews-pp-cli users <userId>` — Get a user's profile including karma and submission history


**Hand-written commands**

- `hackernews-pp-cli sync` — Pull top, best, and new story lists into the local SQLite store for offline use
- `hackernews-pp-cli search <query>` — Hybrid search: local FTS5 with live fallback in `--data-source auto` mode (use `--data-source local` to force offline, `--data-source live` to bypass the store)
- `hackernews-pp-cli live-search <query>` — Algolia live search with relevance or `--by-date` ordering, plus `--tag`, `--since`, `--min-points` filters
- `hackernews-pp-cli local-search <query>` — Offline-only FTS5 search of the synced SQLite store
- `hackernews-pp-cli comments <id>` — Print a thread's comment tree using Algolia's one-shot `/items` fetch
- `hackernews-pp-cli hiring [regex]` — Filter the latest 'Ask HN: Who is hiring' thread by regex
- `hackernews-pp-cli freelance [regex]` — Filter the latest 'Ask HN: Freelancer? Seeking freelancer?' thread by regex
- `hackernews-pp-cli open <id>` — Print or launch a story URL or HN thread (`--launch` to open in browser, `--hn` for the HN thread URL)
- `hackernews-pp-cli bookmark add|list|rm` — Manage local bookmarks
- `hackernews-pp-cli since` — Show what changed on the front page since the last snapshot (auto-snapshots on every run)
- `hackernews-pp-cli pulse <topic>` — Per-day score and comment volume for a topic, plus the top stories that actually mention it
- `hackernews-pp-cli my <username>` — User submission history with score buckets, median, and best posting hour (UTC, weighted by points)
- `hackernews-pp-cli hiring-stats` — Aggregate Who's Hiring threads across N months — top languages, remote ratio, top companies
- `hackernews-pp-cli controversial` — Stories with the highest comment-to-point ratio, fetched live for ranking
- `hackernews-pp-cli repost <url>` — Has this URL been posted on HN? Lists prior submissions with scores and dates
- `hackernews-pp-cli velocity <id>` — Story's rank trajectory across local snapshots
- `hackernews-pp-cli tldr <id>` — Deterministic thread digest (top authors, depth histogram, heat metric)
- `hackernews-pp-cli doctor` — Self-diagnostic: API reachability, store writability, config sanity
- `hackernews-pp-cli export` / `import` — Backup, migrate, or restore local store data
- `hackernews-pp-cli workflow` — Compound workflows that combine multiple operations
- `hackernews-pp-cli api` / `agent-context` — Discoverability surfaces (browse endpoints, emit agent-friendly metadata)


## Freshness Contract

This printed CLI owns bounded freshness only for registered store-backed read command paths. In `--data-source auto` mode, those paths check `sync_state` and may run a bounded refresh before reading local data. `--data-source local` never refreshes. `--data-source live` reads the API and does not mutate the local store. Set `HACKERNEWS_NO_AUTO_REFRESH=1` to skip the freshness hook without changing source selection.

Covered paths:

- `hackernews-pp-cli ask`
- `hackernews-pp-cli jobs`
- `hackernews-pp-cli show`
- `hackernews-pp-cli stories`
- `hackernews-pp-cli stories top`
- `hackernews-pp-cli stories new`
- `hackernews-pp-cli stories best`
- `hackernews-pp-cli stories get <id>`
- `hackernews-pp-cli updates`
- `hackernews-pp-cli search <query>`

When JSON output uses the generated provenance envelope, freshness metadata appears at `meta.freshness`. Treat it as current-cache freshness for the covered command path, not a guarantee of complete historical backfill or API-specific enrichment.

### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
hackernews-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Daily morning scan with selected fields

```bash
hackernews-pp-cli stories top --limit 20 --agent --select id,title,url,score,by
```

Top 20 with only the fields you need — ~80% smaller payload than the full --json.

### Topic monitoring

```bash
hackernews-pp-cli pulse openai --days 7 --agent
```

Per-day breakdown of mentions, average score, and comment volume.

### Pre-submit dupe check

```bash
hackernews-pp-cli repost https://example.com/article
```

Lists every prior submission of that URL with score and date.

### Filter Who's Hiring for remote Go roles

```bash
hackernews-pp-cli hiring '(remote|REMOTE).*\bGo\b'
```

Regex against the latest monthly thread. Add --json for parsing.

## Auth Setup

No authentication required.

Run `hackernews-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  hackernews-pp-cli stories top --agent --select id,title,url,score
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
hackernews-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
hackernews-pp-cli feedback --stdin < notes.txt
hackernews-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.hackernews-pp-cli/feedback.jsonl`. They are never POSTed unless `HACKERNEWS_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `HACKERNEWS_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
hackernews-pp-cli profile save briefing --json
hackernews-pp-cli --profile briefing ask
hackernews-pp-cli profile list --json
hackernews-pp-cli profile show briefing
hackernews-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `hackernews-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → CLI installation
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## CLI Installation

1. Check Go is installed: `go version` (requires Go 1.23+)
2. Install:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/hackernews/cmd/hackernews-pp-cli@latest
   ```
3. Verify: `hackernews-pp-cli --version`
4. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/hackernews/cmd/hackernews-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add hackernews hackernews-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which hackernews-pp-cli`
   If not found, offer to install (see CLI Installation above).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   hackernews-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `hackernews-pp-cli <command> --help`.
