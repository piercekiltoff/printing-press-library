# servicetitan-memberships Absorb Manifest

ServiceTitan is a closed/proprietary ERP API; the competitive landscape is narrow. The only "competitors" are ST's own web UI/mobile, the giant general ST MCP that this per-module split replaces (web search confirms only general 60–100+-tool ST MCP servers exist — `glassdoc/servicetitan-mcp`, `BusyBee3333/servicetitan-mcp-2026-complete`, `JordanDalton/ServiceTitanMcpServer` — each bundling memberships inside a huge suite; no focused memberships CLI/MCP exists), and the five sibling generated CLIs in the user's library (`servicetitan-crm`, `servicetitan-dispatch`, `servicetitan-inventory`, `servicetitan-jpm`, `servicetitan-pricebook`). The absorb manifest matches every endpoint the ST UI exposes, then transcends with cross-entity SQLite queries and JKA-workflow-grounded commands the UI cannot answer.

## Absorbed (match or beat everything that exists)

30 operations across 6 resources + 7 export feeds. All ship as full Cobra commands; none as stubs.

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|------------|-------------------|-------------|
| 1 | List invoice templates | ST UI / general MCP | `invoice-templates list --json` | Offline cache, FTS, agent-native |
| 2 | Create invoice template | ST UI | `invoice-templates create --stdin --dry-run` | Dry-run, batch stdin |
| 3 | Get invoice template | ST UI | `invoice-templates get <id> --select` | --select compact output |
| 4 | Update invoice template | ST UI | `invoice-templates update <id> --dry-run` | Dry-run, idempotent |
| 5 | Export invoice templates | ST general MCP | `export invoice-templates --since <iso>` | Continuation-token state, --json |
| 6 | List membership types | ST UI / general MCP | `membership-types list --json` | Offline cache |
| 7 | Get membership type | ST UI | `membership-types get <id>` | --select |
| 8 | List type discounts | ST UI | `membership-types discounts <id> --json` | Offline cache |
| 9 | List type duration-billing | ST UI | `membership-types duration-billing <id> --json` | Offline cache |
| 10 | List type recurring-service items | ST UI | `membership-types recurring-services <id> --json` | Offline cache (the template) |
| 11 | Export membership types | ST general MCP | `export membership-types --since <iso>` | Continuation-token state |
| 12 | List memberships | ST UI / general MCP | `memberships list --active true --json` | Offline cache, FTS, --select |
| 13 | List membership custom-fields | ST UI | `memberships custom-fields --json` | Offline cache (lookup) |
| 14 | Sell a membership | ST UI | `memberships sale --stdin --dry-run` | Dry-run, batch stdin |
| 15 | Get membership | ST UI | `memberships get <id> --select` | --select compact |
| 16 | Update membership | ST UI | `memberships update <id> --dry-run` | Dry-run |
| 17 | List membership status changes | ST UI | `memberships status-changes <id> --json` | Offline cache |
| 18 | Export memberships | ST general MCP | `export memberships --since <iso>` | Continuation-token state |
| 19 | Export membership status changes | ST general MCP | `export status-changes --since <iso>` | Continuation-token state |
| 20 | List recurring-service events | ST UI / general MCP | `recurring-service-events list --json` | Offline cache, --select |
| 21 | Mark event complete | ST UI | `recurring-service-events mark-complete <id> --dry-run` | Dry-run, --job-id link |
| 22 | Mark event incomplete | ST UI | `recurring-service-events mark-incomplete <id> --dry-run` | Dry-run |
| 23 | Export recurring-service events | ST general MCP | `export recurring-service-events --since <iso>` | Continuation-token state |
| 24 | List recurring-service types | ST UI | `recurring-service-types list --json` | Offline cache |
| 25 | Get recurring-service type | ST UI | `recurring-service-types get <id>` | --select |
| 26 | Export recurring-service types | ST general MCP | `export recurring-service-types --since <iso>` | Continuation-token state |
| 27 | List recurring services | ST UI / general MCP | `recurring-services list --json` | Offline cache, FTS |
| 28 | Get recurring service | ST UI | `recurring-services get <id> --select` | --select compact |
| 29 | Update recurring service | ST UI | `recurring-services update <id> --dry-run` | Dry-run |
| 30 | Export recurring services | ST general MCP | `export recurring-services --since <iso>` | Continuation-token state |

All 30 ship as full implementations. No stubs. Plus framework-provided: `sync` (pulls cacheable entities into local SQLite + snapshots membership status history), FTS5 `search` across cached entities, `sql`, `doctor`, `auth`, `reconcile`, `stale`, `context`.

## Transcendence (only possible with our approach)

12 features, all scored ≥5/10. All grounded in JKA's documented recurring-service workflows and the rich membership lifecycle data model (37-prop memberships, 36-prop recurring services, 11-prop events). Each has a persona and a buildability proof. None ship as stubs.

| # | Feature | Command | Score | How It Works | Persona |
|---|---------|---------|-------|--------------|---------|
| 1 | Renewal pipeline | `renewals --within 30 --json` | 9/10 | Filters synced `memberships` where `to` is within N days AND `status` is Active AND no future renewal task, joins to `membership-types.durationBilling` for the next-step billing cadence, and emits `customerId/businessUnitId/soldById/to/duration` so the agent can open the right renewal task. No single API call returns this shape. | Pierce |
| 2 | Expiring soon (no auth) | `expiring --within 30 --json` | 8/10 | Like `renewals` but raw: every membership whose `to` falls inside a window, including already-cancelled, so renewal-outreach and lapse-recovery use one command. Local SQLite filter; the ST UI only shows one membership at a time. | Pierce |
| 3 | Overdue events | `overdue-events --json` | 9/10 | Joins synced `recurring-service-events` against `memberships.active` and `recurring-services.active`; flags events with `status` ≠ Completed and `date` ≤ today on still-active memberships. Surfaces what JKA should have already visited. | Pierce, dispatch |
| 4 | Upcoming events schedule | `schedule --within 14 --json` | 7/10 | Compact view of upcoming `recurring-service-events` grouped by date, location, membership; joins through `recurring-services` for `businessUnitId`/`preferredTechnicianIds`/`jobStartTime` so dispatch can pre-stage work. | Pierce, dispatch |
| 5 | Template drift audit | `drift --json` | 8/10 | For each active membership, compares its actual `recurring-services` against the `membership-types.recurring-service-items` template that birthed it; reports missing or extra services per member. The ST UI does not show this side-by-side. | Pierce |
| 6 | Churn risk | `risk --json` | 8/10 | Local-SQLite rule engine over memberships: `followUpStatus` ≠ None, `nextScheduledBillDate` past, no `paymentMethodId`, no completed event in N days, `to` inside lapse window. Each rule contributes a score; output is ordered by risk descending. Cross-entity; no API call gives this. | Pierce |
| 7 | Recurring revenue | `revenue --by month --json` | 8/10 | Local SQL roll-up of monthly recurring revenue from synced memberships joined to `membership-types.durationBilling`; breaks down by `businessUnitId` and `billingFrequency`. ST reporting can build this in the UI, not in an agent-callable shape. | Pierce |
| 8 | Memberships health summary | `health --json` | 8/10 | Aggregates renewals + overdue-events + drift + risk + stale-events + revenue-at-risk counts into one compact agent-shaped rollup — sized for agent priming. | Pierce, agent |
| 9 | Stale recurring services | `stale-services --months 6 --json` | 7/10 | Recurring services on active memberships whose last completed event was N+ months ago (or never), even though the recurrence window says a visit should have happened. SLA tracker the API does not expose. | Pierce |
| 10 | Bill preview | `bill-preview <membership-id> --json` | 7/10 | For a single membership, computes the next bill date and line-item amount by joining `memberships.nextScheduledBillDate` with the relevant `invoice-templates.items` resolved through `membership-types.durationBilling`. Answers "what is this customer about to be charged?" — no API call returns this directly. | Pierce |
| 11 | Event quick-complete | `complete <event-id> --job <job-id>` | 6/10 | Wrapper over `recurring-service-events mark-complete` that requires `--job` (the typical reason an event completes), prints a one-line confirmation, and updates the local snapshot so subsequent `overdue-events` filters reflect the change immediately. | Pierce, agent |
| 12 | Natural-language member finder | `find <description> --json` | 6/10 | Forgiving ranked search over synced memberships joined to `customers`-style importable context — `customerId`, `importId`, `memo`, `customFields[]`, `membershipType.name` — tuned for "the well-service plan we sold to the Smith family last spring" instead of exact IDs. Beyond framework `search`: domain-tuned ranking, tech-facing output shape. | Pierce, JKA office staff |

### Dropped prior features
First print — no prior research to reconcile.

### User additions at Phase 1.5 gate
None yet — gate has not run. The 12 above were assembled from the schema-grounded brief + sibling-CLI pattern + JKA recurring-service domain context (memory: ST module split, replace heavy MCP). Step 1.5c.5 novel-features subagent was **not** spawned this run because the brief already has explicit user vision and the JKA recurring-service domain is well-documented in auto-memory — flagging this here so the user can request a subagent pass at the gate if they want broader brainstorming.

## Why this is the GOAT for ServiceTitan Memberships

- **Coverage:** 30/30 ST Memberships v2 operations, no stubs. The general ST MCP advertises these too, but inside a 600+-tool bundle that dominates an agent's context window.
- **Local store + cross-entity joins + status history:** None of the 12 transcendence features can be answered from a single ST API call. Renewals/risk/drift/revenue all require joining 2-3 of {memberships, membership-types, recurring-services, recurring-service-events, invoice-templates, status-changes} locally. The `membership_status_snapshots` table is what makes status-drift and stale-services one-shot.
- **Grounded in JKA's documented workflows:** every transcendence feature traces to recurring-service-business reality — renewal pipeline, overdue visits, template drift, recurring revenue, churn risk. JKA runs care plans; the CLI surfaces what care plans actually need.
- **Composed auth done right:** Five sibling CLIs already prove the `ST_APP_KEY` + OAuth2 bearer + tenant-injection flow against the live ST API. The whitespace-stripping defense prevents the known JKA `invalid_client` gotcha.
- **Token-efficient MCP surface:** `x-mcp` enrichment (stdio+http transport, code orchestration, hidden endpoint tools) means the agent sees a thin search+execute pair over ~55 tools, not 600+.
