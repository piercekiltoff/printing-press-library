# Allrecipes CLI — Absorb Manifest

## The honest constraint

Two sniffs, two pivots, one hard finding: **Dotdash Meredith actively blocks non-browser TLS fingerprints**. curl with valid session cookies returns 403 on every authenticated endpoint. Browser sniff confirmed the real endpoints exist and work from inside a live Chrome page, but a plain Go `http.Client` will not reach them.

crowd-sniff found nothing useful: every community "allrecipes" package on GitHub/npm/PyPI is an HTML scraper, not a REST client. There are no published endpoint strings to mine.

This shapes the build: **the CLI's spine is anonymous Schema.org JSON-LD extraction + a local SQLite cookbook**. The authenticated MyRecipes surface (favorites/collections) is shipped as an **experimental `--browser` mode** that spawns headed Chrome for its calls. Users can opt in; most won't need to.

---

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---|---|---|---|
| 1 | Extract structured recipe from AR URL | hhursev/recipe-scrapers (Python, active) | `recipe get <url>` — Schema.org JSON-LD parser, works for AR + any JSON-LD site | Single static Go binary, `--json`/`--md`/`--print` output, local caching |
| 2 | Extract by AR recipe ID | mdimec4/allrecipes (Go, stale) | `recipe get <id>` resolves `/recipe/<id>/` | --servings scaling, cached |
| 3 | Search AR by keyword | python-allrecipes, shaansubbaiah/scraper | `search <query>` — public AR search page + JSON-LD on result tiles | Offline FTS over cached results, `--rating >=4`, `--max-time 30m` |
| 4 | Filter by ingredient | Paprika (commercial) | `search --with "chicken,rice" --without shellfish` | Works offline on local cookbook |
| 5 | Print-friendly single recipe | RecipeStripper.com (manual site) | `recipe get <url> --print` | No ads, no life story, `--servings 6` scales |
| 6 | Markdown export | Mealie (web app) | `recipe get <url> --md` | Pipe to Obsidian, GitHub, etc. |
| 7 | Save recipe to local library | Paprika, Mealie | `save <url>` → SQLite | Offline, scriptable, bulk via stdin |
| 8 | List saved recipes | all managers | `cookbook` | `--json`, `--select`, pipeable |
| 9 | Search saved recipes | Mealie FTS | `cookbook search <q>` | FTS5 across title + ingredients + notes |
| 10 | Remove saved recipe | all managers | `cookbook remove <id>` | `--dry-run`, undo via audit log |
| 11 | Tag / categorize | Paprika | `cookbook tag <id> weeknight` | Arbitrary tags, tag search, tag exclude |
| 12 | Scale servings | Paprika, AR web UI | `recipe get <url> --servings 6` | Rational fraction math, unit-aware (e.g., "1 1/2 cup" → "2 1/4 cup") |
| 13 | Nutrition totals | AR per-recipe | `recipe get <url> --nutrition` | Per-serving + per-recipe |
| 14 | Unit convert | Paprika | `recipe get <url> --units metric\|us` | Cups↔ml, F↔C, lbs↔g |
| 15 | Open in browser | Paprika | `recipe open <id>` | `$BROWSER` respected |
| 16 | Import from any recipe site | recipe-scrapers (1,500+ sites) | `save <url>` works beyond AR too | Multi-source local cookbook |
| 17 | List MyRecipes saves (auth) | — (net-new, no existing tool) | `remote saves` (browser-mode) | First CLI to touch MyRecipes |
| 18 | List MyRecipes collections (auth) | — (net-new) | `remote collections` (browser-mode) | First CLI to touch MyRecipes |
| 19 | Sync MyRecipes → local (auth) | — (net-new) | `remote sync` — pulls saves into local cookbook | Bidirectional between your AR account and local FTS |
| 20 | Dinner list + grocery list | marcon29/CLI-dinner-finder (stale Ruby) | `recipe get <urls> --shopping-list` + `meal-plan shopping-list` | Actually maintained; not dinner-only |
| 21 | AR "of the day" / trending | AR homepage | `trending` — scrapes the homepage carousel | Headless or cached; machine-readable output |

## Transcendence (only possible because of the local SQLite store)

| # | Feature | Command | Why Only We Can Do This |
|---|---|---|---|
| 1 | Pantry-matched search | `cookbook match --have "chicken,rice,broccoli" --missing-max 2` | Requires local join across recipes + ingredient-normalization table; AR's site can't filter by missing-count |
| 2 | Meal plan for a week | `meal-plan set 2026-04-15 dinner <recipe-id>` + `meal-plan show --week` | Local calendar; no AR web UI for this exists |
| 3 | Shopping list from date range | `meal-plan shopping-list --from 2026-04-15 --to 2026-04-21` | Aggregates ingredients across planned meals, dedups with unit reconciliation (2 cup + 1 cup milk → 3 cup) |
| 4 | "What have I not cooked in N days" | `cookbook stale --days 60 --max-time 30m` | Requires local cook-history log; AR doesn't track this even for logged-in users |
| 5 | Nutrition totals across meal plan | `meal-plan nutrition --week` | Sum per-serving macros across planned meals × servings |
| 6 | Recipe diff between two versions | `recipe diff <id1> <id2>` | Compare two saved recipes (yours vs. AR's current) — catches silent recipe edits over time |
| 7 | Substitution suggestions | `recipe get <url> --swap "buttermilk,heavy-cream"` | Local substitution table + dependency check (will the swap break the recipe?) |
| 8 | Cost estimate | `recipe cost <id>` | Ingredient-to-price table (user-supplied or scraped per region); AR shows neither cost nor per-serving cost |
| 9 | Bulk import from browser export | `cookbook import --from-chrome-bookmarks ~/Desktop/ar-bookmarks.html` | Users can export Chrome bookmarks → CLI parses + scrapes each |
| 10 | Export to another tool | `cookbook export --format paprika\|mealie\|cooklang\|markdown` | Your recipes aren't locked in |
| 11 | "Cooking history" log | `cook log <id> --rating 4 --notes "too salty"` + `cook history` | Track what you actually cooked and how it went — no web equivalent anywhere |
| 12 | Weekly rotation avoidance | `meal-plan suggest --no-repeat-within 14d` | Suggest recipes that avoid repeats in the last N days of cook-log |

Twelve transcendence features, all unlockable with anon-path data + a local store. None require the authenticated TLS-fingerprint gauntlet.

## Auto-suggested novel features (grounded in user rituals)

- **Novel #T1 (Pantry-matched search)** — Evidence: Reddit/forum pain point ("what can I make with what's in my fridge" is the #1 recipe-app ask that no CLI addresses).
- **Novel #T3 (Shopping list from date range)** — Evidence: Paprika's killer feature; unavailable in any CLI.
- **Novel #T4 (Stale-in-cookbook query)** — Evidence: power-user cooks keep a "haven't made in a while" list manually; we just make it queryable.
- **Novel #T11 (Cook log + ratings)** — Evidence: cooks on r/cooking consistently ask for a private notes-per-recipe feature AR's saved-recipes product doesn't provide.

## Dropped from scope

- **Meal planner (remote)** — not exposed in MyRecipes web UI; mobile-only (iOS "AllRecipes Meal Planner"). Building without captured traffic would be guesswork.
- **Shopping list (remote)** — same reason.
- **Ratings/reviews write** — the authenticated write-path is TLS-fingerprint-gated; shipping an unreliable write feature is worse than not having it.
- **Follow cooks / social** — low value for CLI; drop.

## The 12 transcendence features deliver real user value. Building all 21 absorbed + 12 transcendence gives 33 commands.

Score estimate: transcendence features 1, 2, 3, 4, 11 all score 8–9/10 (specific evidence, clearly unlockable by our store, no existing tool). Others score 5–7.
