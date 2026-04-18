# Dub CLI

Manage links, analytics, domains, and partner programs via the Dub API.

Learn more at [dub.co](https://dub.co).

## Install

### Go

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/dub/cmd/dub-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Authentication

Set your API key as an environment variable:

```bash
export DUB_TOKEN="your-api-key"
```

Get your API key from [Dub Settings > API Keys](https://app.dub.co/settings/tokens).

Or store it in the config file:

```bash
dub-pp-cli auth set-token YOUR_API_KEY
```

### Self-Hosted

If you run a self-hosted Dub instance, override the base URL:

```bash
export DUB_BASE_URL="https://api.your-instance.com"
```

## Quick Start

```bash
# Verify your setup
dub-pp-cli doctor

# Sync all data locally for offline analytics
dub-pp-cli sync --full

# See campaign performance across all tags
dub-pp-cli campaigns

# Track click-to-lead-to-sale conversion rates
dub-pp-cli funnel

# Create a short link
dub-pp-cli links create --url https://example.com/landing-page --key summer-sale
```

## Unique Features

These capabilities aren't available in any other tool for this API.

- **`campaigns`** -- See which marketing campaigns drive the most clicks, leads, and sales, aggregated by tag across all your links
- **`funnel`** -- Track click-to-lead-to-sale conversion rates per campaign or link to see where prospects drop off
- **`tags analytics`** -- Compare performance across all tags to find which campaigns and categories drive the most conversions
- **`customers journey`** -- See every link a customer clicked, when they became a lead, and when they purchased, in one timeline
- **`links stale`** -- Find links with declining or zero click velocity over time so you can clean up dead campaigns
- **`partners leaderboard`** -- Rank partners by commission earned, conversion rate, and clicks generated to find your top performers
- **`domains report`** -- See which custom domains are over- or underused with links-per-domain and click distribution breakdown

## Commands

### Link Management

| Command | Description |
|---------|-------------|
| `links` | List all links |
| `links create` | Create a short link |
| `links update` | Update a link |
| `links delete` | Delete a link |
| `links upsert` | Create or update a link |
| `links get-info` | Retrieve a specific link |
| `links get-count` | Get total link count |
| `links bulk-create` | Bulk create links |
| `links bulk-update` | Bulk update links |
| `links bulk-delete` | Bulk delete links |
| `links stale` | Find links with zero or declining clicks |
| `links duplicates` | Find links pointing to the same destination |

### Analytics & Insights

| Command | Description |
|---------|-------------|
| `analytics` | Query locally synced data with count and group-by |
| `analytics retrieve` | Retrieve analytics from the Dub API |
| `campaigns` | Campaign performance dashboard by tag |
| `funnel` | Click-to-lead-to-sale conversion rates |
| `tags analytics` | Tag-level performance rollup |

### Domains

| Command | Description |
|---------|-------------|
| `domains` | List all domains |
| `domains create` | Add a custom domain |
| `domains update` | Update a domain |
| `domains delete` | Remove a domain |
| `domains check-status` | Check domain availability |
| `domains register` | Register a new domain |
| `domains report` | Domain utilization report |

### Partners & Affiliates

| Command | Description |
|---------|-------------|
| `partners` | List partner links |
| `partners create` | Create or update a partner |
| `partners create-link` | Create a link for a partner |
| `partners upsert-link` | Upsert a link for a partner |
| `partners retrieve-analytics` | Partner analytics |
| `partners ban` | Ban a partner |
| `partners deactivate` | Deactivate a partner |
| `partners leaderboard` | Rank partners by performance |
| `commissions` | List all commissions |
| `commissions update` | Update a commission |
| `commissions bulk-update` | Bulk update commissions |
| `payouts` | List all payouts |

### Customers & Events

| Command | Description |
|---------|-------------|
| `customers` | List all customers |
| `customers get-id` | Retrieve a customer |
| `customers update` | Update a customer |
| `customers delete` | Delete a customer |
| `customers journey` | Customer interaction timeline |
| `events` | List all events |

### Tracking

| Command | Description |
|---------|-------------|
| `track lead` | Track a lead conversion |
| `track sale` | Track a sale |
| `track open` | Track a deep link open event |

### Organization

| Command | Description |
|---------|-------------|
| `tags` | List all tags |
| `tags create` | Create a tag |
| `tags update` | Update a tag |
| `tags delete` | Delete a tag |
| `folders` | List all folders |
| `folders create` | Create a folder |
| `folders update` | Update a folder |
| `folders delete` | Delete a folder |
| `qr` | Generate a QR code for a URL |

### Utilities

| Command | Description |
|---------|-------------|
| `doctor` | Check CLI health and credentials |
| `auth` | Manage authentication tokens |
| `sync` | Sync API data to local SQLite |
| `search` | Full-text search across synced data |
| `export` | Export data to JSONL or JSON |
| `import` | Import data from JSONL file |
| `tail` | Stream live changes by polling |
| `workflow archive` | Full offline sync of all resources |
| `workflow status` | Show local archive status |
| `api` | Browse all API endpoints |

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
dub-pp-cli links

# JSON for scripting and agents
dub-pp-cli links --json

# Filter to specific fields
dub-pp-cli links --json --select id,url,clicks

# CSV output
dub-pp-cli links --csv

# Compact output (key fields only, saves tokens)
dub-pp-cli links --compact

# Dry run -- show the request without sending
dub-pp-cli links create --url https://example.com --dry-run

# Agent mode -- JSON + compact + no prompts in one flag
dub-pp-cli links --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** -- never prompts, every input is a flag
- **Pipeable** -- `--json` output to stdout, errors to stderr
- **Filterable** -- `--select id,url,clicks` returns only fields you need
- **Previewable** -- `--dry-run` shows the request without sending
- **Retryable** -- creates return "already exists" on retry, deletes return "already deleted"
- **Confirmable** -- `--yes` for explicit confirmation of destructive actions
- **Piped input** -- `echo '{"key":"value"}' | dub-pp-cli links create --stdin`
- **Cacheable** -- GET responses cached for 5 minutes, bypass with `--no-cache`
- **Agent-safe by default** -- no colors or formatting unless `--human-friendly` is set
- **Progress events** -- paginated commands emit NDJSON events to stderr in default mode

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Cookbook

```bash
# Create a short link for a landing page
dub-pp-cli links create --url https://example.com/summer --key summer-sale --domain dub.sh

# Bulk create links from a JSON file
dub-pp-cli links bulk-create --input links.json

# Update a link's destination URL
dub-pp-cli links update --link-id clx_abc123 --url https://example.com/new-landing

# Find stale links with zero clicks in last 30 days
dub-pp-cli links stale --days 30

# Find duplicate links pointing to the same destination
dub-pp-cli links duplicates --json

# See campaign performance sorted by sales
dub-pp-cli campaigns --sort sales --limit 10

# Check conversion funnel for a specific campaign tag
dub-pp-cli funnel --tag "summer-sale"

# View a customer's full journey
dub-pp-cli customers journey cust_abc123

# Rank your top-performing partners
dub-pp-cli partners leaderboard --sort earnings --limit 10

# See tag-level analytics rollup
dub-pp-cli tags analytics --limit 10 --json

# Domain utilization report
dub-pp-cli domains report

# Export all links for backup
dub-pp-cli export links --format jsonl --output links.jsonl

# Sync data locally then search offline
dub-pp-cli sync --full
dub-pp-cli search "example.com" --type links

# Generate a QR code for a URL
dub-pp-cli qr --url https://dub.sh/my-link
```

## Health Check

```bash
$ dub-pp-cli doctor
  OK Config: ok
  FAIL Auth: not configured
  OK API: reachable
  config_path: /Users/you/.config/dub-pp-cli/config.toml
  base_url: https://api.dub.co
  version: 0.0.1
  hint: export DUB_TOKEN=<your-key>
```

## Configuration

Config file: `~/.config/dub-pp-cli/config.toml`

Environment variables:

| Variable | Description |
|----------|-------------|
| `DUB_TOKEN` | API key for authenticating with the Dub API |
| `DUB_BASE_URL` | Override the API base URL (default: `https://api.dub.co`) |
| `DUB_CONFIG` | Override the config file path |

## Troubleshooting

**Authentication errors (exit code 4)**
- Run `dub-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $DUB_TOKEN`
- Get a key from [app.dub.co/settings/tokens](https://app.dub.co/settings/tokens)

**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the list command to see available items (e.g., `dub-pp-cli links`)

**Rate limit errors (exit code 7)**
- The CLI auto-retries with exponential backoff
- Use `--rate-limit 2` to cap requests per second
- If persistent, wait a few minutes and try again

**Sync/search errors**
- Run `dub-pp-cli sync --full` to rebuild the local database
- Check `dub-pp-cli workflow status` to see sync state

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**dubco**](https://github.com/sujjeee/dubco) -- JavaScript (24 stars)
- [**dubco-mcp-server**](https://github.com/Gitmaxd/dubco-mcp-server-npm) -- JavaScript (7 stars)
- [**dub TypeScript SDK**](https://github.com/dubinc/dub-ts) -- TypeScript
- [**dub Python SDK**](https://github.com/dubinc/dub-python) -- Python

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)

<!-- pr-218-features -->
## Agent workflow features

This CLI was patched to add these agent-workflow capabilities (see [`printing-press patch`](https://github.com/mvanhorn/cli-printing-press/pull/221)):

- **Named profiles** — save a set of flags under a name and reuse them: `dub-pp-cli profile save <name> --<flag> <value>`, then `dub-pp-cli --profile <name> <command>`. Flag precedence: explicit flag > env var > profile > default.
- **`--deliver`** — route command output to a sink other than stdout. Values: `file:<path>` writes atomically via tmp+rename; `webhook:<url>` POSTs as JSON (or NDJSON with `--compact`).
- **`feedback`** — record in-band feedback about the CLI. Entries append as JSON lines to `~/.dub-pp-cli/feedback.jsonl`. When `DUB_FEEDBACK_ENDPOINT` is set and either `--send` is passed or `DUB_FEEDBACK_AUTO_SEND=true`, the entry is also POSTed upstream.
