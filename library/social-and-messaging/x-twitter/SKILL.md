---
name: pp-x-twitter
description: "Use X from your terminal with your browser session — no paid API key, with a local SQLite database that powers relationship analytics no other tool offers. Trigger phrases: `who isn't following me back on X`, `who unfollowed me on Twitter`, `show my mutuals on X`, `ghost followers Twitter`, `search my saved tweets`, `whois on X`, `use x-twitter`, `run x-twitter`."
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata: '{"openclaw":{"requires":{"bins":["x-twitter-pp-cli"]},"install":[{"id":"go","kind":"shell","command":"go install github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/cmd/x-twitter-pp-cli@latest","bins":["x-twitter-pp-cli"],"label":"Install via go install"}]}}'
---

# X (Twitter) — Printing Press CLI

x-twitter is a CLI for X (formerly Twitter) that authenticates via your existing browser cookies — no $100/month API tier required. It mirrors every endpoint from the internal GraphQL API into commands, syncs your followers, following, tweets, and bookmarks into a local SQLite store, and unlocks relationship analytics like 'who isn't following me back', 'mutuals', and 'who unfollowed me' that no free tool offers. Every command is agent-native (--json, --select, exit codes) and exposed through MCP for Claude Desktop and other agent clients.

## When to Use This CLI

Use x-twitter when you need scriptable, agent-driven access to X without paying for the V2 API tier. It excels at relationship analytics (who isn't following you back, who unfollowed you, mutuals overlap between two users), audience hygiene (ghost followers, suspicious-bot detection), and local-store-backed analytics (engagement leaderboards, saved search). For one-off interactive browsing the X web UI is faster, but for any repeated query, automation, or agent workflow x-twitter is a much better fit because it caches data locally and exposes everything as JSON.

## Unique Capabilities

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

## Command Reference

**1-1** — Manage 1 1

- `x-twitter-pp-cli 1-1 get-friends-following-list` — get friends following list
- `x-twitter-pp-cli 1-1 get-search-typeahead` — get search typeahead
- `x-twitter-pp-cli 1-1 post-create-friendships` — post create friendships
- `x-twitter-pp-cli 1-1 post-destroy-friendships` — post destroy friendships

**graphql** — Manage graphql


**other** — other

- `x-twitter-pp-cli other` — This is not an actual endpoint

**twitter-search** — Manage twitter search

- `x-twitter-pp-cli twitter-search` — get search adaptive


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
x-twitter-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Find who you follow that doesn't follow back

```bash
x-twitter sync followers && x-twitter sync following && x-twitter relationships not-following-back --json --select handle,name,follower_count,last_tweet_at | jq '.[] | select(.follower_count > 1000)'
```

Sync both directions, then list the asymmetric edges, narrowing to influencers.

### Track who unfollowed you this week

```bash
x-twitter sync followers && x-twitter relationships unfollowed-me --since 7d --json
```

Each sync builds a follower snapshot; the diff command surfaces the deltas.

### Audit your most engaged tweets last month

```bash
x-twitter sync tweets --user me && x-twitter tweets engagement --top 20 --since 30d --user me --json --select id,text,like_count,retweet_count,reply_count
```

Local SQL ranks your tweets by weighted engagement without burning rate limit.

### Find common followers between two users (warm intro discovery)

```bash
x-twitter sync followers --user @paulg && x-twitter sync followers --user @sama && x-twitter relationships overlap @paulg @sama --json --limit 50
```

Sync both users' followers, then INTERSECT them to find who follows both — useful for partnership outreach.

### Whois with deeply-nested response narrowing

```bash
x-twitter whois @vercel --json --select profile.handle,profile.bio,engagement.tweets_per_day,engagement.avg_likes,mutuals_with_me_count,top_tweets.id,top_tweets.text
```

Whois returns nested data; --select with dotted paths lets agents fetch only the fields they need to keep context small.

## Auth Setup

x-twitter uses cookie-based auth captured from your logged-in browser. Run `x-twitter auth login --chrome` to import your X session cookies (auth_token, ct0, guest_id) from Chrome. The CLI then signs every request with the matching x-csrf-token header and the standard X web-client bearer token. No paid X API tier needed. Cookies expire — re-run `auth login --chrome` whenever `doctor` reports an expired session.

Run `x-twitter-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  x-twitter-pp-cli 1-1 get-friends-following-list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag

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
x-twitter-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
x-twitter-pp-cli feedback --stdin < notes.txt
x-twitter-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.x-twitter-pp-cli/feedback.jsonl`. They are never POSTed unless `X_TWITTER_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `X_TWITTER_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
x-twitter-pp-cli profile save briefing --json
x-twitter-pp-cli --profile briefing 1-1 get-friends-following-list
x-twitter-pp-cli profile list --json
x-twitter-pp-cli profile show briefing
x-twitter-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `x-twitter-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → CLI installation
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## CLI Installation

1. Check Go is installed: `go version` (requires Go 1.23+)
2. Install:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/cmd/x-twitter-pp-cli@latest
   ```
3. Verify: `x-twitter-pp-cli --version`
4. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/cmd/x-twitter-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add x-twitter-pp-mcp -- x-twitter-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which x-twitter-pp-cli`
   If not found, offer to install (see CLI Installation above).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   x-twitter-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `x-twitter-pp-cli <command> --help`.
