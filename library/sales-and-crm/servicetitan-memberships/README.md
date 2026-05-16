# ServiceTitan Memberships CLI

**A focused per-module ServiceTitan CLI for Memberships — sync once, then audit renewals, overdue events, template drift, and recurring revenue the ST UI cannot.**

This CLI (servicetitan-memberships-pp-cli) mirrors every ServiceTitan Memberships v2 endpoint (customer memberships, membership types, recurring services, recurring service types, recurring service events, invoice templates, plus seven export feeds) and adds twelve novel commands that join across them in a local SQLite cache. It replaces the heavy general ST MCP for membership work with a thin search+execute MCP surface — so an agent doing renewal targeting, overdue-visit triage, churn risk scoring, or recurring-revenue rollups loads only the memberships module.

## Install

The recommended path installs both the `servicetitan-memberships-pp-cli` binary and the `pp-servicetitan-memberships` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install servicetitan-memberships
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install servicetitan-memberships --cli-only
```


### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/servicetitan-memberships-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-servicetitan-memberships --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-servicetitan-memberships --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-servicetitan-memberships skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-servicetitan-memberships. The skill defines how its required CLI can be installed.
```

## Authentication

ServiceTitan uses composed auth: a static ST-App-Key header (set ST_APP_KEY) plus an OAuth2 client-credentials bearer token (set ST_CLIENT_ID and ST_CLIENT_SECRET). Run servicetitan-memberships-pp-cli auth login to walk the credential setup, then set the three credentials plus ST_TENANT_ID (your numeric tenant ID). Whitespace is stripped defensively on all three OAuth env vars — a known JKA gotcha that produces opaque invalid_client 400s when a trailing newline sneaks in from a copy-paste. The integration's OAuth client needs the read scopes for every list/get plus tn.mem.memberships:w, tn.mem.invoicetemplates:w, tn.mem.recurringservices:w, and tn.mem.recurringserviceevents:w for the mutation endpoints (memberships sale + update, invoice-templates create + update, recurring-services update, recurring-service-events mark-complete + mark-incomplete).

## Quick Start

```bash
# Confirm ST_APP_KEY, bearer token, ST_TENANT_ID, and base URL before anything else.
servicetitan-memberships-pp-cli doctor


# Pull memberships, membership-types, recurring-services, recurring-service-events, recurring-service-types, and invoice-templates into the local store and snapshot membership status history.
servicetitan-memberships-pp-cli sync


# See renewal, overdue-event, drift, risk, stale-service, and revenue-at-risk counts in one rollup.
servicetitan-memberships-pp-cli health --agent


# List active memberships whose to-date is within the next 30 days for the renewal pipeline.
servicetitan-memberships-pp-cli renewals --within 30 --agent


# Show recurring-service events past their date on still-active memberships — the visits we owe.
servicetitan-memberships-pp-cli overdue-events --agent

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Lifecycle that compounds
- **`renewals`** — List active memberships whose to-date is within a window so the renewal task is one click away.

  _Reach for this when an agent needs the full renewal target list across the tenant, not one membership at a time._

  ```bash
  servicetitan-memberships-pp-cli renewals --within 30 --agent
  ```
- **`expiring`** — Every membership whose to-date falls inside a window, including already-cancelled, for lapse-recovery sweeps.

  _Use this when an agent needs both renewal and recovery targets in one list with deeply-nested data narrowed by --select._

  ```bash
  servicetitan-memberships-pp-cli expiring --within 60 --agent --select id,customerId,to,status
  ```
- **`risk`** — Rule-engine score per membership: follow-up status, no payment method, lapsed bills, stale events, near to-date.

  _Pick this when an agent is prioritizing retention outreach across the membership book._

  ```bash
  servicetitan-memberships-pp-cli risk --agent --limit 25
  ```

### Local state that compounds
- **`overdue-events`** — Recurring-service events past their date on still-active memberships — what we should have already visited.

  _Pick this before scheduling work — overdue events are the SLA tab the UI never quite shows._

  ```bash
  servicetitan-memberships-pp-cli overdue-events --agent
  ```
- **`schedule`** — Compact view of upcoming recurring-service events grouped by date and location.

  _Reach for this when dispatch needs to pre-stage the next two weeks of recurring service work._

  ```bash
  servicetitan-memberships-pp-cli schedule --within 14 --agent
  ```
- **`drift`** — Compares each active membership's recurring-services against the membership-type template, flags missing or extra services.

  _Use this when a member's care-plan deliverables look wrong against their billed membership type._

  ```bash
  servicetitan-memberships-pp-cli drift --agent
  ```
- **`revenue`** — Monthly recurring revenue from memberships joined to membership-types.durationBilling, broken down by business unit and billing frequency.

  _Reach for this when you need the recurring-revenue picture without exporting a report._

  ```bash
  servicetitan-memberships-pp-cli revenue --by month --agent
  ```
- **`stale-services`** — Recurring services on active memberships with no completed event in N or more months, even when the recurrence says one should have happened.

  _Pick this when auditing whether recurring service obligations are actually being delivered._

  ```bash
  servicetitan-memberships-pp-cli stale-services --months 6 --agent
  ```
- **`bill-preview`** — Show the next bill date and line-item amount for a single membership, resolved through its membership-type duration-billing.

  _Reach for this when a customer asks what they're about to be charged before the invoice runs._

  ```bash
  servicetitan-memberships-pp-cli bill-preview 42 --agent
  ```

### Agent-native plumbing
- **`health`** — One compact rollup of renewals, overdue events, drift, risk, stale services, and revenue-at-risk counts.

  _Use this first in any agent session that touches memberships — it primes the model on what is broken._

  ```bash
  servicetitan-memberships-pp-cli health --agent
  ```
- **`complete`** — Mark a recurring-service event complete with a required job link, then refresh the local snapshot so overdue filters update.

  _Use this in field-tech workflows where the event-to-job link is the whole point of marking complete._

  ```bash
  servicetitan-memberships-pp-cli complete 99 --job 12345 --dry-run
  ```
- **`find`** — Forgiving ranked search over synced memberships using customer, importId, memo, customFields, and membership-type name.

  _Pick this when an agent or office staffer is describing a member instead of pasting an exact ID._

  ```bash
  servicetitan-memberships-pp-cli find 'Smith family well plan' --agent
  ```

## Usage

Run `servicetitan-memberships-pp-cli --help` for the full command reference and flag list.

## Commands

### invoice-templates

Manage invoice templates

- **`servicetitan-memberships-pp-cli invoice-templates create`** - Creates new invoice template
- **`servicetitan-memberships-pp-cli invoice-templates get`** - Gets invoice template specified by ID
- **`servicetitan-memberships-pp-cli invoice-templates get-list`** - Please note this endpoint does not allow to enumerate all invoice templates.
Use the Customer Membership endpoint (for billing template) or
Recurring Service endpoint (for invoice template) to get invoice template IDs.
- **`servicetitan-memberships-pp-cli invoice-templates update`** - Updates specified invoice template in "patch" mode

### membership-types

Manage membership types

- **`servicetitan-memberships-pp-cli membership-types get`** - Gets membership type specified by ID
- **`servicetitan-memberships-pp-cli membership-types get-list`** - Gets a list of membership types

### memberships

Manage memberships

- **`servicetitan-memberships-pp-cli memberships customer-create`** - Creates membership sale invoice
- **`servicetitan-memberships-pp-cli memberships customer-get`** - Gets customer membership specified by ID
- **`servicetitan-memberships-pp-cli memberships customer-get-custom-fields`** - Gets a list of custom field types that apply to customer memberships
- **`servicetitan-memberships-pp-cli memberships customer-get-list`** - Gets a list of customer memberships
- **`servicetitan-memberships-pp-cli memberships customer-update`** - Updates specified customer membership in "patch" mode

### memberships-export

Manage memberships export

- **`servicetitan-memberships-pp-cli memberships-export invoice-templates`** - Provides export feed for invoice templates
- **`servicetitan-memberships-pp-cli memberships-export location-recurring-service-events`** - Provides export feed for recurring service events
- **`servicetitan-memberships-pp-cli memberships-export location-recurring-services`** - Provides export feed for recurring services
- **`servicetitan-memberships-pp-cli memberships-export membership-status-changes`** - Provides export feed for customer membership status changes
- **`servicetitan-memberships-pp-cli memberships-export membership-types`** - Provides export feed for membership types
- **`servicetitan-memberships-pp-cli memberships-export memberships`** - Provides export feed for customer memberships
- **`servicetitan-memberships-pp-cli memberships-export recurring-service-types`** - Provides export feed for recurring service types

### recurring-service-events

Manage recurring service events

- **`servicetitan-memberships-pp-cli recurring-service-events location-get-list`** - Gets a list of recurring service events
- **`servicetitan-memberships-pp-cli recurring-service-events mark-complete`** - Mark a recurring service event complete (writes to ServiceTitan)
- **`servicetitan-memberships-pp-cli recurring-service-events mark-incomplete`** - Mark a recurring service event incomplete (writes to ServiceTitan)

### recurring-service-types

Manage recurring service types

- **`servicetitan-memberships-pp-cli recurring-service-types get`** - Gets recurring service type specified by ID
- **`servicetitan-memberships-pp-cli recurring-service-types get-list`** - Gets a list of recurring service types

### recurring-services

Manage recurring services

- **`servicetitan-memberships-pp-cli recurring-services location-get`** - Gets recurring service specified by ID
- **`servicetitan-memberships-pp-cli recurring-services location-get-list`** - Gets a list of recurring services
- **`servicetitan-memberships-pp-cli recurring-services location-update`** - Updates specified recurring service in "patch" mode


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
servicetitan-memberships-pp-cli renewals --within 30

# JSON for scripting and agents
servicetitan-memberships-pp-cli renewals --within 30 --json

# Filter to specific fields
servicetitan-memberships-pp-cli renewals --within 30 --json --select id,customerId,to,status

# Dry run — show the request without sending
servicetitan-memberships-pp-cli memberships customer-update 51234 --dry-run

# Agent mode — JSON + compact + no prompts in one flag
servicetitan-memberships-pp-cli health --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-servicetitan-memberships -g
```

Then invoke `/pp-servicetitan-memberships <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add servicetitan-memberships servicetitan-memberships-pp-mcp \
  -e ST_APP_KEY=<your-app-key> \
  -e ST_CLIENT_ID=<your-client-id> \
  -e ST_CLIENT_SECRET=<your-client-secret> \
  -e ST_TENANT_ID=<your-numeric-tenant-id>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/servicetitan-memberships-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `ST_APP_KEY`, `ST_CLIENT_ID`, `ST_CLIENT_SECRET`, and `ST_TENANT_ID` when Claude Desktop prompts you (composed auth requires all four).

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "servicetitan-memberships": {
      "command": "servicetitan-memberships-pp-mcp",
      "env": {
        "ST_APP_KEY": "<your-app-key>",
        "ST_CLIENT_ID": "<your-client-id>",
        "ST_CLIENT_SECRET": "<your-client-secret>",
        "ST_TENANT_ID": "<your-numeric-tenant-id>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
servicetitan-memberships-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/servicetitan-memberships-pp-cli/config.toml` (override with `SERVICETITAN_MEMBERSHIPS_CONFIG`).

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `ST_APP_KEY` | per_call | Yes | Static ServiceTitan App Key, sent as the `ST-App-Key` header on every call. |
| `ST_CLIENT_ID` | auth_flow_input | Yes | OAuth2 client_id; exchanged for a bearer token. |
| `ST_CLIENT_SECRET` | auth_flow_input | Yes | OAuth2 client_secret paired with `ST_CLIENT_ID`. |
| `ST_TENANT_ID` | per_call | Yes | Numeric ServiceTitan tenant ID; every Memberships path is tenant-scoped. |

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `servicetitan-memberships-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $ST_CLIENT_ID`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **auth login returns invalid_client 400 even though ST_CLIENT_ID and ST_CLIENT_SECRET look right.** — Re-paste the secret into a single-line env entry. ServiceTitan's OAuth server rejects trailing whitespace; this CLI strips it defensively but check that the values don't contain embedded newlines (echo -n on the value should produce no extra blank line).
- **List commands return 403 forbidden even after auth login succeeds.** — Your integration's OAuth client is missing a tn.mem.* scope. Check the developer-portal app and add tn.mem.memberships:r, tn.mem.membershiptypes:r, tn.mem.recurringservices:r, tn.mem.recurringserviceevents:r, tn.mem.recurringservicetypes:r, tn.mem.invoicetemplates:r. Mutation commands additionally require the :w variants.
- **sync says 429 too many requests.** — ServiceTitan rate-limits to ~7000 requests per hour per environment per app key. The CLI's AdaptiveLimiter backs off automatically; if you hit the cap repeatedly, run sync --resources memberships,recurring-service-events to scope and let it finish before retrying the rest.
- **drift reports template gaps but the membership UI looks correct.** — Re-run sync — membership-types.recurring-services is sub-resource data that only refreshes on a full sync. After sync, drift's view is from the same modifiedOn cursor; if a membership type changed after your last sync, the drift report lags until you resync.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
