# Recipe Goat CLI

**Find the best version of any recipe across curated cuisine-authority sites — then plan, shop, and cook with a local kitchen companion.**

Recipe GOAT aggregates a curated set of independent, cuisine-authoritative recipe sites — Nagi (RecipeTin Eats), Swasthi (Indian Healthy Recipes), Elaine (China Sichuan Food), The Woks of Life, Just One Cookbook, Sally's Baking Addiction, King Arthur Baking, Budget Bytes, BBC Food, and more — ranks results by merged trust + rating + review-count signals, and builds a local SQLite cookbook that powers pantry match, cook log, meal plans, and aisle-grouped shopping lists. When users paste URLs from bot-detection-gated sites (allrecipes, food52, etc.), archive.org's Wayback Machine is used to recover the content. Unique commands like `goat` (best-version ranker), `sub` (cross-site substitution aggregation), `tonight` (decision-fatigue killer), and `cookbook match --have` (pantry match) solve problems no single recipe site can.

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/food-and-dining/recipe-goat-pp-cli/cmd/recipe-goat-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Optional: USDA API Key (nutrition backfill)

**All core commands work without any setup.** The API key below is only needed to unlock one feature: nutrition backfill when a recipe site omits macros.

### What the key unlocks

`recipe get <url> --nutrition` computes per-serving calories/protein/carbs/fat from USDA FoodData Central when the source recipe doesn't publish them. Useful because roughly 30% of recipes across the supported sites omit nutrition. Without the key you'll see `[nutrition source: site]` when it's published, or `[nutrition source: unavailable]` when it's not.

### Get a key (free, 1 minute)

1. Sign up at https://fdc.nal.usda.gov/api-key-signup — no payment required, 3,500 requests/hour quota.
2. Copy the key from the confirmation email.
3. Export it:
   ```bash
   export USDA_FDC_API_KEY=<your-key>
   ```
   Or persist it with `recipe-goat-pp-cli auth set-token <your-key>`.

Verify with `recipe-goat-pp-cli doctor` — Auth should show `INFO Auth: optional — not configured` before you set the key, and `OK Auth: configured` after.

## Quick Start

```bash
# Verify USDA key and per-site reachability
recipe-goat-pp-cli doctor


# Rank the best version across the curated corpus
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

- **`goat`** — Query any dish across curated recipe sites and rank results by normalized rating × review count × site trust × recency.

  _Use this when you need the single best version of a dish — the agent gets structured results with provenance and trust signals instead of guessing from a web search._

  ```bash
  recipe-goat-pp-cli goat "chicken tikka masala" --limit 5 --json
  ```
- **`sub`** — Aggregate ingredient substitutions from King Arthur Baking and other trusted baking-science sources. Ranked by source trust with ratios and context.

  _When a recipe needs a sub, agents can pick the best one given the cooking context (baking vs marinade) instead of suggesting the first hit on Google._

  ```bash
  recipe-goat-pp-cli sub buttermilk --context baking
  ```
- **`recipe reviews`** — Surface the top modifications cooks actually made to a recipe ("added an egg: 22 cooks; baked 5 min less: 17; honey instead of sugar: 14").

  _Agents give the user the collective wisdom of reviewers instead of just star ratings._

  ```bash
  recipe-goat-pp-cli recipe reviews <id> --limit 10
  ```
- **`recipe get --nutrition`** — When a site omits nutrition, parse ingredients, match USDA FoodData Central IDs, compute per-serving macros locally.

  _Agents can answer 'is this recipe high-protein?' reliably even when the source doesn't publish macros._

  ```bash
  recipe-goat-pp-cli recipe get https://www.budgetbytes.com/creamy-mushroom-pasta/ --nutrition
  ```
- **`recipe get (auto)`** — Flag out-of-season ingredients inline ("⚠ asparagus is out of season in November — peak April–June") and suggest in-season swaps.

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
- **`recipe cost`** — Rough cost per serving using Budget Bytes line-item data plus USDA retail averages as fallback. Always shows an honesty band (±30%).

  _Agents can triage recipes by rough cost without pretending to precision grocery data doesn't provide._

  ```bash
  recipe-goat-pp-cli recipe cost <id>  # output: '$6–$9 for 4 servings (±30%)'
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
- **Retryable** - creates return "already exists" on retry, deletes return "already deleted"
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - `echo '{"key":"value"}' | recipe-goat-pp-cli <resource> create --stdin`
- **Cacheable** - GET responses cached for 5 minutes, bypass with `--no-cache`
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set
- **Progress events** - paginated commands emit NDJSON events to stderr in default mode

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

**Rate limit errors (exit code 7)**
- The CLI auto-retries with exponential backoff
- If persistent, wait a few minutes and try again

### API-specific

- **HTTP 402 or 403 on AllRecipes / Simply Recipes / EatingWell / Food52 / FoodNetwork fetches** — These sites serve Cloudflare/Akamai bot-detection challenges to non-browser clients and are not in the default `goat` fan-out. When you paste a URL from one of them into `recipe get`, the CLI automatically falls back to archive.org's Wayback Machine and prints `warn: archive fallback: <url>` on stderr so you know the content came from an archived snapshot rather than live.
- **HTTP 403 on Serious Eats fetches** — Still in the default corpus but intermittently Akamai-gated. Run `recipe-goat-pp-cli doctor` to see current reachability; your `goat` ranking still ranks across reachable sites.
- **Nutrition missing or marked `[source: site]` with obviously wrong numbers** — Pass `--nutrition` to force USDA backfill. Requires USDA_FDC_API_KEY. Mark suspect recipes with `cookbook tag <id> nutrition-suspect`.
- **`goat` query returns nothing** — Check `--site` filter if set. Try broader terms. Use `search --debug` to see per-site fetch attempts and which sites fell back to cache or failed.
- **Shopping list has duplicate entries with different units** — Run `cookbook ingredients canonicalize` to re-run the parser. If an ingredient consistently fails, add to `~/.config/recipe-goat-pp-cli/ingredient-aliases.toml`.

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
