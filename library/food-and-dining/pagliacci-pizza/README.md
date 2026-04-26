# Pagliacci Pizza CLI

**Order Seattle's favorite pizza from the terminal — every endpoint, plus discount stacking, slice rotation across stores, and a local order history nobody else has.**

First and only CLI for the Pagliacci API. Browse menus and slice availability across all Seattle stores, build and price orders, manage your reward card and stored coupons, and replay past orders — all with offline search, agent-native output, and Chrome-cookie login (no manual token paste).

Learn more at [Pagliacci Pizza](https://pag-api.azurewebsites.net).

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/food-and-dining/pagliacci-pizza/cmd/pagliacci-pizza-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Authentication

Pagliacci has no public API and uses a custom composed `PagliacciAuth {customerId}|{authToken}` header constructed from cookies. Run `pagliacci-pizza-pp-cli auth login --chrome` while logged into pagliacci.com in Chrome — the CLI reads the auth cookies and constructs the header for you. No manual token paste required.

## Quick Start

```bash
# Log in by reading cookies from your active Chrome session
pagliacci-pizza-pp-cli auth login --chrome


# Sync stores, menu, slices, orders, rewards into the local SQLite store
pagliacci-pizza-pp-cli sync --full


# See what slices are available right now across every Seattle store
pagliacci-pizza-pp-cli slices today --agent


# Check your current reward balance and available rewards
pagliacci-pizza-pp-cli rewards card --json


# Review your last 5 orders
pagliacci-pizza-pp-cli orders list --limit 5 --json


# Re-create your most recent order as a priced cart, without sending
pagliacci-pizza-pp-cli orders reorder --last --dry-run

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local state that compounds

- **`slices today`** — See which Pagliacci slices are available right now at every Seattle store, sorted by proximity to your saved address.

  _When the agent is asked 'what slices can I get tonight?', this returns a single comparable list — no per-store iteration needed._

  ```bash
  pagliacci-pizza-pp-cli slices today --agent
  ```
- **`rewards stack`** — Compute the best application of stored coupons, reward redemption, and account credit for a given order total. Defaults to single-best-coupon + credit; multi-coupon stacking is flagged --experimental.

  _Agents helping with order placement can pick the optimal discount, not just the first valid coupon._

  ```bash
  pagliacci-pizza-pp-cli rewards stack --order-total 45.00 --agent
  ```
- **`orders summary`** — Aggregate order spend over a time range, with top items and store breakdown.

  _Agents helping with budgeting or reimbursement can produce a single roll-up report._

  ```bash
  pagliacci-pizza-pp-cli orders summary --since 90d --agent
  ```

### Time-aware composed lookups

- **`store tonight`** — List stores that are still open and can deliver to your saved address right now, sorted by ETA.

  _Late-night ordering: the agent only surfaces stores that will actually take the order._

  ```bash
  pagliacci-pizza-pp-cli store tonight --address-label home --agent
  ```
- **`address best-time`** — Resolve a saved address label to the next available delivery slot in one call.

  _Agents scheduling orders for a specific address don't need to discover the delivery zone separately._

  ```bash
  pagliacci-pizza-pp-cli address best-time --label home --agent
  ```

### Order workflows

- **`orders reorder`** — Re-create a past order as a fresh cart, with price revalidation since prices change. Add --send to also submit.

  _Agents can replay a routine order without rebuilding the cart line by line._

  ```bash
  pagliacci-pizza-pp-cli orders reorder --last --dry-run
  ```

## Usage

Run `pagliacci-pizza-pp-cli --help` for the full command reference and flag list.

## Commands

### account

Authentication and registration (no auth required for these endpoints)

- **`pagliacci-pizza-pp-cli account confirm_email`** - Confirm a new account by clicking the email-confirmation link's token
- **`pagliacci-pizza-pp-cli account create_token`** - Issue a session token (used internally by the SPA for token refresh)
- **`pagliacci-pizza-pp-cli account login`** - Authenticate with email/phone + password. Response sets customerId and authToken cookies.
- **`pagliacci-pizza-pp-cli account logout`** - Invalidate the current session
- **`pagliacci-pizza-pp-cli account password_forgot`** - Request a password reset email
- **`pagliacci-pizza-pp-cli account password_reset`** - Reset a password using a token from PasswordForgot email
- **`pagliacci-pizza-pp-cli account register`** - Create a new customer account

### address

Address validation and saved address book

- **`pagliacci-pizza-pp-cli address create`** - Create a new saved address
- **`pagliacci-pizza-pp-cli address delete`** - Delete a saved address
- **`pagliacci-pizza-pp-cli address get`** - Get a saved address by ID
- **`pagliacci-pizza-pp-cli address get_info`** - Get address info by saved ID
- **`pagliacci-pizza-pp-cli address list`** - List the authenticated user's saved addresses
- **`pagliacci-pizza-pp-cli address lookup`** - Validate an address and check delivery zone (returns store ID if deliverable)

### cart

Build and price an order before sending it

- **`pagliacci-pizza-pp-cli cart get_quote_building`** - Get the current cart/quote-building state by building ID
- **`pagliacci-pizza-pp-cli cart price_order`** - Compute the total price for an order (cart contents, taxes, fees, delivery) before sending
- **`pagliacci-pizza-pp-cli cart send_order`** - Submit an order. Requires payment information for guests; uses stored payment for authenticated users.
- **`pagliacci-pizza-pp-cli cart update_quote_building`** - Update cart contents (add/remove/modify items)

### credit

Account credit balance and entries

- **`pagliacci-pizza-pp-cli credit delete`** - Remove an account credit entry
- **`pagliacci-pizza-pp-cli credit get`** - Get a single credit entry
- **`pagliacci-pizza-pp-cli credit list`** - List the authenticated user's account credit entries

### customer

Customer profile and devices

- **`pagliacci-pizza-pp-cli customer access_devices_delete`** - Revoke a device's access to the account
- **`pagliacci-pizza-pp-cli customer access_devices_list`** - List devices that have access to this account
- **`pagliacci-pizza-pp-cli customer get`** - Get customer profile by ID
- **`pagliacci-pizza-pp-cli customer migrate_answer`** - Submit the answer to a migration question
- **`pagliacci-pizza-pp-cli customer migrate_question`** - Submit a security/migration question (legacy account migration flow)

### customer_feedback

Customer feedback submissions to Pagliacci

- **`pagliacci-pizza-pp-cli customer_feedback get`** - Get a feedback submission by ID
- **`pagliacci-pizza-pp-cli customer_feedback submit`** - Submit customer feedback (guest or authenticated)

### gifts

Stored gift cards, balance lookup, and transfer

- **`pagliacci-pizza-pp-cli gifts check`** - Check the balance of a gift card by ID and PIN (no auth required to check)
- **`pagliacci-pizza-pp-cli gifts delete`** - Remove a stored gift card from the account
- **`pagliacci-pizza-pp-cli gifts get`** - Get a single stored gift card by ID
- **`pagliacci-pizza-pp-cli gifts list`** - List the authenticated user's stored gift cards
- **`pagliacci-pizza-pp-cli gifts transfer`** - Transfer gift card balance to another account
- **`pagliacci-pizza-pp-cli gifts value`** - Get current value/balance of a saved gift card

### menu

Menus, slices, and product pricing

- **`pagliacci-pizza-pp-cli menu cache`** - Get the full menu (categories, products, prices, descriptions, images) for a store
- **`pagliacci-pizza-pp-cli menu product_price`** - Calculate the price for a customized product (size, toppings, modifiers)
- **`pagliacci-pizza-pp-cli menu slices`** - Get available slices across all stores for the current day (perishable, rotates daily)
- **`pagliacci-pizza-pp-cli menu top`** - Get featured top-of-menu items for a store

### orders

Order history and details

- **`pagliacci-pizza-pp-cli orders clone`** - Get order data shaped for re-ordering (transforms a past order into a new cart)
- **`pagliacci-pizza-pp-cli orders get`** - Get the full detail of a single past order (items, prices, store, time)
- **`pagliacci-pizza-pp-cli orders list`** - List the authenticated user's order history (paginated)
- **`pagliacci-pizza-pp-cli orders list_gift_cards`** - List orders that purchased gift cards
- **`pagliacci-pizza-pp-cli orders list_pending`** - List orders that are currently in flight (placed but not yet delivered/picked up)
- **`pagliacci-pizza-pp-cli orders suggestion`** - Get personalized order suggestions for a customer

### rewards

Loyalty card, rewards history, and stored coupons

- **`pagliacci-pizza-pp-cli rewards card`** - Get the authenticated user's reward card balance, points, and available rewards
- **`pagliacci-pizza-pp-cli rewards coupon_lookup`** - Look up a coupon by its serial number (validate before applying)
- **`pagliacci-pizza-pp-cli rewards history`** - Get reward earning/redemption history (most recent N entries)
- **`pagliacci-pizza-pp-cli rewards stored_coupons`** - List coupons saved to the authenticated user's account

### scheduling

Delivery and pickup time windows

- **`pagliacci-pizza-pp-cli scheduling slot_list`** - List available time-window slots for a store and service type
- **`pagliacci-pizza-pp-cli scheduling slot_list_for_date`** - List allowed slot times for a specific delivery/pickup date (YYYYMMDD)
- **`pagliacci-pizza-pp-cli scheduling window_days`** - List available delivery or pickup days for a store. serviceType is DEL (delivery) or PICK (pickup)

### store

Pagliacci store locations, hours, and quote info

- **`pagliacci-pizza-pp-cli store compute_quote`** - Compute a quote for a specific store with cart contents (returns Delivery, Drone, Pickup wait values)
- **`pagliacci-pizza-pp-cli store get`** - Get a single store by its numeric ID
- **`pagliacci-pizza-pp-cli store get_quote`** - Get quote-store metadata (delivery fee, drone status, pickup wait time) for a single store
- **`pagliacci-pizza-pp-cli store list`** - List all Pagliacci store locations with addresses, hours, GPS, amenities, and available slices
- **`pagliacci-pizza-pp-cli store list_quotes`** - List quote-store metadata (delivery fee, drone status, pickup wait) for all stores

### system

System information and announcements

- **`pagliacci-pizza-pp-cli system site_wide_message`** - Get site-wide announcement banner text (closures, holiday hours, etc.)
- **`pagliacci-pizza-pp-cli system version`** - Get the current API version


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
pagliacci-pizza-pp-cli address list

# JSON for scripting and agents
pagliacci-pizza-pp-cli address list --json

# Filter to specific fields
pagliacci-pizza-pp-cli address list --json --select id,name,status

# Dry run — show the request without sending
pagliacci-pizza-pp-cli address list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
pagliacci-pizza-pp-cli address list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Freshness

This CLI owns bounded freshness for registered store-backed read command paths. In `--data-source auto` mode, covered commands check the local SQLite store before serving results; stale or missing resources trigger a bounded refresh, and refresh failures fall back to the existing local data with a warning. `--data-source local` never refreshes, and `--data-source live` reads the API without mutating the local store.

Set `PAGLIACCI_NO_AUTO_REFRESH=1` to disable the pre-read freshness hook while preserving the selected data source.

Covered command paths:
- `pagliacci-pizza-pp-cli address`
- `pagliacci-pizza-pp-cli address get`
- `pagliacci-pizza-pp-cli address list`
- `pagliacci-pizza-pp-cli credit`
- `pagliacci-pizza-pp-cli credit get`
- `pagliacci-pizza-pp-cli credit list`
- `pagliacci-pizza-pp-cli customer`
- `pagliacci-pizza-pp-cli customer get`
- `pagliacci-pizza-pp-cli gifts`
- `pagliacci-pizza-pp-cli gifts get`
- `pagliacci-pizza-pp-cli gifts list`
- `pagliacci-pizza-pp-cli orders`
- `pagliacci-pizza-pp-cli orders get`
- `pagliacci-pizza-pp-cli orders list`
- `pagliacci-pizza-pp-cli store`
- `pagliacci-pizza-pp-cli store get`
- `pagliacci-pizza-pp-cli store list`

JSON outputs that use the generated provenance envelope include freshness metadata at `meta.freshness`. This metadata describes the freshness decision for the covered command path; it does not claim full historical backfill or API-specific enrichment.

## Use as MCP Server

This CLI ships a companion MCP server for use with Claude Desktop, Cursor, and other MCP-compatible tools.

### Claude Code

```bash
# Some tools work without auth. For full access, set up auth first:
pagliacci-pizza-pp-cli auth login --chrome

claude mcp add pagliacci pagliacci-pizza-pp-mcp
```

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "pagliacci": {
      "command": "pagliacci-pizza-pp-mcp"
    }
  }
}
```

## Health Check

```bash
pagliacci-pizza-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/pagliacci-pizza-pp-cli/config.json` (override with `--config` or `PAGLIACCI_CONFIG`).

Environment variables:
- `PAGLIACCI_CONFIG` — alternate config file path
- `PAGLIACCI_BASE_URL` — override the API base URL (used by mock-server tests)
- `PAGLIACCI_NO_AUTO_REFRESH` — disable the pre-read freshness hook

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `pagliacci-pizza-pp-cli doctor` to check credentials
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **auth login --chrome reports 'no auth cookies found'** — Open pagliacci.com in Chrome and log in. The CLI reads customerId and authToken cookies from the cookie store; if they're missing the session has expired.
- **401 Unauthorized on authenticated commands** — Run `pagliacci-pizza-pp-cli auth status`. If cookies are stale, log in again at pagliacci.com and re-run `auth login --chrome`.
- **Empty MenuSlices result during the day** — Slices rotate daily and may be sold out before close. The endpoint reflects current availability at request time.
- **store tonight returns no rows** — Stores have closed for the night. Use `store list` for the next-day delivery scope or check `scheduling window_days <storeId> DEL` for upcoming windows.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Discovery Signals

This CLI was generated with browser-captured traffic analysis.
- Target observed: https://pagliacci.com/
- Capture coverage: 26 API entries from 26 total network entries
- Reachability: standard_http (95% confidence)
- Protocols: rest_json (98% confidence)
- Auth signals: composed — headers: Authorization — cookies: customerId, authToken
- Generation hints: requires_browser_auth, composed_auth
- Candidate command ideas: store list — GET /Store returned full inventory of locations; menu top — GET /MenuTop/{storeId} drives the home menu UI; menu cache — GET /MenuCache/{storeId} returns the full menu; menu slices — GET /MenuSlices returns today's slices across all stores; address lookup — POST /AddressInfo validates an address and resolves a delivery store; address list — GET /AddressName returns saved addresses; orders list — GET /OrderList/{page}/{size} returns paginated history; orders get — GET /OrderListItem/{id} returns full detail

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
