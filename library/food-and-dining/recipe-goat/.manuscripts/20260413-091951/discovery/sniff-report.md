# Allrecipes / MyRecipes Sniff Report

**Run:** 20260413-091951
**Target:** allrecipes.com (user chose "the website itself"), authenticated via user's Chrome session.
**Tool chain:** agent-browser (session transfer) → browser-use v0.12.5 (headed Chrome via `--profile Default`) + fetch/XHR interceptor.

## Critical architecture finding

The logged-in product is on a **separate domain**: `www.myrecipes.com`, not `www.allrecipes.com`. Dotdash Meredith split the saved-recipes experience into a standalone SPA called **MyRecipes** that imports from AR and 1,500+ other recipe sites. The allrecipes.com site itself is now browse/search/read-only for anonymous users and does not have a logged-in UI.

**Consequence for the CLI:** the authenticated surface must target `www.myrecipes.com`. The CLI's anonymous read path targets `www.allrecipes.com` (and any other recipe-site URL, via Schema.org JSON-LD).

## Reachability: bot detection is hard

- **Headless Chrome (`browser-use` without `--headed`)** → HTTP 402 on every page with a Dotdash "access issue" message, across both domains. Confirmed on home page and recipe detail pages.
- **Headed Chrome (`browser-use --headed`)** → works. Full HTML, JSON-LD, Vue SPA loads. 5,806-byte body on home.
- **curl with valid session cookies + full browser UA + Origin/Referer/X-Requested-With/sec-fetch-* headers** → HTTP 403 on `POST /collections/getall`. 8,495-byte error page.
- **JS `fetch()` inside the live Chrome page** → 200 (observed for `/bookmarks/get` during real Vue app traffic).

This is almost certainly **TLS fingerprint / JA3 checking + Cloudflare bot scoring**. A plain Go `http.Client` will not reach authenticated endpoints. Mitigations:

1. **utls / cycleTLS** — impersonate Chrome's TLS ClientHello. Proven pattern for Cloudflare-fronted sites.
2. **Headless Chrome driver** — ship the CLI with an optional `chrome` backend for authenticated calls. Heavyweight.
3. **Scope reduction** — the CLI's happy path is the anonymous Schema.org JSON-LD extraction on any recipe URL; authenticated features are gated behind a `--browser` flag or documented as best-effort.

## Auth flow

- **Provider:** Keycloak at `auth.myrecipes.com`, realm `myrecipes`, client_id `myrecipes`.
- **Flow:** OIDC authorization code with JWT state. Redirect URI: `https://www.myrecipes.com/authentication/code-exchange?isMyrecipes=true`.
- **Login options observed:** Email, Facebook, Google.
- **Session cookies on `.myrecipes.com` after login:**
  - `myr_ddmsession` (httpOnly) — the active session token. ~31 years expiry (2057).
  - `myr_ddmaccount` — base64-encoded JSON: `{lastLogin, lastAuth, ddmsessionExpiry, hashId, version}`.
  - `ddmaccount`, `hid` — alternates used across properties.
  - Standard Cloudflare (`__cf_bm`, `_cfuvid`) and AWS ALB cookies on auth subdomain.
- **allrecipes.com gets its own companion cookies** after MyRecipes login (`myr_ddmaccount` copied across).

The user's separate allrecipes.com login did not authenticate them on myrecipes.com; two logins are required.

## Captured endpoints

**Host:** `https://www.myrecipes.com/`

| Method | Path | Fires on | Notes |
|---|---|---|---|
| POST | `/collections/getall` | favorites page initial load | Lists user's collections. FormData body (empty observed). Returns JSON. |
| POST | `/bookmarks/getall` | favorites page initial load, repeated per collection | Lists saved recipes. FormData body. Multiple fires (paged or per-collection). Returns JSON. |
| POST | `/bookmarks/get` | individual recipe quick-view | Gets specific bookmark(s). Needs params (empty call → 400). |

**Unobserved but documented in the SPA:**
- Create collection (button exists: `mm-myrecipes-manage-favorite__collections-cta-button`)
- Remove bookmark (`favorites-card__swipe-remove-button` observed, ~20 buttons present)
- Search (search nav link + dedicated `/search` route — uses public AR search UI, likely not auth-gated)
- Add-from-URL (from the "Save Recipes from 1,500+ Sites" affordance — suggests `/bookmarks/add` or similar)

The SPA does **not** expose meal planner or shopping list features in the web UI. Both concepts likely live only in the mobile AllRecipes Meal Planner iOS app (not sniffable here) or don't exist in the current product. This narrows the CLI's authenticated feature set to **favorites + collections**.

## Auth request signal

JS `fetch()` inside the page uses credentials: include and relies on cookie-based auth. No CSRF token observed in:
- meta tags (`meta[name=csrf-token]` — absent)
- localStorage / sessionStorage (no csrf/token/auth keys)
- document.cookie (no CSRF cookie visible to JS)

The server-side CSRF / bot check is opaque. `curl` replay with full cookie jar fails even with matching Origin/Referer/X-Requested-With headers. Chrome's real TLS fingerprint is the differentiator.

## Anonymous read path

Every allrecipes.com recipe page embeds Schema.org `Recipe` JSON-LD. This is SEO-critical and stable. Confirmed by `hhursev/recipe-scrapers` (Python, dedicated AR parser, v15.x active through 2025). This is the CLI's anchor for the read surface.

## Sniff decisions for generation

- **Anonymous read** → Schema.org JSON-LD parser against any AR URL (or any recipe site). No auth, no TLS impersonation needed for headed fetches. Expect occasional HTTP 402 on automated UAs; fall back messaging required.
- **Authenticated cookbook** → `POST /collections/getall`, `POST /bookmarks/getall`, `POST /bookmarks/get` on `www.myrecipes.com`. Requires TLS impersonation (utls recommended) or a browser backend. Document this as a capability gated by the user providing a valid session (via `auth login --chrome`).
- **No meal planner / shopping list** — drop from the absorb manifest. These are mobile-app-only, not web-sniffable, and building without captured traffic is speculative.

## Artifacts

- `session-state.json` — full Playwright state snapshot (will be redacted in archiving)
- `full-cookies.json` — exported cookies (will be redacted)
- `perf-entries.txt` — Performance API resource timing dump
