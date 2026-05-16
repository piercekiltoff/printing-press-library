---
name: pp-servicetitan-memberships
description: "A focused per-module ServiceTitan CLI for Memberships — sync once, then audit renewals, overdue events, template... Trigger phrases: `use servicetitan-memberships`, `run servicetitan-memberships`, `sync the membership book`, `which memberships are expiring`, `renewals due this month`, `overdue recurring service events`, `membership template drift`, `membership churn risk`, `recurring revenue rollup`, `preview the next bill for a member`, `find a member by description`, `membership health check`."
author: "user"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - servicetitan-memberships-pp-cli
---

<!-- GENERATED FILE — DO NOT EDIT.
     This file is a verbatim mirror of library/sales-and-crm/servicetitan-memberships/SKILL.md,
     regenerated post-merge by tools/generate-skills/. Hand-edits here are
     silently overwritten on the next regen. Edit the library/ source instead.
     See AGENTS.md "Generated artifacts: registry.json, cli-skills/". -->

# ServiceTitan Memberships — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `servicetitan-memberships-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install servicetitan-memberships --cli-only
   ```
2. Verify: `servicetitan-memberships-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

This CLI (servicetitan-memberships-pp-cli) mirrors every ServiceTitan Memberships v2 endpoint (customer memberships, membership types, recurring services, recurring service types, recurring service events, invoice templates, plus seven export feeds) and adds twelve novel commands that join across them in a local SQLite cache. It replaces the heavy general ST MCP for membership work with a thin search+execute MCP surface — so an agent doing renewal targeting, overdue-visit triage, churn risk scoring, or recurring-revenue rollups loads only the memberships module.

## When to Use This CLI

Reach for this CLI when an agent task touches ServiceTitan's recurring-revenue book — renewal targeting, overdue-event triage, template-drift audits, churn-risk scoring, recurring-revenue rollups, or bill previews for a single customer. It replaces the general ST MCP for any membership-shaped task, with a thin code-orchestrated MCP surface that loads only the memberships module. Use it instead of the ST UI when you need joined views across memberships, membership-types, recurring-services, and events that the UI can only show one entity at a time.

## Unique Capabilities

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

## Command Reference

**invoice-templates** — Manage invoice templates

- `servicetitan-memberships-pp-cli invoice-templates create` — Creates new invoice template
- `servicetitan-memberships-pp-cli invoice-templates get` — Gets invoice template specified by ID
- `servicetitan-memberships-pp-cli invoice-templates get-list` — Please note this endpoint does not allow to enumerate all invoice templates. Use the Customer Membership endpoint...
- `servicetitan-memberships-pp-cli invoice-templates update` — Updates specified invoice template in 'patch' mode

**membership-types** — Manage membership types

- `servicetitan-memberships-pp-cli membership-types get` — Gets membership type specified by ID
- `servicetitan-memberships-pp-cli membership-types get-list` — Gets a list of membership types

**memberships** — Manage memberships

- `servicetitan-memberships-pp-cli memberships customer-create` — Creates membership sale invoice
- `servicetitan-memberships-pp-cli memberships customer-get` — Gets customer membership specified by ID
- `servicetitan-memberships-pp-cli memberships customer-get-custom-fields` — Gets a list of custom field types that apply to customer memberships
- `servicetitan-memberships-pp-cli memberships customer-get-list` — Gets a list of customer memberships
- `servicetitan-memberships-pp-cli memberships customer-update` — Updates specified customer membership in 'patch' mode

**memberships-export** — Manage memberships export

- `servicetitan-memberships-pp-cli memberships-export invoice-templates` — Provides export feed for invoice templates
- `servicetitan-memberships-pp-cli memberships-export location-recurring-service-events` — Provides export feed for recurring service events
- `servicetitan-memberships-pp-cli memberships-export location-recurring-services` — Provides export feed for recurring services
- `servicetitan-memberships-pp-cli memberships-export membership-status-changes` — Provides export feed for customer membership status changes
- `servicetitan-memberships-pp-cli memberships-export membership-types` — Provides export feed for membership types
- `servicetitan-memberships-pp-cli memberships-export memberships` — Provides export feed for customer memberships
- `servicetitan-memberships-pp-cli memberships-export recurring-service-types` — Provides export feed for recurring service types

**recurring-service-events** — Manage recurring service events

- `servicetitan-memberships-pp-cli recurring-service-events <tenant>` — Gets a list of recurring service events

**recurring-service-types** — Manage recurring service types

- `servicetitan-memberships-pp-cli recurring-service-types get` — Gets recurring service type specified by ID
- `servicetitan-memberships-pp-cli recurring-service-types get-list` — Gets a list of recurring service types

**recurring-services** — Manage recurring services

- `servicetitan-memberships-pp-cli recurring-services location-get` — Gets recurring service specified by ID
- `servicetitan-memberships-pp-cli recurring-services location-get-list` — Gets a list of recurring services
- `servicetitan-memberships-pp-cli recurring-services location-update` — Updates specified recurring service in 'patch' mode


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
servicetitan-memberships-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Renewal pipeline for the month

```bash
servicetitan-memberships-pp-cli renewals --within 30 --agent --select id,customerId,to,membershipTypeId,soldById,businessUnitId
```

Active memberships whose to-date is within 30 days, narrowed to the fields a renewal-outreach agent actually needs.

### Overdue recurring-service events feeding a dispatch agent

```bash
servicetitan-memberships-pp-cli overdue-events --agent --select id,locationRecurringServiceName,membershipName,date,locationRecurringServiceId
```

Past-due events on active memberships with dotted-path field selection so the agent only sees what it needs to schedule.

### Compute monthly recurring revenue by business unit

```bash
servicetitan-memberships-pp-cli revenue --by month --agent
```

Local SQL roll-up over synced memberships joined to membership-types.durationBilling, broken down by business unit and billing frequency.

### Find a member without an ID

```bash
servicetitan-memberships-pp-cli find 'Smith well-care annual' --agent
```

Forgiving ranked search across memberships, importId, memo, custom fields, and membership-type name — useful when office staff describe a member.

### Complete an event and refresh the snapshot

```bash
servicetitan-memberships-pp-cli complete 99 --job 12345
```

Marks event 99 complete via mark-complete with a required job link, then updates the local snapshot so overdue filters reflect the change.

## Auth Setup

ServiceTitan uses composed auth: a static ST-App-Key header (set ST_APP_KEY) plus an OAuth2 client-credentials bearer token (set ST_CLIENT_ID and ST_CLIENT_SECRET). Run servicetitan-memberships-pp-cli auth login to walk the credential setup, then set the three credentials plus ST_TENANT_ID (your numeric tenant ID). Whitespace is stripped defensively on all three OAuth env vars — a known JKA gotcha that produces opaque invalid_client 400s when a trailing newline sneaks in from a copy-paste. The integration's OAuth client needs the read scopes for every list/get plus tn.mem.memberships:w, tn.mem.invoicetemplates:w, tn.mem.recurringservices:w, and tn.mem.recurringserviceevents:w for the mutation endpoints (memberships sale + update, invoice-templates create + update, recurring-services update, recurring-service-events mark-complete + mark-incomplete).

Run `servicetitan-memberships-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  servicetitan-memberships-pp-cli invoice-templates get mock-value mock-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
servicetitan-memberships-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
servicetitan-memberships-pp-cli feedback --stdin < notes.txt
servicetitan-memberships-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.servicetitan-memberships-pp-cli/feedback.jsonl`. They are never POSTed unless `SERVICETITAN_MEMBERSHIPS_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `SERVICETITAN_MEMBERSHIPS_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
servicetitan-memberships-pp-cli profile save briefing --json
servicetitan-memberships-pp-cli --profile briefing invoice-templates get mock-value mock-value
servicetitan-memberships-pp-cli profile list --json
servicetitan-memberships-pp-cli profile show briefing
servicetitan-memberships-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `servicetitan-memberships-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add servicetitan-memberships-pp-mcp -- servicetitan-memberships-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which servicetitan-memberships-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   servicetitan-memberships-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `servicetitan-memberships-pp-cli <command> --help`.
