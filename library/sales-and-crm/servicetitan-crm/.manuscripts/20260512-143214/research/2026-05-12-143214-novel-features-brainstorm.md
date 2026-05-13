# Novel Features Brainstorm: ServiceTitan CRM pp-cli

> Subagent output from Phase 1.5c.5 (full audit trail). Survivors flow into the absorb manifest's transcendence table; the customer model and killed candidates are persisted here for retro/dogfood debugging per the novel-features-subagent contract.

## Customer model

**Persona 1: Maria, the Lead CSR (call-taker) at a 40-truck HVAC shop**

*Today (without this CLI):* Maria has the ServiceTitan Web UI open in two browser tabs (one for active call, one for searching). She has Outlook for confirmation emails, a sticky note pad for partial phone numbers callers half-remember, and the company's internal "VIP/HOA list" Google Sheet open on a second monitor. When a call comes in, she's pasting partial phone numbers into the ST search bar and watching the spinner. If the caller gives only a street name, she has to drill location-by-location. She cannot answer "how many open bookings does this customer have across all their properties" without clicking into each location separately.

*Weekly ritual:* Take 60-120 inbound calls per shift. For every one: identify the caller's customer record within ~30 seconds (or they get impatient), confirm location, see what's already on the books, and either book/reschedule/cancel or capture a new lead. Every Monday morning, she also reviews Friday/weekend voicemails-turned-leads and either matches them to existing customers or escalates.

*Frustration:* Partial-string lookup. ST search is exact-ish and slow; she retypes the same query 3 ways. Worse: when a caller gives a phone number and the customer has multiple locations, she can't see all of them on one screen with their booking status. The 30-second window blows past while she clicks.

**Persona 2: Dan, the Dispatcher at the same shop**

*Today (without this CLI):* Dan lives in the ST dispatch board and the booking calendar. He has a "tomorrow's prep" spreadsheet he hand-updates from ST exports each afternoon, flagging bookings missing contact info or special instructions. He cross-references the tech's truck inventory in his head. When a tech calls "I'm at this address but the gate code isn't in the notes," Dan is in three screens at once.

*Weekly ritual:* Daily — confirm tomorrow's bookings, flag the ones that are under-prepped (no callback number, no gate code, no special instructions, missing tag like `dog-on-property`). Weekly — pull next-7-days bookings by zone and by booking provider tag for the Monday capacity meeting.

*Frustration:* The ST UI shows bookings one at a time. To answer "which of tomorrow's 47 bookings are missing a confirmed contact method," he visually scans each one. There's no "show me bookings whose linked location has no confirmed phone" query. Pre-call prep gaps cause same-day cancellations.

**Persona 3: Pierce, the Operations Owner (the user himself)**

*Today (without this CLI):* Pierce runs JKA Well Drilling. He uses the heavy ST MCP through Claude Code today and pays the ~400-tool token tax every turn. For lead audits and tag-segment exports he writes ad-hoc scripts against the raw ST API or exports CSVs from the Web UI and pivots in Sheets. He has 25 ST OpenAPI specs locally and is mid-stream of building the per-module `pp-cli` swarm that this very brainstorm is for.

*Weekly ritual:* Monday morning lead-followup audit — "of the leads created in the last 30 days, which haven't been touched, which converted, which went stale?" Monthly — pull tag segments (`municipal`, `commercial-warranty`, `seasonal-followup`) for marketing handoff. Continuously — chains shell scripts and Claude agents that should be able to call CRM commands without paying MCP-tool tax.

*Frustration:* The ST Web UI cannot answer the lead-followup question without an Excel pivot. The heavy MCP can answer it but burns the entire context window doing so. He needs CLI commands that an agent can chain and a 2-tool MCP surface that doesn't cost 400 tools per turn.

**Persona 4: Jess, the Marketing/Segmentation Analyst (part-time at a multi-trade contractor)**

*Today (without this CLI):* Jess pulls customer lists from ST for email campaigns. She uses the ST UI's filter-and-export, then VLOOKUPs in Sheets to attach tag membership, then uploads to Mailchimp. The export is manual, paginated, and tag joins are fragile because ST UI export drops some tag fields.

*Weekly ritual:* Friday — pull the week's "new customers tagged `seasonal-followup`" list and hand it to the campaign manager. Monthly — full segment refresh per active campaign tag.

*Frustration:* Re-running the same export every week with the same fragile manual steps. No checkpointed/resumable export, so a session-timeout mid-export means starting over. No way to script "give me everyone tagged X who has at least one location in zone Y who hasn't had a booking in 90 days."

## Candidates (pre-cut)

| # | Name | Command | Description | Persona | Source | Verdict |
|---|------|---------|-------------|---------|--------|---------|
| C1 | Customer 360 join | `customers find <query>` | One command takes phone/email/name/partial address, returns customer + all linked locations + active bookings + contacts + tags from local SQLite via FTS5. | Maria | (a)+(c) | KEEP |
| C2 | Lead-followup audit | `leads audit --since 30d` | Joins leads, customer-creation events, contact-method updates locally; outputs untouched/converted/stale buckets with timestamps. | Pierce | (a)+(c) | KEEP |
| C3 | Tag segment export | `segments export <tag> [--and-tag X] [--zone Y] [--no-booking-since 90d]` | Resolves an arbitrary tag-AND-tag-AND-filter expression locally; emits CSV/JSON with stable schema. | Jess, Pierce | (a)+(c) | KEEP |
| C4 | Pre-prep booking audit | `bookings prep-audit --window 1d` | Lists tomorrow's bookings whose linked location is missing a confirmed contact method, gate code, or required tag. | Dan | (a)+(c) | KEEP |
| C5 | Location calendar | `locations bookings <location-id> --next 7d` | All bookings at a location across the next N days with assigned tech, status, prep-completeness flag. | Dan, Maria | (a) | KILLED — JPM cross-module |
| C6 | Dedupe finder | `customers dedupe [--by phone\|email\|address]` | Surfaces likely duplicate customer records (same phone across customer ids, same address across locations) for CSR merge review. | Maria | (b)+(c) | KEEP |
| C7 | Lead-to-customer conversion in one shot | `leads convert <lead-id> [--book]` | Wraps the lead→customer→location→optional-first-booking sequence with `--dry-run` preview, atomic, idempotent. | Maria | (a) | KEEP (borderline wrapper) |
| C8 | Resumable export with checkpoint | `export <resource> --resume` | Persists last cursor token to local store; re-running picks up exactly where it left off; dedup on insert. | Jess, Pierce | (f) | KILLED — already in absorb |
| C9 | Tenant-aware multi-profile switch | `crm --tenant jka` / config profiles | Manage multiple ST tenants in one config; commands resolve tenant id from profile name. | Pierce | (f) | KILLED — config plumbing |
| C10 | OAuth scope doctor | `crm doctor scopes` | Pings each `tn.crm.<resource>:r/:w` scope, reports which are provisioned vs missing on the configured client. | Pierce | (f) | KILLED — not weekly |
| C11 | Customer-360 timeline | `customers timeline <id>` | Chronological event stream from local store. | Maria, Pierce | (c) | KEEP |
| C12 | Sync delta report | `sync status` / `sync run --since auto` | Shows last-modified-on per entity family, runs incremental sync, reports row counts. | Pierce | (f) | KEEP |
| C13 | Stale-customer scan | `customers stale --no-activity 365d` | Local scan for customers with all activity older than N days. | Pierce, Jess | (c) | KEEP (borderline cadence) |
| C14 | Booking-provider performance | `booking-providers report --since 30d` | Conversion-to-completed rate per provider tag. | Pierce | (b)+(c) | KILLED — JPM cross-module |
| C15 | Reverse-tag finder | `customers untagged --segment <name>` | Customers who SHOULD be tagged but aren't. | Pierce | (c) | KILLED — speculative |
| C16 | NLP tag suggestion | `customers suggest-tags <id>` | LLM reads customer notes and suggests tags. | Jess | (b) | KILLED — LLM dependency |

## Survivors and kills

### Survivors

| # | Feature | Command | Score | How It Works | Evidence |
|---|---------|---------|-------|--------------|----------|
| 1 | Customer 360 find | `customers find <query>` | 10/10 (DF 3 + UP 3 + BF 2 + RB 2) | Local SQLite FTS5 over customer.name/email + location.address + contact methods, then joins customers→locations→bookings→contacts→tags in one query | Brief workflow #2 ("Customer 360 lookup ... 30 seconds"); Data Layer "FTS5 over these fields enables sub-100ms"; absorb manifest gap "no customer-360 join view" |
| 2 | Lead-followup audit | `leads audit --since 30d` | 10/10 (DF 3 + UP 3 + BF 2 + RB 2) | Local join of leads + customer rows (creation timestamps) + contact_methods (modifiedOn) bucketed by untouched / converted / stale | Brief workflow #5 explicit ("impossible in the ST Web UI without an export-to-Excel pivot"); absorb manifest gap "no lead-followup audit" |
| 3 | Tag segment export | `segments export <tag> [--and-tag X] [--zone Y] [--no-booking-since 90d]` | 9/10 (DF 3 + UP 3 + BF 2 + RB 1) | Resolves boolean tag expression + filter predicates against local SQLite tag M:N tables, writes deterministic CSV/JSON | Brief workflow #4 ("ST UI exports require manual filtering"); absorb manifest gap "no tag-based segment export" |
| 4 | Booking prep-audit | `bookings prep-audit --window 1d` | 9/10 (DF 3 + UP 3 + BF 2 + RB 1) | Local join: bookings in window WHERE linked location lacks confirmed contact_method OR lacks gate-code special_instruction OR missing required tag | Brief workflow #3 explicit ("surface bookings that are missing required prep"); Dan persona frustration |
| 5 | Customer dedupe finder | `customers dedupe [--by phone\|email\|address]` | 8/10 (DF 3 + UP 2 + BF 2 + RB 1) | Local SQL GROUP BY normalized phone/email/address-hash across customer rows; ranks duplicate clusters by overlap strength | Brief workflow #1 ("either matches to an existing customer (deduplication)"); CRM content pattern (unique to customer-master domain) |
| 6 | Customer timeline | `customers timeline <id>` | 8/10 (DF 3 + UP 2 + BF 2 + RB 1) | UNION over local tables (customer_created, locations_added, bookings, tags_added, contact_methods_updated) ordered by timestamp | Brief workflow #2 implies escalation context; Maria persona frustration; relationships in Data Layer support it directly |
| 7 | Lead one-shot convert (with dry-run) | `leads convert <lead-id> [--book]` | 7/10 (DF 3 + UP 2 + BF 2 + RB 0) | Wraps POST lead-convert + POST customer + POST location + optional POST booking; `--dry-run` resolves all ids first and prints diff; on commit, idempotent retry via client request id | Brief workflow #1 ("3-5 ST Web UI screens; an agent-native CLI collapses it"); explicit dry-run mention in brief |
| 8 | Sync status + incremental run | `sync status` / `sync run --since auto` | 7/10 (DF 2 + UP 2 + BF 2 + RB 1) | Reads last_modified_on per entity from local store, calls each list endpoint with `modifiedOnOrAfter=<that>`, upserts dedup, persists new high-water mark | Brief Codebase Intelligence ("modifiedOnOrAfter filter ... combined with Export endpoints"); Pierce ops-owner ritual |
| 9 | Stale-customer scan | `customers stale --no-activity 365d` | 6/10 (DF 2 + UP 2 + BF 2 + RB 0) | Local join: customers WHERE max(bookings.modifiedOn, contacts.modifiedOn, notes.modifiedOn) < now - N | Jess marketing persona; CRM domain pattern; weaker direct evidence in brief |

### Killed candidates

| Feature | Kill reason | Closest surviving sibling |
|---------|-------------|---------------------------|
| C5 Location bookings calendar | Tech-assignment enrichment requires JPM cross-module data; CRM-only version is a thin wrapper of `bookings list --location-id` | C4 Booking prep-audit |
| C8 Resumable export `--resume` | Already covered by generator (listed in absorb manifest as a transcendence row produced mechanically); double-counting | C3 Tag segment export |
| C9 Multi-tenant profile switch | Config plumbing, not weekly for any named persona (Pierce runs one tenant); fails weekly-use check | C12 Sync status (also config-adjacent but daily) |
| C10 OAuth scope doctor | Run-once-per-setup, not a weekly ritual; fails weekly-use check | C8 Sync status |
| C14 Booking-provider performance report | Requires booking completion status from JPM module — out of CRM scope; cross-module reach beyond brief | C2 Leads audit |
| C15 Reverse-tag finder | Pure speculation; no evidence in brief or absorb manifest; Research Backing 0 | C9 Stale-customer scan |
| C16 NLP tag suggestion | LLM dependency per kill/keep checks; mechanical reframe collapses into FTS already in C1 | C1 Customer 360 find |
