# X (Twitter) CLI Brief

## API Identity
- **Domain:** Social network — short-form posts ("tweets"), follower graphs, search, lists, bookmarks, DMs
- **Users:** Power posters, growth marketers, OSINT researchers, agent developers, journalists, content moderators
- **Data profile:** High-cardinality. Users have thousands of tweets, thousands of followers/following. Heavy graph data (follow edges, mutes, blocks, lists). Time-series engagement data.

## Reachability Risk
- **Medium-Low.** `x.com` returns 200 via plain HTTP. `api.twitter.com` is Cloudflare-protected (`browser_clearance_http`), but cookie capture from a logged-in Chrome session works around this — the user has confirmed a browser session, which is the canonical unlock.
- The `fa0311/twitter-openapi` spec (10,966 lines, last updated 2026) is the de-facto community spec for the internal GraphQL API used by the web client. Maintained by an active maintainer (181★, 448 commits). Stability of any reverse-engineered API is "breaks every 2-4 weeks" in worst cases — but the maintainer is responsive and the spec is comprehensive.

## Auth Profile
- **Primary auth: cookie-based** (`auth_token` + `ct0` CSRF + `guest_id`). Captured from a logged-in browser via `auth login --chrome`.
- **Required headers**: `x-csrf-token` (must match `ct0` cookie), `Authorization: Bearer <static-public-token>`, `x-twitter-client-language: en`, `x-twitter-active-user: yes`, `x-twitter-auth-type: OAuth2Session`.
- **Bearer token**: A well-known static public token used by all X web clients (`AAAAAAAAAAAAAAAAAAAAANRILgAAAAAA...`). Hardcoded in the CLI as a default, overridable via `X_TWITTER_BEARER_TOKEN`.
- **Alternative for posting/mutating**: V2 API key (paid tier $100+/mo). Most users will use cookie auth.
- **Alternative for posting/mutating (free)**: Cookie-based session can also POST (create tweets, follow/unfollow, etc.) since CSRF is included.

## Top Workflows
1. **Audience Hygiene** — "Who am I following that doesn't follow me back?" → surface the asymmetry, then optionally bulk-unfollow non-mutuals/inactives.
2. **Mutuals discovery** — "Who do I have a two-way relationship with?" Important for marketing, networking, OSINT.
3. **Engagement Tracking** — "Which of my recent tweets got the most likes/retweets/replies, and from whom?"
4. **Search + Save** — "Search for tweets about X, save them locally, run analytics offline." (Avoids rate limits on repeated queries.)
5. **List Management** — Create/manage lists, add/remove members, view list timelines.
6. **Bookmark organization** — Sync bookmarks locally, search them, tag them, export to markdown.
7. **OSINT lookup** — Profile a user (bio, follower count history, post frequency, engagement rate, mutuals with another user).

## Table Stakes (every competitor has these)
- Post tweet (text, media, threads)
- Search tweets (recent, top, latest)
- Get user profile, get user tweets
- Follow/unfollow, like/unlike, retweet/unretweet
- Bookmark add/remove/list
- DM send
- Trends list

## Data Layer
- **Primary entities**: `users`, `tweets`, `follows` (edge: source_id → target_id, scraped_at), `bookmarks`, `lists`, `list_members`, `likes`, `retweets`, `replies`, `mutes`, `blocks`, `dms`, `trends_snapshots`, `search_results_snapshots`
- **Sync cursor**: GraphQL paginates via `cursor`/`bottom_cursor` strings. Store last cursor per resource per user.
- **FTS/search**: SQLite FTS5 over `tweets.full_text`, `users.bio`, `users.name`, `dms.text`. Allows offline regex/substring queries.
- **Snapshots over time**: `follow_snapshots` table stores `(user_id, target_id, scraped_at)` so we can compute "who unfollowed me" by diffing snapshots.

## Codebase Intelligence (DeepWiki / fa0311 ecosystem analysis)
- **Source: `fa0311/twitter-openapi`**: Internal GraphQL API spec, regenerated from observed traffic. Operations grouped by resource: `Tweet`, `User`, `Bookmark`, `List`, `Search`, `Notification`, `Community`, `DM`.
- **Companion**: `fa0311/AwesomeTwitterUndocumentedAPI` — curated list of resources. `fa0311/TwitterInternalAPIDocument` — doc page.
- **Auth pattern (from twikit, trevorhobenshield, twitter-openapi-python)**: Cookie jar with `auth_token` + `ct0` + headers. The `ct0` cookie is the CSRF token mirrored into `x-csrf-token` header. Static bearer token is OK to hardcode; it's not user-secret.
- **Rate limiting**: GraphQL endpoints are rate-limited per session. Limits visible in `x-rate-limit-remaining` / `x-rate-limit-reset` headers. Need adaptive backoff.
- **Architecture insight**: Every GraphQL query has a query ID (e.g., `WkvtA3bNqcPg6EJDc8I9-w`) that rotates occasionally. The spec maintains current IDs. Need to allow override via env or config when IDs go stale.

## User Vision (from briefing)
- **Auth**: Logged-in browser session (cookie capture from Chrome).
- **Killer feature**: "Following but not following back details" — show me who I follow that doesn't follow me back, with rich detail (last active, follow ratio, bio, mutual connections), so I can decide who to unfollow.
- **Goal**: 90+ scorecard, novel features that provide real developer/power-user value.

## Competitive Landscape
| Tool | Type | Auth | Limitation |
|---|---|---|---|
| sferik/x-cli ("t") | Ruby CLI | V1 API key | V1 deprecated; tool is dormant |
| Infatoshi/x-cli | Go CLI | V2 paid key | Paid tier only ($100+/mo) |
| IndieHub25/x-cli | TUI (Ink) | V2 key | Interactive only, not scriptable |
| public-clis/twitter-cli | CLI | Cookie | Read-only, narrow features |
| twikit (Python) | Library | Username/password | Library, not CLI; PW auth fragile |
| trevorhobenshield/twitter-api-client | Python | Cookie | Powerful library, no CLI |
| Rettiwt-API | TS CLI/lib | Cookie | Has CLI but limited |
| Circleboom/Followerwonk/FollowerAudit | SaaS | OAuth | Paid web services, no CLI |
| jfullstackdev/twitter-x-unfollow-tool | Web | ZIP upload | Manual data export, no automation |
| EnesCinr/twitter-mcp | MCP | V2 paid key | Paid tier only |
| Various MCP servers | MCP | V2 key | All require paid tier |

**No existing CLI has all of: cookie auth (free) + local SQLite store + relationship analytics + agent-native output (`--json`, `--select`, exit codes) + MCP-ready.** This is the gap.

## Product Thesis
- **Name**: `x-twitter-pp-cli` (binary), invoked as `x-twitter`
- **Display name**: "X (Twitter)"
- **Headline**: "Use X from your terminal with your browser session — no paid API key, with a local SQLite database that powers relationship analytics no other tool offers."
- **Why it should exist**:
  1. The official V2 API has a $100/mo floor for anything useful. Cookie auth bypasses that.
  2. Existing free tools (twikit, twitter-api-client) are libraries, not CLIs. They have no local store and no novel analytics.
  3. Existing follower-analysis tools (Circleboom, FollowerAudit) are paid SaaS and require manual ZIP uploads or OAuth approval. None are CLI-native or agent-callable.
  4. No existing tool combines cookie auth + local SQLite + relationship analytics + MCP exposure.

## Build Priorities
1. **Cookie auth via browser** (`auth login --chrome` extracts `auth_token`, `ct0`, `guest_id` from Chrome cookie store). Without this, nothing else matters.
2. **Local SQLite store** with primary entities (users, tweets, follows, bookmarks, lists). FTS5 over text fields. Snapshots table for time-series.
3. **Sync commands** (`sync followers`, `sync following`, `sync tweets`, `sync bookmarks`, `sync lists`) with cursor-based pagination.
4. **Absorb pass**: Match every competitor's commands (post, search, follow, like, retweet, bookmark, DM, list management, trends).
5. **Transcendence pass** (the differentiators):
   - `relationships not-following-back` — the killer feature
   - `relationships mutuals` — two-way follows
   - `relationships ghost-followers` — followers who haven't tweeted in N days
   - `relationships overlap <user1> <user2>` — mutual followers between two users
   - `relationships unfollowed-me --since` — diff snapshots
   - `tweets engagement --top` — local SQL over synced tweets
   - `search saved --query <q> --since <t>` — search within local tweet store
   - `bookmarks export --format markdown` — clean exports
   - `audit inactive --days N` — find inactive accounts you follow
6. **Polish**: doctor, agent context, MCP exposure (auto from cobra tree), README, SKILL.md.

## Phase 4.85 Risk Watch
- "not-following-back" output must include real user data (handle, display_name, bio_excerpt, last_tweet_at) — not just IDs. Verify in dogfood.
- Follower sync at scale: a user with 10K following will hit rate limits. Need clear progress + `--limit` + resume from cursor.
- Cookie expiry: auth must surface "your session has expired, run `auth login --chrome` again" with clear messaging.
