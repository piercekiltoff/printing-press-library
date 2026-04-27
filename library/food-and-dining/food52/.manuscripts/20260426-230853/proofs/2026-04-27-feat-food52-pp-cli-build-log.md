# Food52 Build Log

## Generated (Phase 2)
- `printing-press generate --spec research/food52-spec.yaml --output working/food52-pp-cli --spec-source sniffed --force --lenient --validate`
- All 7 quality gates passed: go mod tidy, go vet, go build, binary build, --help, version, doctor.
- Generator emitted: 4 SSR resources (recipes browse/get, articles browse/get) plus client.go (Surf+Chrome impersonation), config.go, doctor.go, store.go (recipes + articles tables, FTS), MCP server, agent-context, auth, profile, deliver, export, import, feedback, which, workflow scaffolding.
- Generated traffic-analysis.json validation rejected my hand-authored evidence shape, so I generated without `--traffic-analysis` and let the spec's explicit `http_transport: browser-chrome` drive Surf transport selection.

## Hand-built (Phase 3)

### internal/food52/ (new package, P0)
- `nextdata.go` — fetch HTML, regex out `<script id="__NEXT_DATA__">`, parse pageProps, extract Next.js buildId, detect Vercel challenge HTML.
- `recipe.go` — typed RecipeSummary + Recipe; `ExtractRecipesByTag(html)`, `ExtractRecipe(html, url)`. Merges SSR `pageProps.recipe` with Schema.org Recipe JSON-LD overlay so callers get whichever source is richer per field. Includes `cleanIngredientStrings` to strip the literal " undefined " tokens Food52 sometimes emits when its CMS qty/unit fields are unset.
- `article.go` — typed ArticleSummary + Article; `ExtractArticlesByVertical(html)`, `ExtractArticle(html, url)`. Walks Sanity portable text into flat strings.
- `discovery.go` — runtime discovery of buildId + Typesense host + search-only key. Fetches `/` to find the active `_app-<hash>.js`, regexes the typesense literal, caches at `$XDG_CACHE_HOME/food52-pp-cli/discovery.json` with a 6-hour TTL. The Typesense key value is never persisted to source, the spec, or any committed artifact — discovered fresh from the public bundle on each TTL expiry.
- `typesense.go` — direct Typesense client using the printed CLI's existing `*http.Client` (Surf+Chrome). Returns RecipeSummary records projected from the search hit shape. Handles 401/403 by surfacing `ErrTypesenseAuth` so callers can `InvalidateDiscovery()` and retry once.
- `tags.go` — curated 40+ tag enum (slug → display name → kind). Scoped by Kind: meal, course, ingredient, cuisine, lifestyle, preparation, convenience.
- `util.go` — typed-field accessors, Sanity portable-text flattener, Sanity image-ref → Sanity CDN URL converter, `cleanIngredientStrings`.
- 4 `_test.go` files: parsers tested against 4 captured HTML fixtures (chicken category, recipe detail, article landing, story detail). Cover happy-path extraction + LooksLikeChallenge + tag-kind filtering + Typesense regex extraction.

### internal/cli/ (replacements + new commands, P1 + P2)

Replaced (the generated handlers used the generic html_extract page mode which returns nav metadata; my replacements parse __NEXT_DATA__ for real recipe/article data):
- `recipes_browse.go` — calls `food52.ExtractRecipesByTag`, supports `--limit`, `--data-source local` queries the recipes table by tag column.
- `recipes_get.go` — calls `food52.ExtractRecipe`, supports both bare slug and full Food52 URL, `--data-source local` queries by slug.
- `articles_browse.go` — calls `food52.ExtractArticlesByVertical`.
- `articles_get.go` — calls `food52.ExtractArticle`.
- `food52_helpers.go` — shared helpers (fetchHTML, slug parsers, canonical URL builders, emit helpers).
- `store_helpers.go` — typed recipe/article upsert + pantry-table migration.

Added (P1 absorbed):
- `recipes_search.go` — Typesense search via `food52.SearchRecipes`. `--tag`, `--page`, `--limit`, `--sort`. Auto-rediscovers Typesense key on auth failure.
- `recipes_top.go` — wide Typesense pull then filter for `testKitchenApproved` + `--min-rating`. `--no-tk` opt-out.
- `articles_browse_sub.go` — `articles browse-sub <vertical> <sub>` for the deeper Food52 article taxonomy (food/baking, life/travel, etc.).
- `tags.go` — `tags list [--kind ...]` emits the curated enum.
- `open.go` — shell-out to xdg-open / open / start with slug-or-URL detection.

Added (P2 transcendence):
- `pantry.go` — `pantry add/list/remove/match` with on-demand `pantry` table migration. Match scores recipes by substring overlap, ranks by coverage.
- `sync_recipes.go` — fetches tag pages, then per-recipe details with bounded concurrency (default 4, max 16). `--summary-only` flag for fast-path that skips per-recipe fetch.
- `sync_articles.go` — same shape for articles.
- `search.go` — local FTS-equivalent (LIKE over title/description/body via json_extract). Cross-corpus by default, `--type recipe|article` to constrain.
- `scale.go` — fetch recipe, parse `recipeYield`, scale ingredient quantities. Supports mixed numbers (1 1/2), simple fractions (1/2), decimals (1.5).
- `print.go` — clean cooking-mode view (no nav, no images, no ads).
- `articles_for_recipe.go` — reverse-index synced articles by their `relatedReading` field.

Edits to generated code:
- `recipes.go` — added 2 subcommands (search, top).
- `articles.go` — added 2 subcommands (browse-sub, for-recipe).
- `root.go` — added 6 top-level commands (tags, open, pantry, search, scale, print).
- `sync.go` — added 2 subcommands (recipes, articles).
- `SKILL.md` — fixed 2 examples that called `articles get` without a positional arg.

## What was intentionally deferred
- Hotline (community Q&A) — `/hotline*` SSR returns only siteSettings; no scrapable Q&A surface. Confirmed dead during browser-sniff; documented in discovery report.
- Shop / Storefront commerce — would require Shopify Storefront GraphQL token discovery; out of scope for v0.1 unauthenticated CLI.
- Workflow-verify manifest — none authored; workflow-verify auto-skipped to PASS.

## Generator limitations encountered
- Generated `extractHTMLResponse` only supports `mode: page` and `mode: links` for HTML extraction. Food52's data lives in `__NEXT_DATA__` which neither mode handles. **Workaround:** kept the spec's `response_format: html` declaration but rewrote the 4 affected command bodies to call my `food52.ExtractRecipesByTag` / `food52.ExtractRecipe` etc. directly instead of `extractHTMLResponse`. Could be a useful future generator mode (`mode: next-data` with a `json_path`).
- Generated traffic-analysis.json schema strictly types `evidence` as `EvidenceRef` objects, while my discovery report had string evidence. Worked around by skipping `--traffic-analysis` and relying on the spec's explicit `http_transport: browser-chrome`. Could be a generator improvement to allow string-shaped evidence.
- The 30 dead helpers from `helpers.go` are real dead code now that my replacements bypass them. Generator could prune or template the helpers behind a `// pp:html-page-handler` build tag so they aren't emitted when no command uses `mode: page`.

## Skipped complex body fields
None — Food52's surface is GET-only.
