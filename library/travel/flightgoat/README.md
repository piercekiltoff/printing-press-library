# Flightgoat CLI

The GOAT flight CLI: search flights, explore nonstop routes, track live status, and find the longest, cheapest, and most-reliable flights from your airport. Wraps FlightAware AeroAPI with novel compound commands that join live data, price search, and local analytics.

## Install

```bash
go install github.com/mvanhorn/printing-press-library/library/travel/flightgoat/cmd/flightgoat-pp-cli@latest
```

Optional: install the `fli` helper for Google Flights price lookups used by `compare` and `gf-search`:

```bash
pipx install flights
```

## Authentication

Get an API key from the [FlightAware AeroAPI portal](https://www.flightaware.com/commercial/aeroapi/).

```bash
export FLIGHTGOAT_API_KEY_AUTH="<paste-your-key>"
```

Any of these environment variables also works (checked in order):
`FLIGHTGOAT_API_KEY_AUTH`, `FLIGHTGOAT_API_KEY`, `FLIGHTAWARE_API_KEY`,
`AEROAPI_API_KEY`, `AEROAPI_KEY`.

To override the base URL (for testing or enterprise endpoints):

```bash
export FLIGHTGOAT_BASE_URL="https://aeroapi.flightaware.com/aeroapi"
```

## Quick Start

```bash
# 1. Check credentials and connectivity
flightgoat-pp-cli doctor

# 2. Sync airports/operators to the local SQLite store for offline analytics
flightgoat-pp-cli sync

# 3. Daily brief for your home airport: departures, delays, weather, disruptions
flightgoat-pp-cli digest SEA

# 4. See every nonstop 8+ hour flight out of SEA this month
flightgoat-pp-cli longhaul SEA --min-hours 8

# 5. Compare Google Flights prices + AeroAPI on-time reliability for one route
flightgoat-pp-cli compare SEA LHR 2026-06-15
```

## Unique Features

These capabilities aren't available in any other tool for the AeroAPI.
They combine FlightAware's flight data with local analytics and (optionally)
Google Flights pricing to answer questions no single endpoint can.

- **`longhaul <airport>`** — Every nonstop flight from an airport that's at least N hours long over a month or date range. Answers the travel-hacker "where can I go nonstop for 10+ hours on points?" question in one call.
- **`explore <airport>`** — Every nonstop destination with typical duration, operating airlines, and frequency. A Kayak /direct matrix in your terminal.
- **`cheapest-longhaul <airport>`** — Cheapest days to fly the longest nonstop routes over a date range.
- **`ontime-now <airport>`** — Every departure from an airport today with live on-time status, delay in minutes, and status in one table.
- **`reliability <origin> <destination>`** — Historical on-time percentage for a specific route over the last N days, grouped by airline.
- **`compare <origin> <destination> <date>`** — Joins Google Flights price results with AeroAPI reliability per airline. Sorts by reliability so you can pick the cheapest flight that's likely to actually run on time.
- **`monitor <ident>`** — Watches a flight through its lifecycle. Polls at an interval, prints only status changes, exits on arrival.
- **`heatmap`** — Single sorted table showing where active delays are right now across every major airport. Filter by region.
- **`digest <airport>`** — One-command daily brief: departures today, active delays, current weather, top destinations.
- **`eta <ident>`** — Weather-adjusted ETA combining Foresight prediction with destination weather forecast.
- **`aircraft-bio <registration>`** — Full history of a tail number: recent flights, owner (when available), last known flight.
- **`gf-search <origin> <destination> <date>`** — Google Flights search via `fli` with optional `--alert-if-under PRICE` that creates a FlightAware alert when a cheap result appears.
- **`resolve <ident>`** — Shows every code for one physical flight (codeshare, canonical, operator) so travelers don't have to hunt codeshares.

## Commands

### Flight search and status

| Command | Description |
|---------|-------------|
| `flights get <ident>` | Information for a specific flight |
| `flights get-by-search` | Search flights with the AeroAPI query language |
| `flights get-by-position-search` | Search by geographic bounding box |
| `flights get-count-by-search` | Count flights matching a search |
| `history get <registration>` | Last known flight for an aircraft |
| `history get-flight <id>` | Historical details for one flight |
| `history get-flight-track <id>` | Position track history |
| `history get-flight-route <id>` | Filed route |
| `history get-flight-map <id>` | Track image |

### Airports, operators, aircraft

| Command | Description |
|---------|-------------|
| `airports get <code>` | Static information about an airport |
| `airports get-all` | All airports |
| `airports get-delays-for-all` | Every airport with active delays |
| `airports get-nearby` | Airports near a coordinate |
| `operators get <code>` | Operator (airline) information |
| `operators get-all` | All operators |
| `aircraft <type>` | Aircraft type details (shortcut) |

### Foresight and predictions

| Command | Description |
|---------|-------------|
| `foresight get-flight-with <ident>` | Flight with ML-enhanced predictions |
| `foresight get-flight-position-with <ident>` | Positions with Foresight data |

### Alerts and schedules

| Command | Description |
|---------|-------------|
| `alerts create` | Configure a new flight alert |
| `alerts get-all` | List configured alerts |
| `alerts update <id>` | Modify an alert |
| `alerts delete <id>` | Remove an alert |
| `alerts set-endpoint` | Set the default alert callback URL |
| `schedules get-by-date` | Scheduled flights by date |

### Data pipeline and analytics

| Command | Description |
|---------|-------------|
| `sync` | Sync API data to local SQLite for offline analytics |
| `search <query>` | Full-text or domain-specific search over synced data |
| `analytics` | Count, group-by, and summary operations on synced data |
| `export` | Export synced data as JSONL or JSON |
| `import` | Import JSONL records via create/upsert API calls |
| `tail` | Stream live changes by polling the API at an interval |

### Account and utilities

| Command | Description |
|---------|-------------|
| `doctor` | Check auth, config, and API connectivity |
| `auth` | Manage authentication tokens |
| `api` | Browse all API endpoints by interface name |
| `version` | Print version and build info |

## Output Formats

Every read command supports these flags:

```bash
# Default: human-readable table in terminal, JSON when piped
flightgoat-pp-cli longhaul SEA --min-hours 8

# Force JSON for scripting
flightgoat-pp-cli longhaul SEA --min-hours 8 --json

# Filter to specific fields
flightgoat-pp-cli explore SEA --json --select destination,airline,duration_minutes

# Compact output (reduced key set) for LLM tokens
flightgoat-pp-cli digest SEA --compact

# CSV for spreadsheet work
flightgoat-pp-cli reliability SEA LHR --days 30 --csv

# Dry run: print the planned request without sending
flightgoat-pp-cli compare SEA LHR 2026-06-15 --dry-run

# Agent mode: --json --compact --no-input --no-color --yes in one flag
flightgoat-pp-cli digest SEA --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** — never prompts, every input is a flag
- **Pipeable** — `--json` output to stdout, human hints and errors to stderr
- **Filterable** — `--select id,name` returns only the fields you need
- **Previewable** — `--dry-run` shows the planned request(s) without sending
- **Cacheable** — GET responses cached for 5 minutes, bypass with `--no-cache`
- **Offline** — `--data-source local` searches the local SQLite without any API call
- **Agent-safe by default** — no colors, no prompts unless `--human-friendly` is set
- **Provenance-aware** — every response carries a `source: live|local|cache` envelope

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## MCP Server

A companion MCP server (`flightgoat-pp-mcp`) exposes all commands as tools for Claude Code, Claude Desktop, Cursor, and any MCP-compatible client.

### Claude Code

```bash
claude mcp add flightgoat flightgoat-pp-mcp -e FLIGHTGOAT_API_KEY_AUTH=<your-key>
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "flightgoat": {
      "command": "flightgoat-pp-mcp",
      "env": {
        "FLIGHTGOAT_API_KEY_AUTH": "<your-key>"
      }
    }
  }
}
```

## Cookbook

```bash
# Find every nonstop 10+ hour flight out of JFK in July 2026
flightgoat-pp-cli longhaul JFK --min-hours 10 --month 2026-07

# Direct destinations from SEA with airlines and frequency
flightgoat-pp-cli explore SEA --json | jq '.[] | {dest: .destination, airlines, flights_per_week}'

# Which SEA to LHR airline is most reliable over the last 60 days
flightgoat-pp-cli reliability SEA LHR --days 60

# Combine price and reliability for a specific date
flightgoat-pp-cli compare SEA LHR 2026-06-15

# Watch UA100 until it lands, checking every 2 minutes
flightgoat-pp-cli monitor UA100 --interval 2m --until-arrival

# Morning brief for your home airport
flightgoat-pp-cli digest SEA --json

# Every US airport with active delays right now
flightgoat-pp-cli heatmap --region US

# Everything UA5 touches (codeshares, operators, canonical)
flightgoat-pp-cli resolve UA5

# Full tail number history for N628TS
flightgoat-pp-cli aircraft-bio N628TS

# Alert when JFK->CDG drops under $600
flightgoat-pp-cli gf-search JFK CDG 2026-07-01 --alert-if-under 600

# Sync operators + airports for offline browsing
flightgoat-pp-cli sync

# Search synced data for a string
flightgoat-pp-cli search "United" --type operators

# Count records by type in the local store
flightgoat-pp-cli analytics

# Export synced operators as JSONL
flightgoat-pp-cli export operators --format jsonl --output operators.jsonl

# Preview a compare without hitting the network or fli
flightgoat-pp-cli compare SEA LHR 2026-06-15 --dry-run
```

## Health Check

```
$ flightgoat-pp-cli doctor
  OK Config: ok
  OK Auth: configured
  OK API: reachable
  config_path: /Users/you/.config/flightgoat-pp-cli/config.toml
  base_url: https://aeroapi.flightaware.com/aeroapi
  version: 4.17.1
```

If `Auth: not configured` appears, set `FLIGHTGOAT_API_KEY_AUTH`. If `API: unreachable`, check your network and that your key has quota remaining.

## Configuration

Config file: `~/.config/flightgoat-pp-cli/config.toml` (override with `FLIGHTGOAT_CONFIG`).

Environment variables:

| Variable | Purpose |
|----------|---------|
| `FLIGHTGOAT_API_KEY_AUTH` | Primary API key variable |
| `FLIGHTGOAT_API_KEY` | Alias for the API key |
| `FLIGHTAWARE_API_KEY` | Alias matching FlightAware's own docs |
| `AEROAPI_API_KEY` | Alias matching the AeroAPI portal |
| `AEROAPI_KEY` | Shorter alias |
| `FLIGHTGOAT_BASE_URL` | Override the AeroAPI base URL |
| `FLIGHTGOAT_CONFIG` | Override the config file path |
| `NO_COLOR` | Disable colored output |

## Troubleshooting

**Authentication errors (exit code 4)**
- `flightgoat-pp-cli doctor` confirms whether your key is set and accepted
- Verify the env var: `echo $FLIGHTGOAT_API_KEY_AUTH`
- AeroAPI keys are not JWT tokens — they're simple opaque strings

**Not found errors (exit code 3)**
- Airport codes must be IATA (3 letters) or ICAO (4 letters), case-insensitive
- Flight idents should be the operator code plus number (e.g., `UA100`, not `UA 100`)

**Rate limit errors (exit code 7)**
- The CLI auto-retries with exponential backoff
- Use `--rate-limit 1.0` to cap requests per second
- Sync and transcendence commands respect `--max-pages` to bound cost

**`fli` not found**
- `compare` and `gf-search` shell out to the `fli` Python CLI
- Install it with `pipx install flights`

**Empty results for `longhaul` / `explore`**
- AeroAPI's scheduled-departures endpoint only returns data for a limited window
- Try a narrower `--from`/`--to` or a current month with `--month YYYY-MM`

## Rate Limits

AeroAPI rate limits depend on your plan tier. This CLI honors them with:

- Proactive client-side rate limiting (configurable with `--rate-limit`)
- Exponential backoff on 429 responses
- A 5-minute local cache for GET requests (bypass with `--no-cache`)
- Paginated commands cap at `--max-pages` (default 5) to bound cost

## Sources and Inspiration

This CLI was built by studying and crediting these community projects:

- [**fli**](https://github.com/punitarani/fli) — Python Google Flights client (used as a subprocess for `compare` and `gf-search`)
- [**aeroapps**](https://github.com/flightaware/aeroapps) — FlightAware's sample apps
- [**mcp-server-flight-aware-aeroapi**](https://github.com/mikedarke/mcp-server-flight-aware-aeroapi) — TypeScript MCP server
- [**flights-mcp-server**](https://github.com/smamidipaka6/flights-mcp-server) — Python MCP server
- [**google-flights-mcp**](https://github.com/HaroldLeo/google-flights-mcp) — Python
- [**salamentic-google-flights-mcp**](https://github.com/salamentic/google-flights-mcp) — Python
- [**fast-flights**](https://github.com/AWeirdDev/flights) — Python
- [**flights-search-cli**](https://github.com/jaebradley/flights-search-cli) — JavaScript

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press).

<!-- pr-218-features -->
## Agent workflow features

This CLI was patched to add these agent-workflow capabilities (see [`printing-press patch`](https://github.com/mvanhorn/cli-printing-press/pull/221)):

- **Named profiles** — save a set of flags under a name and reuse them: `flightgoat-pp-cli profile save <name> --<flag> <value>`, then `flightgoat-pp-cli --profile <name> <command>`. Flag precedence: explicit flag > env var > profile > default.
- **`--deliver`** — route command output to a sink other than stdout. Values: `file:<path>` writes atomically via tmp+rename; `webhook:<url>` POSTs as JSON (or NDJSON with `--compact`).
- **`feedback`** — record in-band feedback about the CLI. Entries append as JSON lines to `~/.flightgoat-pp-cli/feedback.jsonl`. When `FLIGHTGOAT_FEEDBACK_ENDPOINT` is set and either `--send` is passed or `FLIGHTGOAT_FEEDBACK_AUTO_SEND=true`, the entry is also POSTed upstream.
