# Dub CLI Brief

## API Identity
- Domain: Link management, attribution, and affiliate programs
- Users: Marketing teams, growth engineers, developer advocates, affiliate managers
- Data profile: Links (high volume, mutable), analytics events (read-only time series), customers, tags, folders, domains, partners/commissions/payouts

## Reachability Risk
- None. Official API with official SDKs (TypeScript, Python, Go, Ruby, PHP). No 403/blocking issues found on GitHub. The only 403 issue (#1072) relates to self-hosting database config, not API access.

## Top Workflows
1. **Bulk link creation** — Marketing campaigns need 50-100 branded short links at once with tags, UTM params, and geo-targeting
2. **Analytics deep dive** — Query click/lead/sale analytics grouped by country, device, browser, referer, or time period
3. **Domain management** — Add custom domains, verify DNS, check availability
4. **Partner program management** — Create partners, track commissions, approve bounty submissions, view payouts
5. **Link lifecycle** — Create → tag → folder → track clicks/leads/sales → analyze → archive

## Table Stakes
- CRUD for all 14 resources (links, tags, folders, domains, customers, partners, commissions, payouts, bounties, embed tokens, QR codes, events, analytics, track)
- Bulk operations (create/update/delete up to 100 links)
- Upsert (idempotent link creation by URL)
- Analytics with groupBy (country, city, device, browser, os, referer, top_links, top_urls, trigger, timeseries)
- Event listing with filters
- QR code generation
- Domain registration and status checking
- Partner analytics and management

## Data Layer
- Primary entities: links (highest gravity — every operation touches them), tags, folders, domains, customers
- Sync cursor: page-based pagination (page + pageSize)
- FTS/search: links by URL, key, domain, tag; analytics by date range and dimension
- Analytics time series: click/lead/sale counts over time — ideal for local aggregation
- Customer-link relationships for attribution tracking

## Product Thesis
- Name: dub-pp-cli
- Why it should exist: Dub's API is comprehensive (47 operations, 14 resources) but the only CLI (dubco, 24 stars) covers just link creation. The MCP server covers 3 operations. No tool offers offline analytics queries, bulk campaign management from the terminal, partner program administration, or cross-entity search. A CLI with SQLite persistence enables campaign analytics that Dub's dashboard can't — historical comparisons, cross-link performance correlation, and offline reporting.

## Build Priorities
1. Full link lifecycle with bulk operations, upsert, geo-targeting, A/B testing flags
2. Analytics engine: retrieve, group, filter, with local SQLite time series for historical comparison
3. Domain management with DNS verification workflow
4. Tag and folder organization with batch assignment
5. Customer and partner management with commission tracking
6. Transcendence: campaign analytics, link health monitoring, click velocity alerts, partner leaderboards
