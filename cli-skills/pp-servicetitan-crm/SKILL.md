---
name: pp-servicetitan-crm
description: "Every ServiceTitan CRM endpoint plus customer-360 search, lead-followup audit, and a 2-tool MCP surface that... Trigger phrases: `find ServiceTitan customer`, `customer 360 lookup`, `lead followup audit`, `ServiceTitan booking prep`, `tag segment export`, `use servicetitan-crm`, `run servicetitan-crm`."
author: "user"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - servicetitan-crm-pp-cli
---

# ServiceTitan CRM — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `servicetitan-crm-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install servicetitan-crm --cli-only
   ```
2. Verify: `servicetitan-crm-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

A per-module pp-cli for the ServiceTitan CRM module: 86 typed Cobra commands covering customers, locations, bookings, contacts, leads, and tags, backed by an offline SQLite store with FTS5 search. The MCP server uses the Cloudflare orchestration pattern to expose the entire surface as 2 intent tools (`servicetitan_crm_search` + `servicetitan_crm_execute`) instead of 86 raw mirrors, cutting per-turn agent token cost dramatically. Includes nine hand-built transcendence commands — `customers find`, `customers timeline`, `customers dedupe`, `customers stale`, `leads audit`, `leads convert`, `bookings prep-audit`, `segments export`, `sync run` — that answer cross-entity questions ServiceTitan's Web UI cannot.

## When to Use This CLI

Use this CLI when an agent or operator needs to interact with the ServiceTitan CRM module without paying the heavy ServiceTitan MCP's ~400-tool token tax per turn. The CRM surface here is the customer/location/contact/lead/booking/tag stack. Reach for the transcendence commands (`customers find`, `leads audit`, `bookings prep-audit`, `segments export`) when the answer requires joining across CRM entities — the ServiceTitan Web UI and the heavy MCP can hit the endpoints individually, but only this CLI keeps a local copy that supports SQL-shaped queries. For dispatch board, project rollup, or jobs/appointments work, use the sibling `servicetitan-jpm-pp-cli` instead.

## Unique Capabilities

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

## Command Reference

**booking-provider** — Manage booking provider


**booking-provider-tags** — Manage booking provider tags

- `servicetitan-crm-pp-cli booking-provider-tags create` — Create a booking provider tag
- `servicetitan-crm-pp-cli booking-provider-tags get` — Gets a single booking provider tag by ID
- `servicetitan-crm-pp-cli booking-provider-tags get-list` — Gets a list of booking provider tags
- `servicetitan-crm-pp-cli booking-provider-tags update` — Update a booking provider tag

**bookings** — Manage bookings

- `servicetitan-crm-pp-cli bookings get` — Gets a booking by ID
- `servicetitan-crm-pp-cli bookings get-list` — Gets a list of bookings

**contacts** — Manage contacts

- `servicetitan-crm-pp-cli contacts create` — Creates a new contact
- `servicetitan-crm-pp-cli contacts delete` — Deletes a contact
- `servicetitan-crm-pp-cli contacts get` — Gets a contact specified by ID
- `servicetitan-crm-pp-cli contacts get-by-relationship-id` — Gets a list of contacts for a specified relationship ID
- `servicetitan-crm-pp-cli contacts get-list` — Gets a list of contacts
- `servicetitan-crm-pp-cli contacts get-preference-metadata-list` — Gets a list of preferences metadata
- `servicetitan-crm-pp-cli contacts replace` — Replaces a contact
- `servicetitan-crm-pp-cli contacts search-methods` — Search for contact methods
- `servicetitan-crm-pp-cli contacts update` — Updates a contact

**export** — Manage crm export

- `servicetitan-crm-pp-cli export bookings-get` — Provides export feed for bookings
- `servicetitan-crm-pp-cli export contacts-customers-contacts` — Provides export feed for customer contacts
- `servicetitan-crm-pp-cli export contacts-locations-contacts` — Provides export feed for locations contacts
- `servicetitan-crm-pp-cli export customers-get-customers` — Provides export feed for customers
- `servicetitan-crm-pp-cli export leads-leads` — Provides export feed for leads
- `servicetitan-crm-pp-cli export locations-locations` — Provides export feed for appointments

**customers** — Manage customers

- `servicetitan-crm-pp-cli customers create` — Creates a New Customer
- `servicetitan-crm-pp-cli customers get` — Gets a Customer specified by ID
- `servicetitan-crm-pp-cli customers get-custom-field-types` — Gets a list of custom field types available for customers
- `servicetitan-crm-pp-cli customers get-list` — Gets a list of Customers
- `servicetitan-crm-pp-cli customers get-modified-contacts-list` — Gets a list of contacts for a specific modified-on date range or by their Customer IDs. Either CustomerIds,...
- `servicetitan-crm-pp-cli customers update` — Update a customer

**leads** — Manage leads

- `servicetitan-crm-pp-cli leads create` — Creates a lead
- `servicetitan-crm-pp-cli leads get` — Gets a lead specified by ID
- `servicetitan-crm-pp-cli leads get-list` — Gets a list of leads
- `servicetitan-crm-pp-cli leads submit-form` — Submits a lead form
- `servicetitan-crm-pp-cli leads update` — Updates a lead

**locations** — Manage locations

- `servicetitan-crm-pp-cli locations create` — Creates a new location
- `servicetitan-crm-pp-cli locations get` — Gets a location specified by ID
- `servicetitan-crm-pp-cli locations get-contacts-list` — Gets a list of contacts for a specific ModifiedOn date range, CreatedOn date range or by their Location IDs. Either...
- `servicetitan-crm-pp-cli locations get-custom-field-types` — Gets a list of custom field types available for locations
- `servicetitan-crm-pp-cli locations get-list` — Gets a list of locations
- `servicetitan-crm-pp-cli locations update` — Updates a location

**tags** — Manage tags

- `servicetitan-crm-pp-cli tags bulk-add` — Add multiple tags to more than 1 customer
- `servicetitan-crm-pp-cli tags bulk-remove` — Remove multiple tags to more than 1 customer


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
servicetitan-crm-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Customer 360 from a phone fragment

```bash
servicetitan-crm-pp-cli customers find "555-0142" --json --select customer.name,customer.id,locations.address,bookings.start,bookings.status,contacts.email
```

FTS5 lookup joined across customers + locations + bookings + contacts; --select trims the deeply nested response to only the fields a CSR needs on a 30-second call window.

### Monday lead-pipeline review

```bash
servicetitan-crm-pp-cli leads audit --since 30d --json --select untouched.id,untouched.created,converted.id,converted.customer_id,stale.id,stale.last_contact
```

Bucketed lead status with timestamps; pipe to jq for slack/email digests.

### Tomorrow's under-prepped bookings

```bash
servicetitan-crm-pp-cli bookings prep-audit --window 1d --json --select id,location.address,missing
```

Local join surfacing bookings whose location lacks confirmed contact, gate code, or required tag — the dispatcher's afternoon prep query.

### Marketing segment export to CSV

```bash
servicetitan-crm-pp-cli segments export municipal --and-tag commercial-warranty --no-booking-since 90d --csv > segment.csv
```

Boolean tag expression + recency filter; deterministic CSV column order so downstream tooling can rely on the schema.

### Find duplicate customer records by phone

```bash
servicetitan-crm-pp-cli customers dedupe --by phone --json --select cluster.phone,cluster.customers.id,cluster.customers.name,cluster.score
```

GROUP BY normalized phone surfaces duplicate clusters with overlap scores; CSR uses the output to merge before creating yet another duplicate.

## Auth Setup

ServiceTitan uses composed authentication: a static `ST-App-Key` header AND a short-lived OAuth2 bearer token from the client-credentials flow. The CLI reads `ST_APP_KEY`, `ST_CLIENT_ID`, `ST_CLIENT_SECRET`, and `ST_TENANT_ID` from the environment and refreshes the OAuth token automatically (~30 min TTL). Tenant id is path-positional on every endpoint and defaults to `ST_TENANT_ID` for novel commands so you don't have to pass it explicitly. Run `servicetitan-crm-pp-cli doctor` to verify all four credentials and reach the API. Whitespace in `ST_CLIENT_ID` or `ST_CLIENT_SECRET` is stripped defensively (a known JKA env gotcha that produced opaque `invalid_client` 400s).

Run `servicetitan-crm-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  servicetitan-crm-pp-cli booking-provider-tags get mock-value mock-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

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
servicetitan-crm-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
servicetitan-crm-pp-cli feedback --stdin < notes.txt
servicetitan-crm-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.servicetitan-crm-pp-cli/feedback.jsonl`. They are never POSTed unless `SERVICETITAN_CRM_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `SERVICETITAN_CRM_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
servicetitan-crm-pp-cli profile save briefing --json
servicetitan-crm-pp-cli --profile briefing booking-provider-tags get mock-value mock-value
servicetitan-crm-pp-cli profile list --json
servicetitan-crm-pp-cli profile show briefing
servicetitan-crm-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `servicetitan-crm-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add servicetitan-crm-pp-mcp -- servicetitan-crm-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which servicetitan-crm-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   servicetitan-crm-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `servicetitan-crm-pp-cli <command> --help`.
