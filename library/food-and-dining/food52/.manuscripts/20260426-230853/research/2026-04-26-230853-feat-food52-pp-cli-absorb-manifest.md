# Food52 Absorb Manifest

The Food52 ecosystem is unusually quiet: one near-dead Ruby CLI, one generic recipe-scraper Python lib that lists Food52 as one of 200+ supported sites, no MCP server, no Claude plugin or skill. Easy to absorb everything that exists, transcend with offline FTS + local pantry + agent-native plumbing.

## Source Tools

| # | Tool | Type | Status | Notes |
|---|------|------|--------|-------|
| 1 | [imRohan/food52-cli](https://github.com/imRohan/food52-cli) | Ruby gem CLI | Broken (2018-era HTML selectors; plain HTTP gets Vercel-challenged) | Interactive `tty-prompt` only, no `--json`, no flags |
| 2 | [hhursev/recipe-scrapers](https://github.com/hhursev/recipe-scrapers) | Python library | Active | Stub `food52.py` inheriting Schema.org JSON-LD extraction; single-recipe only |
| 3 | Various npm/PyPI recipe scrapers | Generic libs | Various | Mention food52 as one of many supported recipe sites; no Food52-specific features |
| 4 | Food52 itself (no public API) | n/a | n/a | All "competition" is in the website's own search/browse UI |

No MCP server, no Claude Code plugin, no Claude skill exists for Food52. There is also no published GitHub Action or n8n integration.

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 1 | Search recipes by keyword | imRohan/food52-cli `Keywords` mode | `recipes search <query> [--limit N]` via Typesense direct | `--json`, `--select`, ranks by Food52's own popularity/rating, sub-second instead of 2KB HTML scrape, `--tag` filter |
| 2 | Search recipes by ingredient | imRohan/food52-cli `Ingredients` mode (multi-select) | `recipes search <query> --tag <ingredient>` (Typesense `filter_by`) | `--limit`, `--page`, structured output |
| 3 | Browse recipes by cuisine | imRohan/food52-cli `Cuisine` mode | `recipes browse <tag>` against `/recipes/<tag>` SSR | Tag is general (cuisines, ingredients, meal types are all tags), `--json`, paginated |
| 4 | Browse recipes by meal type | imRohan/food52-cli `Meal` mode (Breakfast/Brunch/...) | Same `recipes browse <tag>` (breakfast, dinner, etc. are tags) | Single command for all axes, agent-friendly |
| 5 | Show single recipe with ingredients + steps | imRohan/food52-cli + recipe-scrapers | `recipes get <slug-or-url>` via `__NEXT_DATA__.recipe` + Schema.org JSON-LD | `--json`, `--md`, `--print`, full ratings + kitchen notes + author + tags + product references — fields the existing tools throw away |
| 6 | List recipe tags | imRohan/food52-cli `ingredients` and `cuisines` lists | `tags list [--kind ingredient\|cuisine\|meal\|preparation\|...]` from a curated enum sourced from the homepage navigation | Discoverability via `--json`, no broken `/recipes/ingredient/all` scrape |
| 7 | Recipe data extraction (single URL) | hhursev/recipe-scrapers food52.py (JSON-LD) | `recipes get` does the same via JSON-LD as a fallback when `__NEXT_DATA__` is missing | Bundled into a CLI that also does search, browse, sync, FTS — not just one URL at a time |
| 8 | Article browsing | None (no existing tool covers Food52 articles) | `articles browse <vertical> [<sub>]` and `articles get <slug>` via SSR `pageProps.blogPosts` / `blogPost` | New surface no competing tool exposes |

Every absorbed feature ships with `--json`, `--limit` where applicable, typed exit codes, and works against a local SQLite cache after `sync`.

## Transcendence (only possible with our approach)

| # | Feature | Command | Why Only We Can Do This | Score |
|---|---------|---------|------------------------|-------|
| 1 | Offline pantry → recipe matcher | `pantry add <ingredient>...`, `pantry list`, `pantry remove <ingredient>`, `pantry match [--min-coverage 0.7]` | Requires a local pantry inventory + a synced index of recipe ingredients. Food52's site only lets you search by single ingredient; we score *every* synced recipe by what fraction of its ingredients you already have. | 8/10 |
| 2 | Local FTS5 search across cookbook | `search <query> [--type recipe\|article]` | Requires synced recipes + articles in a local store. Searches title + description + ingredients + body text in one call, works on a plane, regex/SQL composable. | 7/10 |
| 3 | Sync slice of catalog | `sync recipes <tag>...`, `sync articles <vertical> [<sub>]` | Foundation for transcendence. Pulls a tag page or vertical landing into the local store, FTS-indexed. Non-mutating, polite by default. | 7/10 |
| 4 | Test-Kitchen-only browse | `recipes top <tag> [--min-rating N] [--limit N]` | Uses Food52's `testKitchenApproved` + `averageRating` editorial signals — the existing tools throw these fields away. The Food52 site itself doesn't expose a "Test Kitchen Approved + 4-star+" filtered listing. | 7/10 |
| 5 | Recipe scaling via JSON-LD | `scale <slug-or-url> --servings N` | Parses `recipeYield` from the JSON-LD on the recipe page, applies fractional scaling to every `recipeIngredient` line. Food52's site has no scaler; agent-friendly arithmetic over the structured ingredients list. | 6/10 |
| 6 | Cooking-mode print view | `print <slug-or-url>` | Renders ingredients + numbered steps to a clean fixed-width view (no nav, no images, no ads, no comments). Designed for tearing off and sticking on the fridge or piping to `lp`. The site's "Print Recipe" button still loads ad-laden chrome. | 6/10 |
| 7 | Article ↔ recipe cross-reference | `articles for-recipe <slug>` | Reverse-indexes synced articles by their `relatedReading` field to find the article(s) that reference a given recipe. Food52 recipes link out, but never *back* to the articles that mention them. | 6/10 |
| 8 | Open in browser | `open <slug-or-url>` | Trivial UX feature, but the existing CLI is interactive-only and offers no path back to the live page. | 4/10 — still ship it, it's one shell-out. |

Cut from this manifest at user request after the gate readout: `recipes recent` (synced-history novelty, scoped out for v0.1) and the broader `tags list` + `verticals list` taxonomy command (the absorbed simpler `tags list` from row 6 above remains).

Total: **8 absorbed features + 8 transcendence features = 16 commands** (excluding `doctor`, `version`, `--help`, `auth` which the generator emits standardly). One transcendence row (`open`) is intentionally low-score but easy enough to ship.

## Stubs

None. Every feature in the manifest is buildable with the unauthenticated surface discovered in browser-sniff. No paid API gates, no headless Chrome required at runtime (Surf with Chrome impersonation handles Vercel mitigation), no auth wall.

## Excluded (out of scope per user constraint)

These would require sign-in or a different runtime shape and were intentionally dropped:

- Account, profile, saved recipes, personal collections — sign-in required
- Comments and ratings (write side) — sign-in required
- Shop / Storefront commerce — would require Shopify Storefront GraphQL token discovery; deferred to a future v0.2
- Hotline community Q&A — no public surface; the page renders only `siteSettings`
- Recipe contests and contest submissions — sign-in required
