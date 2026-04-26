# Pagliacci CLI Brief (v2 — regenerate fresh on v2.3.6)

## API Identity
- Domain: Local pizza delivery & carryout (Seattle area, 8+ stores)
- Users: Seattle-area pizza enthusiasts, office lunch organizers, agents ordering on behalf of users
- Data profile: Stores (locations, hours, GPS, delivery zones, available slices), Menu (top items, slices, full cache, prices, product detail), Time windows (delivery/pickup days + slots), Orders (build, price, send, history), Account (rewards, stored coupons/credit/gifts, addresses), CMS (site-wide messages)
- API type: **Undocumented** — Angular SPA with two backends:
  - `pag-api.azurewebsites.net/api` — Custom Azure-hosted REST API (ordering, stores, menu)
  - `cdn.contentful.com` — Headless CMS (homepage, blog, loyalty info)
- Bundle source: `main.BPDM6VAU.js` (Angular main bundle) revealed 33 endpoints

## Auth (critical, learned from prior run)
- **Custom Authorization header scheme**: `Authorization: PagliacciAuth {customerId}|{authToken}`
- The Angular app reads `customerId` and `authToken` from cookies/storage and constructs the header client-side
- **Cookie replay alone returns 401** — must capture and replay the constructed Authorization header
- Login flow: POST `/Login` with username/password → response sets `customerId` and `authToken`
- Auth-required endpoints: `/OrderList`, `/RewardCard`, `/StoredCoupons`, `/StoredCredit`, `/StoredGift`, customer profile endpoints

## Reachability Risk
- **Low** — Prior run achieved 98% verify pass-rate (41/42) on first generation. No 403/bot detection observed. Azure-hosted REST API responds to standard HTTPS clients.

## Top Workflows (User-First)
1. **Order a pizza for delivery** — Find store → browse menu → customize pizza → check delivery zone → price order → place order → track. Most common goal.
2. **"What's available right now?"** — Today's slices across all 8+ stores (perishable; rotates daily). Unique to Pagliacci, no other tool surfaces this efficiently.
3. **Office lunch order** — Multiple addresses (split delivery), large pizzas, group rewards. Bulk-cart workflow.
4. **Reward & coupon awareness** — "Do I have a free pizza waiting?" — scan stored coupons, rewards balance, account credit, gift balances before deciding what to order.
5. **Reorder a favorite** — Pull last good order from history, re-price (prices change), re-send.

## Personas (User-First Discovery)
1. **The Seattle Regular** — Orders 1-2 times a week, has rewards built up, knows their favorite pizza by heart but wants to see what slices are available today.
2. **The Office Lunch Coordinator** — Orders weekly for the team, needs to split across colleagues' addresses (sometimes), needs receipts for reimbursement, cares about delivery windows.
3. **The Agent (AI)** — Ordering on behalf of a user. Needs structured output, idempotent operations, dry-run, precise error messages, and ability to verify cart before sending.
4. **The Hungry Late-Night User** — "Are any stores still open?" + "What slices are still available?" + "Closest store to me right now?" — time-sensitive, location-sensitive.

## Table Stakes (must match)
- Menu browsing (categories, prices, descriptions, images)
- Pizza customization (size, toppings, crust, half-and-half)
- Store locator with delivery zone validation
- Address autocomplete + validation
- Cart build → price → submit
- Order history
- Rewards/loyalty balance
- Login/logout

## Data Layer (sync targets for transcendence)
- Primary entities: Stores, MenuItems, MenuSlices (perishable, daily), TimeWindows, Orders, RewardCard, StoredCoupons, StoredCredit, StoredGift, AddressInfo
- Sync cursor: Menu changes weekly (cache OK); slices change daily (re-sync each session); orders append-only on auth.
- FTS/search: Menu item search by name/description, store search by address/zip, order search by item

## Codebase Intelligence (from prior bundle extraction)
- 33 API endpoints discovered from main.BPDM6VAU.js
- Auth: Custom `PagliacciAuth {customerId}|{authToken}` header constructed client-side
- Data model: Stores → MenuTop/MenuCache/MenuSlices per store; Cart in-flight via QuoteStore; Orders assoc to customerId
- Architecture: Angular SPA + Azure REST + Contentful CMS

## User Vision
> "I want to use the authenticated and unauthenticated commands in the cli, so I can login when you need me to. They do not have an official API so you'll need to browser sniff."

This run prioritizes:
- **Both auth and unauth surfaces** — equal first-class support, not auth as an afterthought
- **Browser session login** — `auth login --chrome` should work because the user has an active Chrome session at pagliacci.com
- **The full menu of authenticated features** — order history, rewards, stored coupons/credit/gifts, saved addresses

## Source Priority
- Single source: pagliacci.com (no combo CLI). The Multi-Source Priority Gate does not apply.

## Product Thesis
- Name: **pagliacci-pp-cli**
- Why it should exist: The first and only CLI for Pagliacci's ordering API. Ordering Seattle artisan pizza from the terminal was a stunt; ordering it with **today's slices snapshot, multi-store comparison, rewards-aware deal stacking, and a local order history** is a useful agent-native tool. The Pagliacci API is undocumented, has no SDK, has no MCP — this CLI is the only programmatic interface to it.

## Build Priorities
1. **Foundation** — Sync stores, menu, slices, time windows, customer profile, orders, rewards into local SQLite
2. **Public surface** — Menu browsing, store finder, slice availability, time windows, address validation
3. **Auth surface** — Login (cookie + custom header capture), order history, rewards, stored coupons/credit/gifts, address book
4. **Order workflow** — Cart build, price, send, verify; reorder from history
5. **Transcendence** — Today's slices snapshot across stores, multi-store menu diff, rewards-aware ordering, lunch coordinator multi-cart, store-of-the-night picker
