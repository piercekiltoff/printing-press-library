# Recipe Goat CLI

**Find the best version of any recipe across 37 trusted sites — then plan, shop, and cook with a local kitchen companion.**

Recipe GOAT aggregates 37 of the web's most trusted recipe sites (King Arthur, Serious Eats, Budget Bytes, Smitten Kitchen, Food52, AllRecipes, Food Network, Simply Recipes, EatingWell, BBC Good Food, Bon Appétit, Epicurious, and 25 more), ranks results by merged trust + rating + review-count signals, and builds a local SQLite cookbook that powers pantry match, cook log, meal plans, and aisle-grouped shopping lists. Unique commands like `goat` (best-version ranker), `sub` (cross-site substitution aggregation), `tonight` (decision-fatigue killer), and `cookbook match --have` (pantry match) solve problems no single recipe site can.

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/food-and-dining/recipe-goat/cmd/recipe-goat-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Authentication

USDA FoodData Central (free, 3,500 req/hr) enables nutrition backfill when a recipe site omits macros. Get a key at https://fdc.nal.usda.gov/api-key-signup and export USDA_FDC_API_KEY. All other features work without any authentication.

## Quick Start

```bash
# Verify USDA key and per-site reachability
recipe-goat-pp-cli doctor


# Rank the best version across 37 sites
recipe-goat-pp-cli goat "chicken tikka masala" --limit 5


# Save to your local cookbook
recipe-goat-pp-cli save https://www.seriouseats.com/the-best-chicken-tikka-masala-recipe


# What can I make tonight?
recipe-goat-pp-cli cookbook match --have "chicken,rice,tomato" --missing-max 2


# Out of buttermilk — what works in cakes?
recipe-goat-pp-cli sub buttermilk --context baking


# Pick dinner in 2 seconds
recipe-goat-pp-cli tonight --max-time 30m --kid-friendly

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Cross-site intelligence

- **`goat`** — Query any dish across 37 recipe sites and rank surviving Recipe-JSON-LD candidates by `0.55·rating + 0.25·log(reviews+1)/log(1000) + 0.15·site_trust + 0.05·recency`. Rating and review count come from each source page's Schema.org `aggregateRating`. `site_trust` is hand-curated (see `internal/recipes/sites.go`): editorially-curated chef/baker sites get 0.9–0.95, mass-market crowdsourced aggregators (AllRecipes, Food Network, Simply Recipes, EatingWell) get 0.70–0.75. Two trust-aware adjustments run before scoring: (1) curated sites with no Schema.org rating get an imputed 4.5/100 baseline (editorial vetting ≈ 100 implicit favorable reviews); (2) aggregator-site ratings are Bayesian-smoothed toward 4.0 with credibility C=200 — so AllRecipes "5.0 with 100 reviews" effectively becomes 4.33, while "4.7 with 5000 reviews" stays 4.67. Net effect: a niche curated recipe with no ratings can outrank a mid-tier AllRecipes result, but a heavily-reviewed AllRecipes blockbuster still wins.

  _Use this when you need the single best version of a dish — the agent gets structured results with provenance and trust signals instead of guessing from a web search._

  ```bash
  recipe-goat-pp-cli goat "chicken tikka masala" --limit 5 --json
  ```
- **`sub`** — Curated ingredient-substitution table sourced from King Arthur, Serious Eats, Budget Bytes, Minimalist Baker, and AllRecipes community reviews. Hand-curated and shipped with the binary (no live fetching at query time); ranked by source trust with ratios and context filters.

  _When a recipe needs a sub, agents can pick the best one given the cooking context (baking vs marinade) instead of suggesting the first hit on Google._

  ```bash
  recipe-goat-pp-cli sub buttermilk --context baking
  ```
- **`recipe reviews`** *(planned, work-in-progress — emits a stub today, no review aggregation yet)* — Will surface the top modifications cooks made to a recipe (e.g., "added an egg: 22 cooks; baked 5 min less: 17") once the source-review fetcher is wired. Today the command returns a clearly-labeled placeholder so agents don't depend on it.

  ```bash
  recipe-goat-pp-cli recipe reviews <id> --limit 10  # emits stub message in v1
  ```
- **`recipe get --nutrition`** — When a site omits nutrition, parse ingredients, match USDA FoodData Central IDs, compute per-serving macros locally.

  _Agents can answer 'is this recipe high-protein?' reliably even when the source doesn't publish macros._

  ```bash
  recipe-goat-pp-cli recipe get https://www.budgetbytes.com/creamy-mushroom-pasta/ --nutrition
  ```
- **`recipe get`** — Flag out-of-season ingredients inline ("⚠ asparagus is out of season in November — peak April–June") and suggest in-season swaps.

  _Agents surface cost + quality signals the user wouldn't otherwise see._

  ```bash
  recipe-goat-pp-cli recipe get <url>  # seasonal flag appears automatically
  ```

### Local state that compounds

- **`cookbook match`** — Find recipes in the local cookbook that you can make right now with listed ingredients, or with ≤N missing ingredients.

  _When the user says 'what can I make with what's in my fridge,' the agent gets ranked candidates with missing-ingredient counts instead of guessing._

  ```bash
  recipe-goat-pp-cli cookbook match --have "chicken,rice,broccoli" --missing-max 2
  ```
- **`tonight`** — Pick dinner in 2 seconds: filter cookbook by time budget, recency from cook log, and dietary/kid-friendly flags.

  _Ends the 20-minute 'what are we having' debate with three data-backed candidates._

  ```bash
  recipe-goat-pp-cli tonight --max-time 30m --no-repeat-within 7d --kid-friendly
  ```
- **`search --kid-friendly`** — Filter recipes against an editable ingredient-exclusion list (capers, anchovies, excess heat, raw fish, etc.). Personalizable per-child.

  _Parents get results actually edible by their kids, not marketing's idea of 'kid-friendly'._

  ```bash
  recipe-goat-pp-cli search "chicken dinner" --kid-friendly --limit 10
  ```
- **`meal-plan shopping-list`** — Aggregate ingredients across planned meals, reconcile units (2 cup + 1 cup milk → 3 cup), group by grocery aisle.

  _The agent hands the user a complete shopping list ready for grocery day, aisle-grouped._

  ```bash
  recipe-goat-pp-cli meal-plan shopping-list --week --export md
  ```
- **`recipe cost`** *(approximate, work-in-progress — placeholder heuristic only)* — Will eventually estimate cost per serving from Budget Bytes line-item data plus USDA retail averages. Today the command emits a clearly-labeled stub with a note that ingredient-price data integration is not yet wired.

  ```bash
  recipe-goat-pp-cli recipe cost <id>  # emits placeholder + wip note in v1
  ```

## Usage

Run `recipe-goat-pp-cli --help` for the full command reference and flag list.

## Commands

### foods

USDA FoodData Central — ingredient nutrition lookups

- **`recipe-goat-pp-cli foods get`** - Get a specific food by FDC ID
- **`recipe-goat-pp-cli foods list`** - List foods paginated
- **`recipe-goat-pp-cli foods search`** - Search USDA FoodData Central for foods matching a query


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
recipe-goat-pp-cli foods list

# JSON for scripting and agents
recipe-goat-pp-cli foods list --json

# Filter to specific fields
recipe-goat-pp-cli foods list --json --select id,name,status

# Dry run — show the request without sending
recipe-goat-pp-cli foods list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
recipe-goat-pp-cli foods list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use as MCP Server

This CLI ships a companion MCP server for use with Claude Desktop, Cursor, and other MCP-compatible tools.

### Claude Code

```bash
claude mcp add recipe-goat recipe-goat-pp-mcp -e USDA_FDC_API_KEY=<your-key>
```

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "recipe-goat": {
      "command": "recipe-goat-pp-mcp",
      "env": {
        "USDA_FDC_API_KEY": "<your-key>"
      }
    }
  }
}
```

## Health Check

```bash
recipe-goat-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/recipe-goat-pp-cli/config.toml`

Environment variables:
- `USDA_FDC_API_KEY`

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `recipe-goat-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $USDA_FDC_API_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **HTTP 402 or 403 on AllRecipes / Simply Recipes / EatingWell / Serious Eats fetches** — These are Dotdash Meredith sites with TLS-fingerprint bot detection. Run `recipe-goat-pp-cli doctor` to see per-site reachability. The CLI falls back to other sources automatically; your `goat` ranking will show results from reachable sites.
- **Nutrition missing or marked `[source: site]` with obviously wrong numbers** — Pass `--nutrition` to force USDA backfill. Requires USDA_FDC_API_KEY. Mark suspect recipes with `cookbook tag <id> nutrition-suspect`.
- **`goat` query returns nothing** — Check `--site` filter if set. Try broader terms. Run `doctor` to see per-site reachability; sites with `WARN` are the likely zero-result culprits. Note: Food52's search HTML is JS-rendered — `goat`/`search` returns 0 from Food52, but `recipe get <food52-url>` and `save <food52-url>` work normally.
- **Shopping list has duplicate entries with different units** — Unit reconciliation is wip; the v1 aggregator counts ingredient lines verbatim. Until that lands, edit the export manually or pass `--csv` and reconcile in a spreadsheet.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**Mealie**](https://github.com/mealie-recipes/mealie) — Python (11500 stars)
- [**Tandoor Recipes**](https://github.com/TandoorRecipes/recipes) — Python (7200 stars)
- [**hhursev/recipe-scrapers**](https://github.com/hhursev/recipe-scrapers) — Python (2100 stars)
- [**KitchenOwl**](https://github.com/TomBursch/kitchenowl) — Dart (1700 stars)
- [**Cooklang**](https://github.com/cooklang/cookcli) — Rust (580 stars)
- [**python-allrecipes**](https://github.com/remaudcorentin-dev/python-allrecipes) — Python (20 stars)
- [**marcon29/CLI-dinner-finder-grocery-list**](https://github.com/marcon29/CLI-dinner-finder-grocery-list) — Ruby (5 stars)

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
