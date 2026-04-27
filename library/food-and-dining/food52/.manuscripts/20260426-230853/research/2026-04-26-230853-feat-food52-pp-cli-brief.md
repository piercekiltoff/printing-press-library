# Food52 CLI Brief

## API Identity
- **Domain:** Recipe community + content publisher (Food52 LLC, founded 2009 by Amanda Hesser & Merrill Stubbs). Curated recipes, food blog/articles, community Hotline Q&A, and a home/kitchen Shop.
- **Users:** Home cooks looking for tested, well-rated, editorially curated recipes. Strong editorial voice (food-writer-driven vs algorithmic). Frequent recipe contests and community contributors.
- **Data profile:** Recipes (with structured ingredients/steps/times via schema.org JSON-LD), articles/blog posts, hotline questions and answers, contests, collections. No public REST API.

## Reachability Risk
- **HIGH** — Food52 deploys on Vercel and turns on **Vercel Bot Mitigation** site-wide. Every direct HTTP request returns `HTTP 429` with `x-vercel-mitigated: challenge` and an HTML "Vercel Security Checkpoint" page. This applies to:
  - `/` (homepage)
  - `/robots.txt`, `/sitemap.xml`
  - `/blog.rss` and any feed URL
  - `/recipes` and recipe detail pages
  - `/api/*` paths
  - `Googlebot` UA does not bypass; realistic Chrome UA + headers does not bypass
- **No public REST API.** `api.food52.com` 301-redirects to the homepage (and the homepage is challenged).
- **Implication:** A useful Printed CLI requires either (a) a one-time browser session to capture a Vercel **clearance cookie** (`_vcrcs`) that subsequent direct-HTTP requests can replay, or (b) on-demand cleared-browser fetches. Option (a) is the canonical Printing Press `browser_clearance_http` reachability mode and is compatible with the user's "unauthenticated only" constraint — clearance cookies are bot-detection state, not Food52 user authentication.

## Top Workflows (read-only, public surface)
1. **Search recipes by keywords** — "find me brownie recipes" → ranked results.
2. **Browse recipes by ingredient** — "what can I make with leeks?" → tagged recipes.
3. **Get a single recipe in agent-friendly form** — given a URL or slug, return structured ingredients, steps, times, ratings, author, comments count.
4. **Browse cuisines / meal types / collections** — Italian, Breakfast, Weeknight Dinners.
5. **Read articles / blog posts** — Food52 publishes editorial alongside recipes.
6. **Browse the Hotline Q&A** — community cooking questions and answers, often the best signal for "is this recipe forgiving?" type queries.
7. **(Read-only) browse the Shop** — products in the home/kitchen Shop, available unauthenticated.
8. **Sync favorites locally** — pull a list of recipes (search results, a collection page) and save offline for later cooking.

## Table Stakes (existing CLI: imRohan/food52-cli)
The only existing CLI is a **2018-era Ruby gem** that HTML-scrapes with hardcoded CSS selectors (`div.card__details h3 a`, `div.recipe__list--ingredients > ul > li`). Its features:
1. Search by keyword
2. Search by ingredient
3. Search by cuisine
4. Search by meal type (Breakfast/Brunch/Lunch/Dinner/Snacks)
5. Show single recipe ingredients + steps
6. List all ingredient tags
7. List all cuisine tags

It is **interactive only** (TTY prompts via `tty-prompt`), no `--json`, no flags, no offline cache, and almost certainly broken now since it makes plain HTTP requests with no Vercel clearance. Easy to surpass on every axis.

The other notable competitor is **`hhursev/recipe-scrapers`** (Python) — supports food52 as one of 200+ sites via a one-line subclass that relies on schema.org JSON-LD inheritance. It only covers single-recipe extraction, not search or browsing.

## Data Layer
- **Primary entities:**
  - `recipes` (id, slug, url, title, author, summary, ingredients[], steps[], prep_time, cook_time, total_time, servings, yield, rating, review_count, image_url, tags[], cuisine, meal_type, json_ld)
  - `articles` (id, slug, url, title, author, published_at, summary, body_html, tags[])
  - `hotline_questions` (id, slug, url, title, body, topic, asked_by, answer_count, asked_at)
  - `hotline_answers` (id, question_id, body, author, votes, answered_at)
  - `cuisines` and `ingredients` (taxonomy: name, slug, recipe_count) — cached browsable lookups
  - `shop_products` (id, slug, url, title, price, image_url, category, brand) — read-only catalog snapshots
- **Sync cursor:** Recipes and articles are slug-keyed; pagination via `?page=N`. Cursor-by-newest-published-at on /recipes/recent.
- **FTS/search:** SQLite FTS5 across recipes (title + summary + ingredients + tags) and articles (title + summary + body).

## User Vision
- **Unauthenticated only.** Do not explore or build commands that require sign-in (account, saved recipes, personal collections, contest submissions, comments, profile management).
- The CLI must be useful purely from the public read-only surface. Discovery, planning, and storage all happen locally.

## Source Priority
- Single source: `food52.com`. No combo CLI; the multi-source priority gate does not apply.

## Codebase Intelligence
- **Source:** `imRohan/food52-cli` (Ruby) — confirmed feature set above. No GitHub stars worth noting.
- **Auth:** None — the existing CLI does no auth and makes naked HTTP. This is consistent with our "unauthenticated only" constraint. Vercel clearance is the only "auth" the CLI will need, and it's a one-time browser handshake.
- **Data model:** Recipes carry full schema.org `Recipe` JSON-LD inline in the page `<script type="application/ld+json">`. Articles carry `Article` JSON-LD. Hotline questions are HTML-only with stable URL slugs.
- **Rate limiting:** Vercel mitigation is the practical limit — once cleared, a realistic Chrome fingerprint is unlikely to be re-challenged for hours. Be polite (1 req/s default, configurable).
- **Architecture:** The site is a Next.js / SSR-rendered app on Vercel. SSR pages contain the data the CLI needs in plain HTML and JSON-LD — no client-side hydration needed for read-only extraction.

## Product Thesis
- **Name:** `food52-pp-cli` (slug: `food52`).
- **Why it should exist:** The only existing Food52 CLI is interactive-only, JSON-less, agent-hostile, and broken against the current Vercel-protected site. There is no MCP server. Recipe discovery from the terminal — and especially from agents — is currently a "scrape-it-yourself" problem. A first-class CLI that exposes search, browse, and structured recipe extraction with `--json`, `--select`, offline FTS, and a local SQLite cache turns Food52 into a tool that fits into agent workflows: "find me 5 vegetarian weeknight recipes from Food52, give me the one with the highest rating, output as JSON."

## Build Priorities
1. **Foundation: HTTP transport with Vercel clearance.** Browser-clearance cookie import (`auth login --chrome` style flow), Surf/browser-compatible HTTP client with realistic Chrome fingerprint, polite rate limiting, response caching. Local SQLite store with FTS5.
2. **Absorb (match the Ruby CLI, beat it):** `search` (keywords/ingredients/cuisine/meal-type), `recipe get <slug-or-url>` (structured ingredients/steps/times via JSON-LD), `cuisines list`, `ingredients list`, `meal-types list`. All with `--json`, `--select`, `--limit`, agent-native exit codes.
3. **Absorb (extend beyond the Ruby CLI):** `articles search`, `articles get <slug>`, `hotline questions list/get`, `collections list`, `shop list/get` (all read-only), `recent` (newest recipes), `popular` (sorted by community rating).
4. **Transcendence:** `sync` (pull and FTS-index a slice of the catalog locally), `pantry` (local ingredient inventory → recipe matcher — "what can I make with X, Y, Z"), `pair` (find articles that reference a given recipe), `recipe scale --servings N` (scale ingredients via JSON-LD `recipeYield`), `cookalong` (guided step-by-step with timers). All work offline once the cache is populated.
5. **Polish:** README, SKILL.md, recipe `--md` output mode (Markdown for pasting into notes), `--print` mode (cleaned ingredients + steps for actually cooking from).
