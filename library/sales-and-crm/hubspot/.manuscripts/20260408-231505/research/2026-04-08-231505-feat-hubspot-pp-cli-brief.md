# HubSpot CLI Brief

## API Identity
- Domain: CRM, Marketing, CMS, Automation, Sales Engagement
- Users: Sales reps, RevOps teams, marketing ops, CRM admins, integration developers
- Data profile: 200+ official OpenAPI specs across 20 domains. Core CRM has unified CRUD pattern (list, get, create, update, delete, search, batch) across all object types. Bearer token auth via Private App access tokens. Rate limit: 100 req/10s (OAuth apps), 110 req/10s (private apps).

## Reachability Risk
- None. Official OpenAPI specs maintained by HubSpot. API available at api.hubapi.com. No reports of programmatic access issues.

## Top Workflows
1. **Deal pipeline management** - Track deals through stages, move them, report on velocity and conversion rates. This is the #1 power-user workflow in HubSpot.
2. **Contact/Company enrichment & hygiene** - Bulk update properties, merge duplicates, enforce naming conventions. RevOps spends hours on this weekly.
3. **Engagement logging** - Log calls, emails, meetings, tasks against contacts/deals. Sales reps need this fast.
4. **List building & segmentation** - Create/manage contact lists for campaigns. Marketing ops' bread and butter.
5. **Association management** - Link contacts to companies to deals to tickets. Understanding the relationship graph is critical.

## Table Stakes (from bcharleson/hubspot-cli - 55 commands across 11 groups)
- contacts: list, get, create, update, delete, search, merge
- companies: list, get, create, update, delete, search
- deals: list, get, create, update, delete, search
- tickets: list, get, create, update, delete, search
- owners: list, get
- pipelines: list, get, stages
- engagements: create-note, create-email, create-call, create-task, create-meeting, list, get, delete
- associations: list, create, delete
- lists: list, get, create, update, delete, add-members, remove-members, get-members
- properties: list, get, create, update, delete
- search: cross-object search

## MCP Ecosystem
- **Official HubSpot MCP Server** (beta) - Read-only access to contacts, companies, deals, tickets, carts, products, orders, line items, invoices, quotes, subscriptions
- **peakmojo/mcp-hubspot** - Community MCP with vector search, caching. Covers contacts, companies, conversations. Uses HUBSPOT_ACCESS_TOKEN env var, Bearer auth

## SDK Wrappers
- **@hubspot/api-client** (npm) - Official Node.js SDK v13.5.0, 135+ dependents
- **hubspot-api-client** (PyPI) - Official Python SDK v12.0.0
- **clarkmcc/go-hubspot** (Go) - Fully-featured OpenAPI-generated Go client
- **belong-inc/go-hubspot** (Go) - CRM-focused Go client

## Data Layer
- Primary entities: contacts, companies, deals, tickets, tasks, notes, calls, emails, meetings, owners, pipelines, pipeline_stages, lists, associations, properties
- Sync cursor: HubSpot objects have `updatedAt` timestamps. Incremental sync via search API with `lastmodifieddate` filter.
- FTS/search: HubSpot has server-side search, but offline FTS across all entities enables cross-object correlation the API can't do (e.g., "find all deals where contact email contains @acme.com AND ticket status is open")

## Product Thesis
- Name: hubspot-pp-cli
- Why it should exist: The official HubSpot CLI only covers CMS dev tools (design manager, serverless functions). There is NO official CLI for CRM operations. bcharleson/hubspot-cli covers 55 commands but lacks offline data, cross-object joins, pipeline analytics, and agent-native features. HubSpot power users (RevOps, sales managers) need command-line access to CRM data for automation, reporting, and bulk operations that the web UI makes painful. With 200+ API endpoints synced to SQLite, this CLI can answer questions ("which deals are stuck?", "who hasn't been contacted in 30 days?") that require multiple web UI clicks or custom HubSpot reports.

## Build Priorities
1. Core CRM CRUD for contacts, companies, deals, tickets (table stakes)
2. Pipeline & deal stage management with velocity tracking
3. Engagement logging (tasks, notes, calls, emails, meetings)
4. Full sync to SQLite with incremental updates
5. Cross-object search and association traversal
6. Transcendence: pipeline analytics, stale deal detection, contact coverage gaps, engagement velocity
