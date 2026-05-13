# ServiceTitan CRM Module CLI Brief

## API Identity
- **Domain:** Field-services CRM (customers, locations, contacts, leads, bookings, tags) for residential/commercial trades businesses running on ServiceTitan
- **Users:** Operations owners, CSRs (call-takers), dispatchers, and marketing/segmentation analysts at trades companies (HVAC, plumbing, electrical, well-drilling, roofing, etc.) — not API developers per se. They use ST CRM to keep customer records clean, capture leads from web/phone, link bookings to locations, and segment customers by tag for marketing or operational reporting.
- **Data profile:** ~86 endpoints across 10 resource families: Customers (15), Locations (17), Contacts (12), Bookings (12), Leads (9), ContactMethods (6), Export (6), BookingProviderTags (4), ContactPreferences (3), BulkTags (2). All paths are tenant-scoped (`/tenant/{tenant}/<resource>`). Composed auth: `apiKey` (`ST-App-Key` header) + OAuth2 `clientCredentials` (token URL `https://auth.servicetitan.io/connect/token`). Per-scope tokens by resource (`tn.crm.customers:r`, `tn.crm.bookings:w`, etc.).

## Reachability Risk
- **None.** Sibling JPM module print on 2026-05-12 confirmed the same composed-auth shape works against JKA tenant `848413091`: 196/198 live dogfood tests passed (99%). ServiceTitan is the platform of record for tens of thousands of trades businesses; the API is production-grade and stable.

## Top Workflows

The CRM module is the call-handling and customer-data backbone of the ST install. Five rituals dominate:

1. **New-lead intake (CSR ritual, multiple times/day).** Inbound call or web-form lead arrives → CSR pulls up the lead → adds contact methods (phone, email) → either matches to an existing customer (deduplication) or converts the lead into a new customer + location → optionally schedules a first booking. Today this is 3–5 ST Web UI screens; an agent-native CLI collapses it into one workflow with `--dry-run` previews.
2. **Customer 360 lookup (CSR + dispatch ritual).** Caller says "I'm at <address>" or gives a phone — CSR needs the linked customer record, every location they own, every active booking, every contact method, and any tags (HOA member, municipal account, VIP) inside 30 seconds. Today: ST search is finicky on partial names/phones, requires multi-screen drill-down. CLI: one `customers find` command joins customer + locations + bookings + contacts from local SQLite.
3. **Booking management (dispatch ritual).** Confirm/reschedule/cancel bookings; see all bookings at a location for the next 7 days; surface bookings that are missing required prep (no contact method, no special instructions). Today: ST UI booking calendar is the primary tool but lacks fast filters across location ownership.
4. **Tag-based segmentation for marketing/ops.** Tag customers and locations (e.g., `municipal`, `HOA`, `commercial-warranty`, `seasonal-followup`) and pull the segment list for export to email tooling, Sheets, or campaign planning. Today: ST UI exports require manual filtering; the CRM Export endpoints are documented but not used by CSRs because there's no script-friendly wrapper.
5. **Lead-followup audit (operations-owner ritual, weekly).** "Which leads created in the last 30 days have not been touched by a CSR? Which ones converted to customers vs went stale?" This requires joining leads + customer-creation timestamps + contact-method updates and is impossible in the ST Web UI without an export-to-Excel pivot.

## Table Stakes

The CLI must match every feature any competing tool offers for ST CRM:

- All 86 endpoints exposed as commands (the heavy ServiceTitan MCP that we are replacing exposes them as raw MCP tools — we expose them as CLI commands AND as 2 MCP intent tools via the Cloudflare pattern).
- List/get/create/update for each of: customers, locations, contacts, leads, bookings, booking-provider-tags, contact-methods, contact-preferences.
- Bulk tag operations (BulkTags resource).
- Export endpoints with cursor-based pagination (Export resource — used for downstream sync to BI tools).
- `--json`, `--select`, `--csv`, `--dry-run`, typed exit codes on every command.

## Data Layer
- **Primary entities:** customers, locations, contacts, leads, bookings, tags, contact_methods, contact_preferences.
- **Relationships (the source of transcendence value):** customer→locations (1:N), location→bookings (1:N), customer→contacts (1:N), contact→contact_methods (1:N), customer↔tag (M:N), location↔tag (M:N), lead→customer (0..1, on conversion).
- **Sync cursor:** ST CRM endpoints support `modifiedOnOrAfter` filter on most list ops; combined with the dedicated Export endpoints (which return cursor tokens) for incremental sync.
- **FTS/search:** customer.name, customer.email, location.address, contact.email, lead.summary, tag.name. SQLite FTS5 over these fields enables sub-100ms typeahead lookup that the ST Web UI cannot match.

## Codebase Intelligence
- **Source:** ServiceTitan publishes per-module OpenAPI 3.1 specs (the user has 25 of them locally including this one); no public SDK. The "ServiceTitan MCP" exposes 400+ tools per turn across all modules — that's the heavy-MCP Pierce is replacing per-module with `pp-cli` + `pp-mcp` binaries.
- **Auth:** Composed. Dual headers: `ST-App-Key: <static_key>` AND `Authorization: Bearer <oauth2_token>`. Token endpoint: `POST https://auth.servicetitan.io/connect/token` with `grant_type=client_credentials&client_id=$ST_CLIENT_ID&client_secret=$ST_CLIENT_SECRET`. Token TTL ~30 min. Per-scope authz: `tn.crm.customers:r`, `:w`, etc. JKA tenant has all CRM scopes provisioned.
- **Data model:** Tenant-scoped multi-tenancy (every path includes `/tenant/{tenant}/`). Customer is the root entity; locations belong to customers; bookings belong to locations; contacts belong to customers (with contact methods + preferences as sub-resources). Leads are a pre-customer entity that converts on demand. Tags are M:N attachments to customer or location.
- **Rate limiting:** ServiceTitan enforces rate limits per integration (typically 120 req/min/tenant on the standard tier). The JPM print confirmed the generated client's adaptive limiter handles 429s gracefully.
- **Architecture:** Pure REST over JSON. Tenant id is path-positional, not header. All list endpoints support `page`/`pageSize` (cursor-based for Export). Dates are ISO-8601 in the API, `time.Time` in the generated client.

## Source Priority

Single source — no priority gate.

## Product Thesis
- **Name:** `servicetitan-crm-pp-cli` (binary), `servicetitan-crm` (library slug), `servicetitan-crm-pp-mcp` (MCP server binary).
- **Why it should exist:** The heavy ServiceTitan MCP costs ~400 tools of context per turn — every agent interaction with ST pays that token tax even when only one module is needed. A per-module `pp-cli` collapses a 86-endpoint module into 2 MCP intent tools (Cloudflare pattern: `<api>_search` + `<api>_execute`), cutting per-turn token cost by ~98% for the CRM-only use case. Plus the printed CLI gets offline SQLite, FTS5 search, `--json`/`--select`/`--csv` output, typed exit codes, and lead-followup/customer-360 transcendence commands the official API does not provide. Pierce can chain `servicetitan-crm-pp-cli customers find <phone>` into shell scripts and CSR macros that the ST Web UI cannot support.

## Build Priorities

1. **Foundation (Priority 0):** SQLite store schema for customers, locations, contacts, leads, bookings, tags, contact_methods, contact_preferences with proper FK relationships. Sync command that walks every list endpoint with `modifiedOnOrAfter` cursor.
2. **Absorbed (Priority 1):** All 86 endpoints as Cobra commands (auto-generated by the press), with composed-auth wired, all list endpoints supporting `--json`/`--select`/`--csv`/`--limit`. Apply the JPM-retro patch sweep (OAuth2 client.go, defaultSyncResources population, prefix-rename sweep, narrative example pattern).
3. **Transcendence (Priority 2):** Customer 360 lookup, lead-followup audit, tag-based segment export, location-bookings calendar, dedupe finder, and 2–4 more from the Step 1.5c.5 subagent.
