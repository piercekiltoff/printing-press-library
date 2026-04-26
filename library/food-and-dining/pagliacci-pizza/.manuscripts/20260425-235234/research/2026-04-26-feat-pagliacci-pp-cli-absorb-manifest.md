# Pagliacci CLI Absorb Manifest (v2 — regenerate fresh on v2.3.6)

## Ecosystem Scan Results

| # | Tool | Type | Features | Status |
|---|------|------|----------|--------|
| 1 | (none found) | — | — | Greenfield: no community CLIs, MCP servers, Claude plugins, npm/PyPI wrappers |
| 2 | Prior pagliacci-pizza-pp-cli (v0.4.0, our own) | Internal | 22 absorbed, 0 transcendence | Reference for endpoint inventory; this run regenerates from scratch with composed auth + transcendence layer |

No external ecosystem to absorb. Pagliacci's API is undocumented and not widely reverse-engineered. The 33-endpoint inventory comes from JS bundle extraction + browser-sniff.

## Absorbed (every feature exposed by the API)

Each row is a feature the printed CLI must implement. The "Auth" column distinguishes commands that require a logged-in session (composed PagliacciAuth header) from anonymous commands.

### System
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 1 | API version | sniff /Version | `system version` | public |
| 2 | Site-wide announcement | sniff /SiteWideMessage | `system site-wide-message` | public |

### Stores
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 3 | List all stores | sniff /Store | `store list --json --select` | public |
| 4 | Get a store by ID | bundle /Store/{id} | `store get <id>` | public |
| 5 | List quote-store metadata | sniff /QuoteStore | `store list-quotes` | public |
| 6 | Get quote for a store | sniff /QuoteStore/{id} | `store get-quote <id>` | public |
| 7 | Compute quote with cart | sniff POST /QuoteStore/{id} | `store compute-quote <id> --stdin` | public |

### Menu
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 8 | Featured menu items | sniff /MenuTop/{storeId} | `menu top <storeId>` | public |
| 9 | Full menu | sniff /MenuCache/{storeId} | `menu cache <storeId>` | public |
| 10 | Today's slices across stores | sniff /MenuSlices | `menu slices` | public |
| 11 | Calculate product price | bundle POST /ProductPrice | `menu product-price --stdin` | public |

### Scheduling
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 12 | Available delivery/pickup days | sniff /TimeWindowDays/{store}/{type} | `scheduling time-window-days <storeId> <type>` | public |
| 13 | All available time windows | sniff /TimeWindows/{store}/{type} | `scheduling time-windows <storeId> <type>` | public |
| 14 | Slots for a specific date | sniff /TimeWindows/{store}/{type}/{date} | `scheduling time-windows <storeId> <type> <date>` | public |

### Address
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 15 | Validate address (delivery zone) | bundle POST /AddressInfo | `address lookup --stdin` | public |
| 16 | Get address info by ID | bundle /AddressInfo/{id} | `address get-info <id>` | public |
| 17 | List saved addresses | sniff /AddressName | `address list` | required |
| 18 | Get a saved address | bundle /AddressName/{id} | `address get <id>` | required |
| 19 | Create saved address | bundle POST /AddressName | `address create --stdin` | required |
| 20 | Delete saved address | bundle DELETE /AddressName/{id} | `address delete <id>` | required |

### Account / Auth
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 21 | Login (sets cookies) | sniff POST /Login | `account login` | public (creates session) |
| 22 | Logout | bundle POST /Logout | `account logout` | required |
| 23 | Register | bundle POST /Register | `account register --stdin` | public |
| 24 | Forgot password | bundle POST /PasswordForgot | `account password-forgot --stdin` | public |
| 25 | Reset password | bundle POST /PasswordReset | `account password-reset --stdin` | public |
| 26 | Confirm email | bundle GET /ConfirmEmail/{token} | `account confirm-email <token>` | public |
| 27 | Create token (refresh) | bundle POST /CreateToken | `account create-token` | required |

### Customer
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 28 | Get customer profile | sniff /Customer/{id} | `customer get <id>` | required |
| 29 | List access devices | bundle /AccessDevice | `customer access-devices list` | required |
| 30 | Revoke device access | bundle DELETE /AccessDevice/{id} | `customer access-devices delete <id>` | required |
| 31 | Submit migrate question | bundle POST /MigrateQuestion | `customer migrate-question --stdin` | required |
| 32 | Submit migrate answer | bundle POST /MigrateAnswer | `customer migrate-answer --stdin` | required |

### Cart / Quote Building
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 33 | Get current cart | sniff /QuoteBuilding/{id} | `cart get <buildingId>` | required |
| 34 | Update cart | bundle POST /QuoteBuilding/{id} | `cart update <buildingId> --stdin` | required |
| 35 | Price an order | bundle POST /OrderPrice | `cart price --stdin` | public |
| 36 | Submit an order | bundle POST /OrderSend | `cart send --stdin` | public (auth optional but uses stored payment when authenticated) |

### Orders
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 37 | List order history | sniff /OrderList/{page}/{size} | `orders list --limit --page` | required |
| 38 | Get order detail | sniff /OrderListItem/{id} | `orders get <id>` | required |
| 39 | List pending orders | sniff /OrderListPending | `orders list-pending` | required |
| 40 | List gift-card orders | bundle /OrderListGC | `orders list-gift-cards` | required |
| 41 | Clone (re-order) | bundle /OrderClone/{id} | `orders clone <id>` | required |
| 42 | Order suggestions | bundle /OrderSuggestion/{customerId} | `orders suggestion <customerId>` | required |

### Rewards
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 43 | Reward card balance | sniff /RewardCard | `rewards card` | required |
| 44 | Reward history | bundle /RewardHistory/{id}/{n} | `rewards history <customerId> <count>` | required |
| 45 | List stored coupons | sniff /StoredCoupons | `rewards stored-coupons` | required |
| 46 | Lookup coupon by serial | bundle /CouponSerial/{serial} | `rewards coupon-lookup <serial>` | public |

### Credit
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 47 | List stored credit | sniff /StoredCredit | `credit list` | required |
| 48 | Get stored credit entry | bundle /StoredCredit/{id} | `credit get <id>` | required |
| 49 | Delete stored credit | bundle DELETE /StoredCredit/{id} | `credit delete <id>` | required |

### Gifts
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 50 | List stored gift cards | bundle /StoredGift | `gifts list` | required |
| 51 | Get gift card by ID | bundle /StoredGift/{id} | `gifts get <id>` | required |
| 52 | Delete stored gift | bundle DELETE /StoredGift/{id} | `gifts delete <id>` | required |
| 53 | Check gift balance (public) | bundle /CheckGift/{id}/{pin} | `gifts check <id> <pin>` | public |
| 54 | Get gift current value | bundle /GiftValue/{id} | `gifts value <id>` | required |
| 55 | Transfer gift balance | bundle POST /TransferGift | `gifts transfer --stdin` | required |

### Feedback
| # | Feature | Source | Implementation | Auth |
|---|---------|--------|----------------|------|
| 56 | Submit feedback | bundle POST /Feedback | `feedback submit --stdin` | public |
| 57 | Get feedback by ID | bundle /Feedback/{id} | `feedback get <id>` | required |

**Total absorbed: 57 features mapping to 33 unique endpoints (most have list+get+create+delete variants).**

Every command supports: `--json`, `--agent`, `--select`, `--csv`, `--compact`, `--data-source`, `--dry-run`, agent-friendly exit codes.

## Transcendence (only possible with our approach)

User-first feature design driven by personas in the Brief.

**Scope cut 2026-04-26:** dropped T5 (stores cheapest), T7 (rewards forecast), T9 (menu drift), T10 (cart fanout) per user re-approval. Final set is 6 features.

| # | Feature | Command | Persona | Why Only We Can Do This | Score |
|---|---------|---------|---------|-------------------------|-------|
| T1 | Slices today across all stores | `slices today` | Seattle Regular, Late-Night User | Requires sync of all 8+ stores' MenuSlices into local SQLite, joined by store proximity. The web UI shows slices per-store; nobody surfaces them in one view. | 8/10 |
| T2 | Open stores tonight (close-of-day filter) | `stores tonight` | Late-Night User | Requires real-time TimeWindowDays + Store.OpenHour + delivery-zone resolution. The site shows all stores; we filter to "still open and can deliver to me." | 8/10 |
| T3 | Stack discounts (best single coupon + credit; multi-coupon flagged --experimental) | `rewards stack` | Seattle Regular, Agent | Requires joining StoredCoupons + RewardCard + StoredCredit and applying optimal-application logic. The site applies coupons one-at-a-time at checkout. | 8/10 |
| T4 | Reorder last / by ID | `orders reorder --last` or `orders reorder <id>` | Seattle Regular | Requires OrderListItem + OrderClone composed with cart construction + price re-validation (prices change). The site has a "reorder" button but exposes no batch/headless mode. | 7/10 |
| T6 | Address-aware delivery time picker | `address best-time --label home` | Late-Night User, Agent | Resolves saved address → store → next available TimeWindow slot in one call. | 7/10 |
| T8 | Spend summary (orders aggregation) | `orders summary --since 30d` | Seattle Regular, Office Coordinator | Aggregates OrderListItem prices over a time range. The site lets you scroll through history; we sum it. | 6/10 |

**Transcendence total: 6 features, all scoring >= 6/10, 3 scoring >= 8/10.**

## Stub policy
None. Every feature listed in this manifest is shipping scope. If implementation is infeasible during build, return to this gate per the rules.

## Summary
- **Absorbed:** 57 features (mapping to 33 unique endpoints, with list/get/create/delete variants where the API supports them)
- **Transcendence:** 6 user-first features with scores 6-8/10
- **Total:** 63 features
- **Best existing tool:** None (greenfield API)
- **Our advantage:** First and only CLI for the Pagliacci API, with composed cookie auth (`auth login --chrome`), local SQLite for offline + agent use, and 6 transcendence features no Pagliacci surface offers.

## Source Priority
Single source. The Multi-Source Priority Gate does not apply — only `pagliacci.com` is named.

## Auth Profile
- Type: `composed`
- Header: `Authorization`
- Format: `PagliacciAuth {customerId}|{authToken}`
- Cookie domain: `pagliacci.com`
- Required cookies: `customerId`, `authToken`
- Login mechanism: `auth login --chrome` reads cookies from active Chrome profile and writes to config. `auth set-token` accepts a paste of the constructed header for users who don't want to use Chrome.
