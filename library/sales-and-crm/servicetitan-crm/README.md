# ServiceTitan CRM CLI

**Every ServiceTitan CRM endpoint plus customer-360 search, lead-followup audit, and a 2-tool MCP surface that replaces the heavy ServiceTitan MCP for the CRM module.**

A per-module pp-cli for the ServiceTitan CRM module: 86 typed Cobra commands covering customers, locations, bookings, contacts, leads, and tags, backed by an offline SQLite store with FTS5 search. The MCP server uses the Cloudflare orchestration pattern to expose the entire surface as 2 intent tools (`servicetitan_crm_search` + `servicetitan_crm_execute`) instead of 86 raw mirrors, cutting per-turn agent token cost dramatically. Includes nine hand-built transcendence commands — `customers find`, `customers timeline`, `customers dedupe`, `customers stale`, `leads audit`, `leads convert`, `bookings prep-audit`, `segments export`, `sync run` — that answer cross-entity questions ServiceTitan's Web UI cannot.

## Install

The recommended path installs both the `servicetitan-crm-pp-cli` binary and the `pp-servicetitan-crm` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install servicetitan-crm
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install servicetitan-crm --cli-only
```


### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/servicetitan-crm-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-servicetitan-crm --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-servicetitan-crm --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-servicetitan-crm skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-servicetitan-crm. The skill defines how its required CLI can be installed.
```

## Authentication

ServiceTitan uses composed authentication: a static `ST-App-Key` header AND a short-lived OAuth2 bearer token from the client-credentials flow. The CLI reads `ST_APP_KEY`, `ST_CLIENT_ID`, `ST_CLIENT_SECRET`, and `ST_TENANT_ID` from the environment and refreshes the OAuth token automatically (~30 min TTL). Tenant id is path-positional on every endpoint and defaults to `ST_TENANT_ID` for novel commands so you don't have to pass it explicitly. Run `servicetitan-crm-pp-cli doctor` to verify all four credentials and reach the API. Whitespace in `ST_CLIENT_ID` or `ST_CLIENT_SECRET` is stripped defensively (a known JKA env gotcha that produced opaque `invalid_client` 400s).

## Quick Start

```bash
# Verify ST_APP_KEY, ST_CLIENT_ID, ST_CLIENT_SECRET, ST_TENANT_ID and confirm the OAuth token exchange works.
servicetitan-crm-pp-cli doctor


# Pull customers, locations, contacts, bookings, leads, and tags into the local SQLite store; required before any offline command.
servicetitan-crm-pp-cli sync run --since auto


# Look up a customer by phone with all linked locations, bookings, and contacts in one shot — the headline transcendence command.
servicetitan-crm-pp-cli customers find "555-0142" --json


# See untouched, converted, and stale leads — the Monday pipeline-review answer the Web UI cannot give.
servicetitan-crm-pp-cli leads audit --since 30d --json


# Surface tomorrow's bookings missing contact methods or gate codes for dispatch to fix before the truck rolls.
servicetitan-crm-pp-cli bookings prep-audit --window 1d --json

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Customer-360 lookups
- **`customers find`** — Find a customer by phone, email, name, or partial address and see every linked location, active booking, contact method, and tag in one shot.

  _Reach for this when a CSR or agent needs the full picture of a customer in under a second; faster than the ServiceTitan Web UI and avoids 4 chained API calls._

  ```bash
  servicetitan-crm-pp-cli customers find "555-0142" --json
  ```
- **`customers dedupe`** — Find likely duplicate customer records across normalized phone, email, or address, ranked by overlap strength so a CSR can review and merge.

  _Reach for this on the lead-intake step where duplicate-customer creation is the most common data-quality failure._

  ```bash
  servicetitan-crm-pp-cli customers dedupe --by phone --json
  ```
- **`customers timeline`** — Chronological event stream for one customer across creation, locations added, bookings, tag changes, and contact-method updates from the local store.

  _Reach for this on escalation calls or disputes when the operator needs the full customer history without screen-jumping._

  ```bash
  servicetitan-crm-pp-cli customers timeline 12345 --json --select events.kind,events.at,events.summary
  ```

### Pipeline audits
- **`leads audit`** — Bucket leads from the last N days into untouched, converted, and stale, with timestamps for each transition.

  _Reach for this for the Monday lead-pipeline review; the only way to surface stale leads without manual spreadsheet work._

  ```bash
  servicetitan-crm-pp-cli leads audit --since 30d --json
  ```
- **`segments export`** — Resolve a tag-AND-tag-AND-filter expression against the local store and emit a deterministic CSV or JSON segment list for marketing or campaign handoff.

  _Reach for this when handing a campaign-ready segment to email tooling or Sheets; replaces a fragile manual export pipeline._

  ```bash
  servicetitan-crm-pp-cli segments export municipal --and-tag commercial-warranty --no-booking-since 90d --csv
  ```
- **`customers stale`** — List customers whose latest booking, contact-method update, and notes are all older than N days — for re-engagement campaigns or archival review.

  _Reach for this when planning a re-engagement campaign or pruning the active customer list for marketing relevance._

  ```bash
  servicetitan-crm-pp-cli customers stale --no-activity 365d --csv
  ```

### Dispatch prep
- **`bookings prep-audit`** — List bookings in a window whose linked location is missing a confirmed contact method, gate code, or required tag — so dispatch can fix gaps before the truck rolls.

  _Reach for this in dispatch's afternoon prep ritual; surfaces the missing-info bookings that cause same-day cancellations._

  ```bash
  servicetitan-crm-pp-cli bookings prep-audit --window 1d --json
  ```

### Lifecycle operations
- **`leads convert`** — Run the lead-to-customer-to-location-to-optional-first-booking sequence as one operation, with --dry-run preview and idempotent retry.

  _Reach for this on lead intake to collapse a 3-5 screen ServiceTitan UI sequence into a previewable, scriptable single command._

  ```bash
  servicetitan-crm-pp-cli leads convert 8842 --book --dry-run
  ```

### Sync substrate
- **`sync-status`** — Show last-modified-on per CRM entity from the local store and report row counts for the offline cache.

  _Reach for this before any analysis command — it is the substrate that makes the offline transcendence commands possible._

  ```bash
  servicetitan-crm-pp-cli sync-status --json
  ```

## Usage

Run `servicetitan-crm-pp-cli --help` for the full command reference and flag list.

## Commands

### booking-provider

Manage booking provider


### booking-provider-tags

Manage booking provider tags

- **`servicetitan-crm-pp-cli booking-provider-tags create`** - Create a booking provider tag
- **`servicetitan-crm-pp-cli booking-provider-tags get`** - Gets a single booking provider tag by ID
- **`servicetitan-crm-pp-cli booking-provider-tags get-list`** - Gets a list of booking provider tags
- **`servicetitan-crm-pp-cli booking-provider-tags update`** - Update a booking provider tag

### bookings

Manage bookings

- **`servicetitan-crm-pp-cli bookings get`** - Gets a booking by ID
- **`servicetitan-crm-pp-cli bookings get-list`** - Gets a list of bookings

### contacts

Manage contacts

- **`servicetitan-crm-pp-cli contacts create`** - Creates a new contact
- **`servicetitan-crm-pp-cli contacts delete`** - Deletes a contact
- **`servicetitan-crm-pp-cli contacts get`** - Gets a contact specified by ID
- **`servicetitan-crm-pp-cli contacts get-by-relationship-id`** - Gets a list of contacts for a specified relationship ID
- **`servicetitan-crm-pp-cli contacts get-list`** - Gets a list of contacts
- **`servicetitan-crm-pp-cli contacts get-preference-metadata-list`** - Gets a list of preferences metadata
- **`servicetitan-crm-pp-cli contacts replace`** - Replaces a contact
- **`servicetitan-crm-pp-cli contacts search-methods`** - Search for contact methods
- **`servicetitan-crm-pp-cli contacts update`** - Updates a contact

### export

Manage crm export

- **`servicetitan-crm-pp-cli export bookings-get`** - Provides export feed for bookings
- **`servicetitan-crm-pp-cli export contacts-customers-contacts`** - Provides export feed for customer contacts
- **`servicetitan-crm-pp-cli export contacts-locations-contacts`** - Provides export feed for locations contacts
- **`servicetitan-crm-pp-cli export customers-get-customers`** - Provides export feed for customers
- **`servicetitan-crm-pp-cli export leads-leads`** - Provides export feed for leads
- **`servicetitan-crm-pp-cli export locations-locations`** - Provides export feed for appointments

### customers

Manage customers

- **`servicetitan-crm-pp-cli customers create`** - Creates a New Customer
- **`servicetitan-crm-pp-cli customers get`** - Gets a Customer specified by ID
- **`servicetitan-crm-pp-cli customers get-custom-field-types`** - Gets a list of custom field types available for customers
- **`servicetitan-crm-pp-cli customers get-list`** - Gets a list of Customers
- **`servicetitan-crm-pp-cli customers get-modified-contacts-list`** - Gets a list of contacts for a specific modified-on date range or by their Customer IDs. Either CustomerIds, modifiedOn or modifiedOnOrAfter parameter must be specified
- **`servicetitan-crm-pp-cli customers update`** - Update a customer

### leads

Manage leads

- **`servicetitan-crm-pp-cli leads create`** - Creates a lead
- **`servicetitan-crm-pp-cli leads get`** - Gets a lead specified by ID
- **`servicetitan-crm-pp-cli leads get-list`** - Gets a list of leads
- **`servicetitan-crm-pp-cli leads submit-form`** - Submits a lead form
- **`servicetitan-crm-pp-cli leads update`** - Updates a lead

### locations

Manage locations

- **`servicetitan-crm-pp-cli locations create`** - Creates a new location
- **`servicetitan-crm-pp-cli locations get`** - Gets a location specified by ID
- **`servicetitan-crm-pp-cli locations get-contacts-list`** - Gets a list of contacts for a specific ModifiedOn date range, CreatedOn date range or by their Location IDs. Either LocationIds, modifiedOn, modifiedOnOrAfter, createdBefore or createdOnOrAfter parameter must be specified.
- **`servicetitan-crm-pp-cli locations get-custom-field-types`** - Gets a list of custom field types available for locations
- **`servicetitan-crm-pp-cli locations get-list`** - Gets a list of locations
- **`servicetitan-crm-pp-cli locations update`** - Updates a location

### tags

Manage tags

- **`servicetitan-crm-pp-cli tags bulk-add`** - Add multiple tags to more than 1 customer
- **`servicetitan-crm-pp-cli tags bulk-remove`** - Remove multiple tags to more than 1 customer


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value

# JSON for scripting and agents
servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value --json

# Filter to specific fields
servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value --json --select id,name,status

# Dry run — show the request without sending
servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-servicetitan-crm -g
```

Then invoke `/pp-servicetitan-crm <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add servicetitan-crm servicetitan-crm-pp-mcp -e ST_CLIENT_ID=<your-token>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/servicetitan-crm-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `ST_CLIENT_ID` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "servicetitan-crm": {
      "command": "servicetitan-crm-pp-mcp",
      "env": {
        "ST_CLIENT_ID": "<your-key>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
servicetitan-crm-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/crm-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `ST_CLIENT_ID` | auth_flow_input | Yes | Set during initial auth setup. |
| `ST_CLIENT_SECRET` | auth_flow_input | Yes | Set during initial auth setup. |

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `servicetitan-crm-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $ST_CLIENT_ID`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **doctor reports 400 invalid_client from the OAuth endpoint** — Trim whitespace on ST_CLIENT_ID / ST_CLIENT_SECRET. Run `printenv ST_CLIENT_ID | cat -A` to see leading/trailing space. Re-export with the trimmed value.
- **All commands return 401 Unauthorized** — Both ST-App-Key AND OAuth bearer are required. Verify `ST_APP_KEY` is set and OAuth scopes include the resource you are calling (e.g., `tn.crm.customers:r`).
- **novel commands prompt for --tenant** — Set ST_TENANT_ID in the environment; novel commands default to that value. CSR/dispatch operators usually have it in their shell profile.
- **sync run reports 429 rate limited** — ServiceTitan's standard tier limits to ~120 req/min/tenant. The client's adaptive limiter handles 429s; add `--page-size 50` to slow the walk on the busiest list endpoints.
- **customers find returns no results after sync** — FTS5 indexes build during sync; if you ran sync mid-flight, run `sync run --rebuild-fts` to force re-indexing of customer.name, customer.email, location.address, and contact methods.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**BusyBee3333/servicetitan-mcp-2026-complete**](https://github.com/BusyBee3333/servicetitan-mcp-2026-complete) — TypeScript
- [**glassdoc/servicetitan-mcp**](https://github.com/glassdoc/servicetitan-mcp) — Python
- [**elliotpalmer/servicepytan**](https://github.com/elliotpalmer/servicepytan) — Python
- [**JordanDalton/ServiceTitanMcpServer**](https://github.com/JordanDalton/ServiceTitanMcpServer) — PHP
- [**compwright/servicetitan**](https://github.com/compwright/servicetitan) — PHP

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
