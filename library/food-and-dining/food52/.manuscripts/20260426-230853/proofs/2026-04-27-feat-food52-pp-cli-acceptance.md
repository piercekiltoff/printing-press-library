# Food52 Acceptance Report

**Level:** Full Dogfood
**Run:** 20260426-230853
**Tests:** 44/44 PASS

## Test Matrix

The full mechanical test matrix exercised every Food52-specific leaf subcommand against the live food52.com SSR + Typesense backend, plus operational commands (doctor, version, help). For each leaf:

- **Help check** — `--help` exits 0 and prints an Examples section
- **Happy path** — one realistic invocation with `--json`, exits 0
- **JSON fidelity** — output parses as valid JSON
- **Error path** — invocation with missing/invalid arg exits non-zero

See [`dogfood-results.md`](dogfood-results.md) for the per-test pass/fail rows.

### Coverage

| Surface | Commands tested | Pass |
|---------|-----------------|------|
| Operational | doctor, version, help | 3/3 |
| Recipes (absorbed) | browse, get, search, top | 12/12 |
| Articles (absorbed) | browse, browse-sub, get, for-recipe | 8/8 |
| Tags (absorbed) | list, list --kind | 3/3 |
| Open (UX) | help, dry | 2/2 |
| Pantry (transcendence) | add, list, match, remove | 7/7 |
| Sync (transcendence) | recipes | 2/2 |
| Search (transcendence) | search, search --type | 2/2 |
| Scale (transcendence) | scale, scale missing-servings | 3/3 |
| Print (transcendence) | print, print on real recipe | 2/2 |

### Behavioral correctness samples

- **`recipes search brownies`** returns 175 results from Typesense; top hit is "Lunch Lady Brownies" (semantically correct).
- **`recipes get sarah-fennel-s-best-lunch-lady-brownie-recipe`** returns `Lunch Lady Brownies` with 9 ingredients (cleaned, no "undefined" tokens).
- **`recipes top chicken --tk-only`** returns only Test-Kitchen-approved chicken recipes (testKitchenApproved=true), correctly using Food52's editorial signal.
- **`recipes browse chicken --limit 3`** returns 3 chicken recipes from `__NEXT_DATA__.props.pageProps.recipesByTag.results`.
- **`articles browse food`** returns blog posts from `pageProps.blogPosts.results`.
- **`articles browse-sub food baking`** returns baking-vertical articles.
- **`articles get best-mothers-day-gift-ideas`** returns the article with full body text (Sanity portable text flattened).
- **`pantry match`** scores synced recipes by ingredient overlap with the pantry, ranks by coverage, lists matched + missing ingredients.
- **`scale mom-s-japanese-curry-chicken-with-radish-and-cauliflower --servings 8`** parses `recipeYield: "4-5"`, scales every ingredient line by factor 2.0.
- **`print sarah-fennel-s-best-lunch-lady-brownie-recipe`** renders title + numbered ingredients + numbered steps with no nav, ads, or images.
- **`tags list --kind cuisine`** returns 11 cuisine tags.
- **`sync recipes vegetarian --limit 5`** pulled 5 summaries + 5 details with concurrency 2.
- **`articles for-recipe <slug>`** returns synced articles whose relatedReading mentions the slug (correctly empty for an article-less store).

## Failures

None. 44/44 tests passed.

## Fixes applied during dogfood

Two real bugs discovered and fixed in-session (not deferred):

1. **Examples in --help were not detected by dogfood's parser** — `strings.TrimSpace` on the multi-line Example string stripped the leading 2-space indent, causing dogfood's `extractExamplesSection` to break at the first unindented line. Fixed by replacing `strings.TrimSpace(...)` with `strings.Trim(..., "\n")` in 17 source files. Result: example coverage went from 0/10 to 10/10.

2. **Ingredient strings rendered "4 undefined tablespoons"** — Food52's pre-rendered JSON-LD `recipeIngredient` strings sometimes carry the literal " undefined " token where the source CMS field was unset. Fixed by adding `cleanIngredientStrings` post-processor in `internal/food52/recipe.go` that strips standalone "undefined" tokens before returning ingredients. Verified clean output on a real recipe (Salt and Pepper Ribs and Wings).

3. **SKILL.md had two `articles get` examples without positional args** — `verify-skill` flagged them. Fixed by adding the canonical slug `best-mothers-day-gift-ideas` to both examples. `verify-skill` now exits 0 with no findings.

## Printing Press improvements (for retro)

These are systemic gaps that surfaced during this run — candidates for upstream generator improvements rather than printed-CLI fixes:

1. **Generator's `extractHTMLResponse` only supports `mode: page` and `mode: links`** — neither handles Next.js `__NEXT_DATA__` extraction, which is the dominant pattern for SSR-React sites (Food52, Producthunt, many others). Adding a `mode: next-data` with a `json_path` would let the generator emit the right scaffolding instead of forcing every Next.js CLI to hand-replace the 4 generated handlers.

2. **`traffic-analysis.json` schema rejects string-shaped evidence in reachability** — the generator code accepts `browser_http` as a reachability mode but the schema enum omits it; the schema also requires evidence to be EvidenceRef objects (entry_index + status fields) which only make sense when the analysis came from `printing-press browser-sniff` over a HAR. Hand-authored discovery reports with prose evidence get rejected. Either widen the schema or document the canonical evidence shape.

3. **The generic `extractHTMLResponse` helpers (printOutput, filterFields, html_extract, etc.) leave 30 dead helper functions when the printed CLI replaces all four generated handlers** — these inflate the dead-code dimension of the scorecard (0/5). Either prune unused helpers post-replacement or template them behind a build tag so they're only emitted when at least one command uses the page/links HTML extractor.

4. **`Example: strings.TrimSpace(...)` is the obvious pattern but produces a non-indented first line that breaks dogfood's example detection** — generator templates should emit `Example: strings.Trim(..., "\n")` (preserves leading 2-space indent). I had to write a sed pass over 17 files to fix this. The bug is silent until dogfood catches it.

## Gate

**PASS.** All ship-threshold conditions met:

- `verify` PASS (83% pass rate, 0 critical failures)
- `dogfood` WARN-only (30 dead helpers, see retro #3 above; examples + novel-features pass)
- `workflow-verify` workflow-pass (no manifest, auto-pass)
- `verify-skill` exits 0
- `scorecard` 79/100 Grade B
- Live dogfood 44/44

No known functional bugs in shipping-scope features.
