# Absorb Manifest: ServiceTitan CRM pp-cli

## Source competitors (Phase 1.5a)

| Tool | URL | Lang | Surface count | Role |
|------|-----|------|---------------|------|
| Rowvyn enterprise MCP | https://mcpmarket.com/server/servicetitan | (hosted) | claimed 467 tools | Heaviest competing MCP — the one Pierce wants to replace per-module |
| BusyBee3333/servicetitan-mcp-2026-complete | https://github.com/BusyBee3333/servicetitan-mcp-2026-complete | TypeScript | 100+ tools | Open-source MCP, includes CRM tools (`list_customers`, `get_customer`, `list_locations`, `list_contacts`, `list_leads`, `list_bookings`) |
| glassdoc/servicetitan-mcp | https://github.com/glassdoc/servicetitan-mcp | Python | ~60 tools | MCP across all ST modules incl. CRM |
| elliotpalmer/servicepytan | https://github.com/elliotpalmer/servicepytan | Python | Python class methods | Most-cited Python wrapper for v2 API; covers CRM resources |
| Titanpy (PyPI) | https://pypi.org/project/Titanpy/ | Python | autogen handler library | Auto-generated wrapper covering CRM endpoints |
| servicetitan-pyapi (PyPI) | https://libraries.io/pypi/servicetitan-pyapi | Python | Python wrapper | Covers customers, contacts, locations |

**Universal gaps across all competitors:** no local SQLite store, no offline FTS over customer/contact data, no cross-entity SQL (customer→locations→bookings), no checkpointed `/export/*` runs, no lead-followup audit, no customer-360 join view, no tag-based segment export, no intent-level MCP surface.

## Absorbed (match or beat everything that exists)

Generator builds all 86 spec endpoints as typed Cobra commands. Each gets `--json/--select/--csv/--dry-run/--compact` plus typed exit codes. Local SQLite for sync targets.

| # | Family | Spec ops | Cobra commands (generator-produced) | Best Source | Our Implementation | Added Value |
|---|--------|---------|--------------------------------------|-------------|---------------------|-------------|
| 1 | Customers | 15 | `customers list/get/create/update`, `customers contacts list/create/update/delete`, `customers notes list/create/update/delete`, `customers tags add/remove/bulk` | Rowvyn, BusyBee3333 (`list_customers`, `get_customer`), glassdoc, servicepytan, Titanpy | Generated client, SQLite-backed list, FTS-indexed | Offline read after sync, JSON-out everywhere, agent-native exit codes |
| 2 | Locations | 17 | `locations list/get/create/update`, `locations contacts list/create/update/delete`, `locations notes list/create/update/delete`, `locations tags add/remove/bulk` | Rowvyn, BusyBee3333 (`list_locations`), glassdoc, servicepytan | Generated, SQLite-backed, FTS-indexed | `--dry-run` on every mutation, idempotent retry, agent-native |
| 3 | Bookings | 12 | `bookings list/get/create/update`, `bookings dismiss`, `bookings provider list/get/create/update` | Rowvyn, BusyBee3333 (`list_bookings`), glassdoc | Generated, SQLite-backed | Same as above |
| 4 | Contacts | 12 | `contacts list/get/create/update/delete`, `contacts methods list/get/create/update/delete`, `contacts preferences list/get/create/update` | Rowvyn, glassdoc, servicepytan | Generated, all CRUD | Offline lookup, JSON-out |
| 5 | Leads | 9 | `leads list/get/create/update`, `leads dismiss`, `leads convert` | Rowvyn, glassdoc, servicepytan | Generated, SQLite-backed | Lead conversion tracking in store |
| 6 | ContactMethods | 6 | `contact-methods list/get/create/update/delete` | Rowvyn, glassdoc | Generated | Cached locally |
| 7 | ContactPreferences | 3 | `contact-preferences list/get/update` | Rowvyn, glassdoc | Generated | Cached locally |
| 8 | BookingProviderTags | 4 | `booking-provider-tags list/get/create/update` | Rowvyn, glassdoc | Generated | Cached locally |
| 9 | BulkTags | 2 | `bulk-tags add/remove` | Rowvyn | Generated | `--stdin` batch input |
| 10 | Export | 6 | `export customers`, `export locations`, `export contacts`, `export leads`, `export bookings`, `export tag-changes` | Rowvyn (raw exports), servicepytan | Generated wrapper + checkpointed `--resume` flag | Resumable, dedup-on-insert, last-token persisted |

**Every row above is a feature we MUST build.** The generator handles ~95% mechanically; the remaining 5% is naming polish + the post-gen JPM-retro patch sweep (OAuth2 wire, prefix renames, sync resource list, narrative example pattern).

## Transcendence (only possible with our approach)

| # | Feature | Command | Score | How It Works | Evidence |
|---|---------|---------|-------|--------------|----------|
| 1 | Customer 360 find | `customers find <query>` | 10/10 | Local SQLite FTS5 over customer.name/email + location.address + contact methods, then joins customers→locations→bookings→contacts→tags in one query | Brief workflow #2 ("Customer 360 lookup ... 30 seconds"); Data Layer "FTS5 over these fields enables sub-100ms"; competitor gap "no customer-360 join view" |
| 2 | Lead-followup audit | `leads audit --since 30d` | 10/10 | Local join of leads + customer rows (creation timestamps) + contact_methods (modifiedOn) bucketed by untouched / converted / stale | Brief workflow #5 explicit ("impossible in the ST Web UI without an export-to-Excel pivot"); competitor gap "no lead-followup audit" |
| 3 | Tag segment export | `segments export <tag>` (with `--and-tag`, `--zone`, `--no-booking-since`) | 9/10 | Resolves boolean tag expression + filter predicates against local SQLite tag M:N tables, writes deterministic CSV/JSON | Brief workflow #4 ("ST UI exports require manual filtering"); competitor gap "no tag-based segment export" |
| 4 | Booking prep-audit | `bookings prep-audit --window 1d` | 9/10 | Local join: bookings in window WHERE linked location lacks confirmed contact_method OR lacks gate-code special_instruction OR missing required tag | Brief workflow #3 explicit ("surface bookings that are missing required prep"); Dan dispatcher persona frustration |
| 5 | Customer dedupe finder | `customers dedupe` (with `--by phone\|email\|address`) | 8/10 | Local SQL GROUP BY normalized phone/email/address-hash across customer rows; ranks duplicate clusters by overlap strength | Brief workflow #1 ("either matches to an existing customer (deduplication)"); CRM content pattern |
| 6 | Customer timeline | `customers timeline <id>` | 8/10 | UNION over local tables (customer_created, locations_added, bookings, tags_added, contact_methods_updated) ordered by timestamp | Brief workflow #2 implies escalation context; Maria CSR persona; relationships in Data Layer support it directly |
| 7 | Lead one-shot convert (with dry-run) | `leads convert <lead-id>` (with `--book`) | 7/10 | Wraps POST lead-convert + POST customer + POST location + optional POST booking; `--dry-run` resolves all ids first and prints diff; on commit, idempotent retry via client request id | Brief workflow #1 ("3-5 ST Web UI screens; an agent-native CLI collapses it"); explicit dry-run mention in brief |
| 8 | Sync status + incremental run | `sync status` / `sync run --since auto` | 7/10 | Reads last_modified_on per entity from local store, calls each list endpoint with `modifiedOnOrAfter=<that>`, upserts dedup, persists new high-water mark | Brief Codebase Intelligence ("modifiedOnOrAfter filter ... combined with Export endpoints"); Pierce ops-owner ritual |
| 9 | Stale-customer scan | `customers stale --no-activity 365d` | 6/10 | Local join: customers WHERE max(bookings.modifiedOn, contacts.modifiedOn, notes.modifiedOn) < now - N | Jess marketing persona; CRM domain pattern; weaker direct evidence in brief |
| 10 | Intent-tool MCP surface | (architectural — `mcp:` block in spec, not a Cobra command) | 10/10 | MCP server exposes 2 intent tools (`servicetitan_crm_search` + `servicetitan_crm_execute`) via Cloudflare pattern; the ~90 raw CRUD Cobra commands are marked `endpoint_tools: hidden` so agents don't pay for them | Brief Product Thesis: replace ~400-tool/turn heavy MCP. Module 2 of 25 proving the per-module pattern. JPM module print proved 2 tools instead of 66 raw mirrors. Pierce persona. |

**No stubs.** Every feature above ships fully. If implementation reveals an in-session blocker on any row, the manifest returns to Phase 1.5 for explicit re-approval per the no-mid-build-downgrade rule — never silently stubbed.

## Killed candidates (audit trail)

| Feature | Kill reason | Closest surviving sibling |
|---------|-------------|---------------------------|
| C5 Location bookings calendar | Tech-assignment enrichment requires JPM cross-module data; CRM-only version is a thin wrapper of `bookings list --location-id` | #4 Booking prep-audit |
| C8 Resumable export `--resume` | Already covered by generator + absorb manifest row #10 (Export); double-counting | #3 Tag segment export |
| C9 Multi-tenant profile switch | Config plumbing, not weekly for any named persona (Pierce runs one tenant); fails weekly-use check | #8 Sync status |
| C10 OAuth scope doctor | Run-once-per-setup, not a weekly ritual; fails weekly-use check | #8 Sync status |
| C14 Booking-provider performance report | Requires booking completion status from JPM module — out of CRM scope | #2 Leads audit |
| C15 Reverse-tag finder | Pure speculation; no evidence in brief or absorb manifest; Research Backing 0 | #9 Stale-customer scan |
| C16 NLP tag suggestion | LLM dependency per kill/keep checks; mechanical reframe collapses into FTS already in #1 | #1 Customer 360 find |

Full brainstorm artifact (personas + per-pass detail): `2026-05-12-143214-novel-features-brainstorm.md`

## Notes

- The intent-tool MCP surface (#10) is implemented via spec-level enrichment (`mcp:` block) rather than as a new Cobra command — same pattern proven on the JPM module, which collapsed 66 raw mirrors into 2 intent tools.
- `--resume` on export commands is part of the absorbed Export row (#10 of the absorbed table), not a separate transcendence feature — the generator already produces the checkpoint pattern.
- All 9 hand-built transcendence rows operate on the local SQLite store; no LLM dependencies, no external services, all auth fits within the same ST composed-auth shape used by the absorbed commands.
