# ServiceTitan Memberships CLI Brief

## API Identity
- **Domain:** ServiceTitan Memberships v2 — customer memberships, membership types (templates), recurring services (the active billing/visit services tied to a membership), recurring service events (scheduled occurrences), recurring service types (catalog), and invoice templates. The lifecycle module behind every annual care plan, maintenance subscription, and recurring-visit program JKA runs.
- **Users:** JKA Well Drilling ops/automation owner (Pierce). Same persona as the five sibling per-module ST CLIs already in the library — replaces the heavy 600+-tool general ST MCP with a focused module CLI an agent can load alone.
- **Data profile:** Tenant-scoped (`/tenant/{tenant}/...` on every path). Composed auth (static `ST-App-Key` header + OAuth2 client-credentials bearer). 30 operations across 6 resources — 23 GET + 4 POST + 3 PATCH, plus 7 export feeds. No DELETE: ST memberships are status-driven, not destroyed.
- **Servers:** `https://api.servicetitan.io/memberships/v2` (Production), `https://api-integration.servicetitan.io/memberships/v2` (Integration).

## Reachability Risk
- **None.** ServiceTitan is a tier-1 SaaS API. Five sibling CLIs (`servicetitan-crm`, `servicetitan-dispatch`, `servicetitan-inventory`, `servicetitan-jpm`, `servicetitan-pricebook`) are already shipped in `~/printing-press/library/` against this exact base host. All four ST auth env vars (`ST_APP_KEY`, `ST_CLIENT_ID`, `ST_CLIENT_SECRET`, `ST_TENANT_ID`) are present and confirmed in the user's environment. Phase 5 live read-only testing will succeed.

## Top Workflows
1. **Membership renewal pipeline** — find memberships whose `to` is within N days (or already past `to` with `status=Active`), surface them with `soldById`/`businessUnitId`/`paymentMethodId` + the matching `MembershipType` `durationBilling` next-step so the renewal task is one click away. The ST UI shows one membership at a time; an agent needs the whole pipeline as one list.
2. **Recurring service event scheduling** — list `recurring-service-events` with `status` ≠ Completed and `date` ≤ today (overdue) or upcoming inside a window. Mark events complete (`POST .../mark-complete`) or incomplete after a tech finishes, with a `jobId` link so revenue recognition stays consistent.
3. **Membership-type template drift** — every `Membership` is born from a `MembershipType` with N `recurringServices` entries. Membership lifecycle (cancellations, renewals, type changes) drifts the actual recurring services off the template. The ST UI does not surface this; locally we can join `memberships` × `membership-types` × `recurring-services` and report missing/extra services per member.
4. **Cancellation/churn risk audit** — memberships in `followUpStatus` ≠ None, near `to`, with no payment method, or with no completed recurring-service activity recently → likely churn targets. Joins three resources locally; no API call exposes this.
5. **Recurring revenue forecast** — local SQL roll-up of monthly recurring revenue from synced memberships × membership-type `durationBilling` entries, broken down by `businessUnitId` and `billingFrequency`. ServiceTitan Reporting can build this in the UI, but not in an agent-callable shape.

## Table Stakes
- Every endpoint produces a working Cobra command (30 operations after complex-body skips). Base CRUD (GET/POST/PATCH) must work end-to-end.
- Composed auth works first try: `auth` exchanges `ST_CLIENT_ID`/`ST_CLIENT_SECRET` for a bearer token, persists it, every call adds both `ST-App-Key` and `Authorization` headers; refresh on 401.
- `--json` / `--select` / `--csv` / `--limit` on every list command; `--dry-run` on every mutation (POST/PATCH).
- Local SQLite store for `memberships`, `membership-types`, `membership-type-discounts`, `membership-type-duration-billing`, `membership-type-recurring-services`, `recurring-services`, `recurring-service-types`, `recurring-service-events`, `invoice-templates`, `membership-status-changes` + `meta` for sync cursor.
- `doctor` confirms `ST_APP_KEY` + bearer token + `ST_TENANT_ID` + base URL.
- MCP surface (Cobra-tree mirror) for Claude Desktop / Code agents — the reason this per-module project exists.

## Data Layer
- **Primary entities (sync target):** `memberships`, `membership-types`, `recurring-services`, `recurring-service-types`, `recurring-service-events`, `invoice-templates`. Nested under membership-types: `discounts`, `duration-billing-items`, `recurring-service-items`. Per-membership: `status-changes`. Export feeds (invoice-templates, membership-status-changes, membership-types, memberships, recurring-service-events, recurring-service-types, recurring-services) are continuation-token feeds — `sync` should use them when present for incremental pulls.
- **Sync cursor:** `modifiedOnOrAfter` query param on every list endpoint; `modifiedOn` is on every entity. Export feeds use `from` + `continuationToken`. Store last-sync timestamp in `meta`.
- **FTS/search:** membership `customerId`/`importId`/`memo`, membership-type `name`/`displayName`, recurring-service `name`/`memo`/`jobSummary`, recurring-service-type `name`/`jobSummary`, invoice-template `name`, recurring-service-event `locationRecurringServiceName`/`membershipName`.
- **Snapshot table:** `membership_status_snapshots` — append `(membership_id, status, follow_up_status, active, from, to, next_scheduled_bill_date, modifiedOn, snapshot_at)` on every sync. This is what makes status-drift, expiring-soon, and risk transcendence features one-shot.

## Codebase Intelligence
- Source: 5 sibling generated CLIs in `~/printing-press/library/` (`servicetitan-crm`, `servicetitan-dispatch`, `servicetitan-inventory`, `servicetitan-jpm`, `servicetitan-pricebook`) + their manuscripts. Pricebook (shipped 2026-05-14 against v4.6.1) is the freshest reference template — same enriched spec shape, same composed-auth config, same `x-mcp` code-orchestration enrichment.
- Auth: composed (`apiKey` header + OAuth2 bearer). The enriched spec already carries `components.securitySchemes` with `x-auth-vars`: `ST_APP_KEY` (`per_call`), `ST_CLIENT_ID` + `ST_CLIENT_SECRET` (`auth_flow_input`), each noting whitespace is stripped defensively (memory: `feedback_credential_diagnostics`, `feedback_powershell_vs_bash_env`).
- Tenant: `x-tenant-env-var = ST_TENANT_ID` on the spec root. Every path is `/tenant/{tenant}/...` — the generated client injects the tenant ID from config.
- MCP: spec already carries `x-mcp` = `{ transport: [stdio, http], orchestration: code, endpoint_tools: hidden }` — the Cloudflare large-surface pattern is pre-applied. 30 endpoints + ~13 framework tools + ~12 novel commands ≈ ~55 tools, which keeps code orchestration appropriate.
- Rate limiting: ServiceTitan documents ~7,000 req/hr per environment per app key; `cliutil.AdaptiveLimiter` handles 429 backoff (sibling CLIs already use this).
- Scopes: spec lists granular OAuth scopes per resource (`tn.mem.memberships:r/w`, `tn.mem.membershiptypes:r`, etc.). The integration's client must have the read scopes for every list/get and the write scopes for `memberships:w`, `invoicetemplates:w`, `recurringservices:w`, `recurringserviceevents:w`.

## User Vision
From environment + auto-memory:
- **Replace the heavy general ST MCP** with per-module CLIs/MCPs so an agent doing membership work loads only membership tools.
- **Cut tokens** by exposing ~55 focused MCP tools (code-orchestrated) instead of the 600+ the general ST MCP advertises.
- **Same auth/store pattern** as the existing sibling CLIs so the JKA agent stack stays coherent.
- The memberships CLI should turn JKA's recurring-service lifecycle — renewal targeting, overdue-visit detection, template drift, recurring revenue rollups — into one-shot agent-native commands instead of UI clicking.

## Source Priority
- Single source (ServiceTitan Memberships v2 OpenAPI, user-provided enriched spec). No priority gate needed.

## Product Thesis
- **Name:** `servicetitan-memberships` (binary: `servicetitan-memberships-pp-cli`, MCP: `servicetitan-memberships-pp-mcp`)
- **Why it should exist:** Memberships are the recurring revenue spine of JKA's service business. Every `Membership` is a clock counting down to its `to` date, every `RecurringService` is an obligation to visit, every `RecurringServiceEvent` is either a kept or missed appointment. The ST UI shows one of these at a time; an agent needs them as joined sets. A per-module agent-native CLI with a local SQLite cache turns "which memberships expire this month and need renewal?" and "which recurring service events are overdue?" into one command, and slots into the existing per-module ST CLI family without loading the 600+-tool general ST MCP.

## Build Priorities
1. **Foundation** — config (tenant + composed auth), client (auto-inject `ST-App-Key` + bearer, refresh on 401, tenant substitution), store (the 10 cacheable entities listed under Data Layer + `meta` + `membership_status_snapshots`).
2. **Absorbed** — all 30 operations as Cobra commands. `sync` pulls cacheable entities via the export feeds where available, list endpoints otherwise. FTS5 search across cached entities.
3. **Transcendence** — novel features only this CLI can do because everything is in SQLite, composed-auth-aware, ST-memberships-shape-aware, and grounded in JKA's documented membership workflows. (See absorb manifest.)
