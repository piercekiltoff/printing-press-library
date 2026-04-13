# Pagliacci Sniff Report

## User Goal Flow
- Goal: Order a pizza for delivery
- Steps completed: 4 of 7 (homepage, menu browsing, locations, login page)
- Steps skipped: Cart add, checkout, account pages (auth required)
- Coverage: Public surface well-covered; auth surface identified but not exercised

## Pages & Interactions
1. Homepage (pagliacci.com/) — loaded, captured Contentful CMS calls
2. Menu/Order (pagliacci.com/order) — loaded, discovered Store, Menu, QuoteStore, TimeWindow APIs
3. Locations (pagliacci.com/locations) — loaded, confirmed Store + location data
4. Login (pagliacci.com/login) — form visible, username/password inputs
5. Loyalty (pagliacci.com/loyalty) — Contentful CMS page, no API calls

## Sniff Configuration
- Backend: browser-use (--profile "Default" for Chrome profile with cookies)
- Auth status: Session expired — login page shown despite Chrome profile loaded
- Effective rate: ~3 req/s

## Key Discovery
Pagliacci uses a dual-backend architecture:
1. **pag-api.azurewebsites.net/api** — Custom Azure-hosted REST API for ordering, stores, menu, pricing
2. **cdn.contentful.com** — Headless CMS for static content (homepage, blog, loyalty info)

The ordering API was also reverse-engineered from the Angular main bundle (main.BPDM6VAU.js), revealing 33 endpoints including authenticated ones.

## Endpoints Discovered
| Method | Path | Status | Auth |
|--------|------|--------|------|
| GET | /Version | 200 | public |
| GET | /Store | 200 | public |
| GET | /QuoteStore | 200 | public |
| GET | /QuoteStore/{storeId} | 200 | public |
| GET | /MenuTop/{storeId} | 200 | public |
| GET | /MenuCache/{storeId} | 200 | public |
| GET | /MenuSlices | 200 | public |
| GET | /TimeWindowDays/{storeId}/{type} | 200 | public |
| GET | /TimeWindows/{storeId}/{type}/{date} | 200 | public |
| GET | /SiteWideMessage | 200 | public |
| POST | /AddressInfo | known | public |
| POST | /Login | known | public (creates session) |
| GET | /OrderList | known | auth-required |
| POST | /OrderPrice | known | public (guest) |
| POST | /OrderSend | known | public (guest) |
| GET | /RewardCard | known | auth-required |
| GET | /StoredCoupons | known | auth-required |

## Coverage Analysis
- Stores: Complete — all 8+ locations with hours, GPS, amenities, available slices
- Menu: Complete — full menu with categories, prices, descriptions, images
- Ordering: API paths identified but not exercised (need cart + checkout interaction)
- Auth: Endpoint paths found in JS bundle but not validated (session expired)
- Rewards: Endpoint identified but requires auth

## Authentication Context
- Chrome profile loaded but Pagliacci session was expired
- Login form uses username/password → POST /Login
- Auth tokens likely stored in HttpOnly cookies or session storage (not localStorage)
- Auth-required endpoints: OrderList, RewardCard, StoredCoupons, StoredCredit, StoredGift
