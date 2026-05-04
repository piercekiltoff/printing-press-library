# X (Twitter) CLI

**Use X from your terminal with your browser session — no paid API key, with a local SQLite database that powers relationship analytics no other tool offers.**

x-twitter is a CLI for X (formerly Twitter) that authenticates via your existing browser cookies — no $100/month API tier required. It mirrors every endpoint from the internal GraphQL API into commands, syncs your followers, following, tweets, and bookmarks into a local SQLite store, and unlocks relationship analytics like 'who isn't following me back', 'mutuals', and 'who unfollowed me' that no free tool offers. Every command is agent-native (--json, --select, exit codes) and exposed through MCP for Claude Desktop and other agent clients.

## Install

### Binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/x-twitter-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

### Go

```
go install github.com/mvanhorn/printing-press-library/library/other/x-twitter/cmd/x-twitter-pp-cli@latest
```

## Authentication

x-twitter uses cookie-based auth captured from your logged-in browser. Run `x-twitter auth login --chrome` to import your X session cookies (auth_token, ct0, guest_id) from Chrome. The CLI then signs every request with the matching x-csrf-token header and the standard X web-client bearer token. No paid X API tier needed. Cookies expire — re-run `auth login --chrome` whenever `doctor` reports an expired session.

## Quick Start

```bash
# Capture session cookies from your logged-in Chrome browser. No API key needed.
x-twitter auth login --chrome


# Verify auth, network, rate-limit headroom, and DB integrity.
x-twitter doctor


# Mirror your follower list into local SQLite. Required before relationship analytics.
x-twitter sync followers


# Mirror who you follow into local SQLite. Required before relationship analytics.
x-twitter sync following


# The killer feature: who you follow that doesn't follow you back.
x-twitter relationships not-following-back --json --select handle,name,last_tweet_at


# Aggregated profile + relationship + engagement summary in one call.
x-twitter whois @paulg --json

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Asymmetric relationship analytics
- **`relationships not-following-back`** — Surface accounts you follow that don't follow you back, with bio, last-active, and engagement context so you can decide who to unfollow.

  _Pick this for asymmetric-relationship audits. The local store makes it a single SQL query instead of a paid dashboard or a manual ZIP export._

  ```bash
  x-twitter relationships not-following-back --json --select handle,name,last_tweet_at,follower_count
  ```
- **`relationships mutuals`** — Show accounts that follow you AND that you follow back. Filter by recency, language, or follower count.

  _Pick this to identify your reciprocal network for outreach, networking, or community building._

  ```bash
  x-twitter relationships mutuals --json --since 90d
  ```
- **`relationships unfollowed-me`** — Diff follower snapshots over time to find accounts that unfollowed you in the last N days, with their bio and last activity.

  _Use this to understand audience churn. Snapshot-based, so no API call is needed for the diff itself._

  ```bash
  x-twitter relationships unfollowed-me --since 7d --json
  ```
- **`relationships fans`** — Inverse of not-following-back: people who follow you that you haven't followed back. Useful for reciprocal-follow strategies.

  _Pick this when growing reciprocal relationships. Filterable by recency to focus on new followers._

  ```bash
  x-twitter relationships fans --json --since 30d
  ```
- **`relationships overlap`** — Find accounts that follow BOTH user A and user B. Useful for network mapping, partnership outreach, OSINT.

  _Use for warm-intro discovery, audience comparison, or community analysis between any two accounts._

  ```bash
  x-twitter relationships overlap @paulg @sama --json --limit 50
  ```
- **`relationships new-followers`** — Diff follower snapshots forward to surface newly-acquired followers in the last N days.

  _Use this to acknowledge new followers and detect spikes in audience growth._

  ```bash
  x-twitter relationships new-followers --since 7d --json
  ```

### Audience hygiene
- **`relationships ghost-followers`** — List your followers who haven't tweeted in N days. Detect dead accounts inflating your follower count.

  _Pick this for audience hygiene. Helps explain low engagement-per-follower ratios._

  ```bash
  x-twitter relationships ghost-followers --days 90 --json
  ```
- **`audit inactive`** — Find accounts in your following list that have gone silent for N days. Recommend unfollow candidates.

  _Use this to clean up your timeline by unfollowing accounts that no longer post._

  ```bash
  x-twitter audit inactive --days 90 --json --select handle,name,last_tweet_at
  ```
- **`audit suspicious-followers`** — Heuristic bot detection: missing profile pic, default-pattern names, recent account age, high follow-to-follower ratio, no original tweets.

  _Pick this to estimate fake-follower percentage and identify likely bots polluting your audience._

  ```bash
  x-twitter audit suspicious-followers --threshold 0.7 --json
  ```

### Local intelligence
- **`tweets engagement`** — Local SQL over synced tweets: rank by likes + 2x retweets + 3x replies. Filter by user, language, media presence, or time window.

  _Pick this for content-performance analysis without re-querying the API for each report._

  ```bash
  x-twitter tweets engagement --top 10 --since 30d --user me --json
  ```
- **`whois`** — One command stitching profile, post velocity, engagement rate, mutuals with you, last active, and top recent tweets into a single agent-friendly view.

  _Use this as the first call when researching an account; subsequent commands can drill into specifics._

  ```bash
  x-twitter whois @paulg --json
  ```
- **`search saved`** — Search your synced tweet store with FTS5: regex, substring, by user, by language, by media presence. No rate limits, fully composable.

  _Pick this when iterating on the same query, building dashboards, or running analyses that need many search variations._

  ```bash
  x-twitter search saved --query 'embeddings' --since 30d --user me --json
  ```

### Local-first onboarding & export
- **`archive import`** — Bootstrap your local SQLite store from a Twitter data archive ZIP (the free download X provides on request). No rate-limit-bound sync needed for historical tweets, likes, DMs, blocks, mutes.

  _Pick this as the first step when onboarding — it bootstraps the entire local store from a free archive download instead of weeks of rate-limited sync._

  ```bash
  x-twitter archive import ~/Downloads/twitter-2026-04-01.zip --json
  ```
- **`export jsonl`** — Export tweets, DMs, or follows as version-controllable JSONL files, partitioned by year. Pairs with `--format markdown` for human-readable backups.

  _Use this for durable, text-diffable backups of your X data that survive any tool change._

  ```bash
  x-twitter export jsonl --resource tweets --output ./backup --yearly
  ```

## Usage

Run `x-twitter-pp-cli --help` for the full command reference and flag list.

## Commands

### 1-1

Manage 1 1

- **`x-twitter-pp-cli 1-1 get-friends-following-list`** - get friends following list
- **`x-twitter-pp-cli 1-1 get-search-typeahead`** - get search typeahead
- **`x-twitter-pp-cli 1-1 post-create-friendships`** - post create friendships
- **`x-twitter-pp-cli 1-1 post-destroy-friendships`** - post destroy friendships

### graphql

Manage graphql


### other

other

- **`x-twitter-pp-cli other other`** - This is not an actual endpoint

### twitter-search

Manage twitter search

- **`x-twitter-pp-cli twitter-search get-search-adaptive`** - get search adaptive


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
x-twitter-pp-cli 1-1 get-friends-following-list

# JSON for scripting and agents
x-twitter-pp-cli 1-1 get-friends-following-list --json

# Filter to specific fields
x-twitter-pp-cli 1-1 get-friends-following-list --json --select id,name,status

# Dry run — show the request without sending
x-twitter-pp-cli 1-1 get-friends-following-list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
x-twitter-pp-cli 1-1 get-friends-following-list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Retryable** - creates return "already exists" on retry, deletes return "already deleted"
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-x-twitter -g
```

Then invoke `/pp-x-twitter <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/x-twitter/cmd/x-twitter-pp-mcp@latest
```

Then register it:

```bash
claude mcp add x-twitter x-twitter-pp-mcp -e X_TWITTER_AUTH_TOKEN=<token> -e X_TWITTER_CT0=<ct0> -e X_TWITTER_GUEST_ID=<guest_id>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/x-twitter-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in the `X_TWITTER_AUTH_TOKEN`, `X_TWITTER_CT0`, and `X_TWITTER_GUEST_ID` cookie values from x.com DevTools when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.

```bash
go install github.com/mvanhorn/printing-press-library/library/other/x-twitter/cmd/x-twitter-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "x-twitter": {
      "command": "x-twitter-pp-mcp",
      "env": {
        "X_TWITTER_AUTH_TOKEN": "<token>",
        "X_TWITTER_CT0": "<ct0>",
        "X_TWITTER_GUEST_ID": "<guest_id>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
x-twitter-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

**Config file**: `~/.config/x-twitter-pp-cli/config.toml` (TOML format, mode 0600).
Override with the `X_TWITTER_CONFIG` env var or `--config` flag.

**Local SQLite store**: `~/.cache/x-twitter-pp-cli/store.db` (created on first sync).

**Environment variables** (all optional — preferred path is `auth login --chrome`):
- `X_TWITTER_AUTH_TOKEN` — X session cookie (`auth_token`). Required if not loaded from config.
- `X_TWITTER_CT0` — CSRF token cookie (`ct0`). Required; mirrored into `x-csrf-token` header.
- `X_TWITTER_GUEST_ID` — X guest tracking cookie (`guest_id`). Optional.
- `X_TWITTER_BEARER_TOKEN` — Override the default web-client bearer token.
- `X_TWITTER_BASE_URL` — Override the API base URL (default `https://x.com/i/api`). Useful for verify/test fixtures.

## Cookbook

Worked examples that exercise the CLI's unique capabilities. Every command below was verified against the shipped CLI; flag names and subcommand paths are exact.

### Find who you follow that doesn't follow back, top 50 by follower count

```bash
x-twitter-pp-cli sync followers
x-twitter-pp-cli sync following
x-twitter-pp-cli relationships not-following-back \
  --limit 50 \
  --json --select handle,name,followers_count,last_tweet_at
```

### Track who unfollowed you in the last week

```bash
# Run sync at least twice over time (e.g., daily cron) to populate snapshots.
x-twitter-pp-cli sync followers
# Then diff:
x-twitter-pp-cli relationships unfollowed-me --since 7d --json
```

### Audit dead accounts you follow (silent for 90+ days)

```bash
x-twitter-pp-cli audit inactive --days 90 --limit 100 \
  --json --select handle,name,last_tweet_at
```

### Detect likely bots in your follower list

```bash
x-twitter-pp-cli audit suspicious-followers --threshold 0.7 --json
```

### Find common followers between two users (warm-intro discovery)

```bash
x-twitter-pp-cli sync followers --user paulg
x-twitter-pp-cli sync followers --user sama
x-twitter-pp-cli relationships overlap @paulg @sama --limit 50 --json
```

### One-shot user lookup with deeply-nested response narrowing

```bash
x-twitter-pp-cli whois @vercel --json \
  --select profile.handle,profile.bio,engagement.tweets_per_day,engagement.avg_likes,relationships_with_me.mutuals_with_me_count,top_tweets.id,top_tweets.text
```

### Local SQL leaderboard of your top tweets last 30 days

```bash
x-twitter-pp-cli sync tweets --user me
x-twitter-pp-cli tweets engagement --user me --top 10 --since 30d --json \
  --select tweet_id,text,like_count,retweet_count
```

### Search your synced tweet store with FTS5 (no rate limits)

```bash
# Boolean / phrase queries supported by SQLite FTS5
x-twitter-pp-cli search saved --query "embeddings AND (vector OR retrieval)" --json
x-twitter-pp-cli search saved --query "anthropic" --user me --since 30d --has-media --json
```

### Bootstrap from a Twitter archive ZIP (skip rate-limit-bound sync)

```bash
x-twitter-pp-cli archive import ~/Downloads/twitter-2026-04-01.zip --json
```

### Export to Git-friendly JSONL with yearly shards

```bash
x-twitter-pp-cli export jsonl --resource tweets --output ./backup --yearly
git -C ./backup add .
git -C ./backup commit -m "X archive snapshot $(date +%F)"
```

### Pipe to jq for ad-hoc analysis

```bash
# Top 5 mutuals by follower count
x-twitter-pp-cli relationships mutuals --json | jq 'sort_by(-.followers_count) | .[:5]'
# CSV-ready follower handles for spreadsheet workflows
x-twitter-pp-cli relationships not-following-back --csv --select handle,name,last_tweet_at > non-mutuals.csv
```

## Troubleshooting

**Authentication errors (exit code 4)**
- Run `x-twitter-pp-cli doctor` to check credentials
- Cookies have expired? Re-run `x-twitter-pp-cli auth login --chrome`
- Verify env vars: `echo $X_TWITTER_AUTH_TOKEN | wc -c` (should be >40 chars)

**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the corresponding `list` command to see available items

**Rate limited (exit code 7)**
- Sync resumes from the last cursor. Re-run the sync command after the reset window
- Lower `--rate-limit` (e.g. `--rate-limit 0.5`) for slower sustained polling

### API-specific

- **doctor reports 'session expired' or commands return 401** — Cookies have expired. Re-run `x-twitter-pp-cli auth login --chrome` to capture fresh cookies from your browser.
- **'csrf cookie does not match header' (error code 353)** — Your ct0 cookie rotated. Re-run `auth login --chrome` to refresh.
- **Rate limit hit during sync** — Sync resumes from the last cursor. Re-run the sync command after the reset window in the error message.
- **GraphQL operation 'X' returned 'unknown query id'** — X rotates GraphQL query IDs occasionally. Update the spec source (`fa0311/twitter-openapi`) and regenerate, or override the query ID via env.
- **relationships commands return empty results** — Run `x-twitter-pp-cli sync followers` and `sync following` first — relationship commands query the local store.
- **archive import shows 0 tweets** — Older Twitter archives sometimes use a different file layout. The importer recognizes `data/tweets.js`, `data/tweet.js`, `data/follower.js`, `data/following.js`, `data/account.js`.

## Exit Codes

| Code | Meaning | Common cause |
|------|---------|--------------|
| 0 | Success | — |
| 2 | Usage error | Wrong flag, bad input |
| 3 | Not found | Bad ID, missing resource |
| 4 | Authentication | Expired cookies, wrong CSRF |
| 5 | Validation error | Schema/format mismatch |
| 7 | Rate limit | Slow down, retry after window |
| 10 | Server error | Upstream X.com 5xx |

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**sferik/x-cli**](https://github.com/sferik/x-cli) — Ruby (7000 stars)
- [**d60/twikit**](https://github.com/d60/twikit) — Python (4400 stars)
- [**trevorhobenshield/twitter-api-client**](https://github.com/trevorhobenshield/twitter-api-client) — Python (2500 stars)
- [**Rishikant181/Rettiwt-API**](https://github.com/Rishikant181/Rettiwt-API) — TypeScript (800 stars)
- [**Infatoshi/x-cli**](https://github.com/Infatoshi/x-cli) — Go (200 stars)
- [**EnesCinr/twitter-mcp**](https://github.com/EnesCinr/twitter-mcp) — TypeScript (200 stars)
- [**fa0311/twitter-openapi**](https://github.com/fa0311/twitter-openapi) — OpenAPI (181 stars)
- [**Infatoshi/x-mcp**](https://github.com/Infatoshi/x-mcp) — TypeScript (100 stars)
- [**steipete/birdclaw**](https://github.com/steipete/birdclaw) — TypeScript (50 stars)

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
