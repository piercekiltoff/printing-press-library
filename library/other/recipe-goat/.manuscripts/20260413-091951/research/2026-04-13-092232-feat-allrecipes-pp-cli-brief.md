# Allrecipes CLI Brief

## API Identity
- **Domain:** allrecipes.com — Dotdash Meredith property; recipe discovery + community ratings + saved recipes / meal planner / shopping list for logged-in users.
- **Users:** home cooks who collect recipes, plan meals, build grocery lists, and want to escape the ad-laden site UI.
- **Data profile:** Schema.org `Recipe` JSON-LD on every recipe page (stable, SEO-critical), plus authenticated XHRs for cookbook/meal-plan/shopping-list (to be discovered via sniff).

## Reachability Risk
- **Low–Moderate.** No Cloudflare Turnstile / aggressive bot-detection; standard UA + cookies works. `onzie9` reports anonymous bans after 10–15 rapid requests — **rate limiting exists**. recipe-scrapers v15.7.1 (May 2025) "simplified" the AR scraper, suggesting markup shifts but no breakage. Cookie-authed traffic looks like a logged-in user → safest path.

## User Vision
- User explicitly does NOT want a paid API. Build everything on the public website, browser-session auth (Chrome cookies), and Schema.org JSON-LD parsing.

## Top Workflows
1. **Search → save → cookbook/collection** (the core loop)
2. **Plan meals for the week** (meal planner calendar)
3. **Generate shopping list** from planned recipes
4. **Strip ads + scale + print** a single recipe (the #1 user complaint about AR)
5. **Recall what I cooked** (revisit ratings/reviews/cookbook history)

## Table Stakes
- Anonymous fetch of any recipe URL → clean JSON (every scraper does this)
- Recipe search with filters (ingredients, time, rating, cuisine)
- Schema.org JSON-LD extractor (universal across recipe sites — bonus)
- Print/markdown export of a recipe

## Data Layer
- **Primary entities:** `recipes` (id, slug, title, author, ingredients[], instructions[], times, yield, nutrition, rating), `cookbook_entries` (recipe_id, saved_at, collection), `collections` (id, name, owner), `meal_plan_entries` (date, meal, recipe_id), `shopping_list_items` (ingredient, qty, unit, source_recipe_id, checked), `reviews` (recipe_id, rating, body, posted_at).
- **Sync cursor:** `last_modified` per cookbook entry; meal-plan keyed by date range.
- **FTS5/search:** ingredients + title + tags; this enables the killer "what can I make with these 3 ingredients" command that AR's site UI cannot.

## Codebase Intelligence
- **Source signals (no DeepWiki run yet):**
  - `hhursev/recipe-scrapers` (Python, very active, dedicated AR parser): JSON-LD-first; explicit `_AbstractScraper.allrecipes` class. Treat as ground truth for stable HTML extraction.
  - `marcon29/CLI-dinner-finder-grocery-list` (Ruby, stale): closest-CLI prior art — search dinner recipes + grocery list. Narrow but proves the pattern.
  - `python-allrecipes` (mechanize-based) and `mdimec4/allrecipes` (Go) confirm anonymous-only paths; **no prior tool touches authenticated cookbook/meal-plan/shopping-list endpoints**.
- **Auth (anticipated):** session cookies (`SAR-AUTH`/similar) set after web login. Sniff will confirm exact cookie names + headers.

## Source Priority
- Single source: `allrecipes.com` (website itself).

## Product Thesis
- **Name:** `allrecipes-pp-cli`
- **Why it should exist:** No tool today combines (a) cookie-auth'd access to logged-in AR features (cookbook, meal planner, shopping list) with (b) CLI ergonomics (--json, --select, scripting, scaling, ad-free print output). All prior art is anonymous-read-only or full web app. We will absorb every scraper's read path, then transcend with the entire authenticated surface plus a local SQLite store that enables compound queries the AR website cannot answer (e.g., "what ingredients do I need this week" across the planner; "find a saved recipe under 30 minutes I haven't cooked in 60 days").

## Build Priorities
1. **Anonymous read path** — Schema.org JSON-LD extractor for any recipe URL/ID; cleanly returns title/ingredients/instructions/times/nutrition/rating. Beats every scraper on output ergonomics.
2. **Authenticated cookbook + collections** — list/add/remove/create. Net-new, only possible with cookie auth.
3. **Meal planner** — show/add/remove for a date range; transcendence: aggregate across week.
4. **Shopping list** — show/add/check-off/clear; transcendence: derive directly from a date range of meal plan.
5. **Search** — recipe search with filters + offline FTS over saved cookbook.
6. **Recipe scaling + print/markdown export** — the #1 user complaint about AR.

## Reachability Mitigations
- Default `--rate-limit` (≤2 req/sec); single-batch sync; cache scraped recipes in SQLite to avoid refetch.
- `auth login --chrome` extracts session cookies from local Chrome profile (the user is already logged in).
- `doctor` checks: cookie present, /me endpoint or cookbook XHR returns 200.
