# Food52 CLI

**Search, browse, and read Food52 from your terminal — with offline FTS, pantry matching, recipe scaling, and the editorial signals other tools throw away.**

Every recipe and article on Food52, queryable without a browser. Ships with `pantry match` (find recipes from what you already have), `search` (offline FTS over your synced cookbook), `recipes top` (Test-Kitchen approved + rating-floored), and `scale` (resize ingredient lists via JSON-LD). The only existing Food52 CLI is a 2018-era Ruby HTML scraper that no longer runs against today's Vercel-protected site; this is a clean rebuild on Surf with Chrome TLS impersonation.

Learn more at [Food52](https://food52.com).

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/cmd/food52-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Authentication

**Nothing to set up.** No Food52 sign-in, no API key, no env var.

Two pieces of plumbing make that possible:

1. **Vercel bot mitigation** is bypassed at the transport layer. Food52 sits
   behind Vercel's TLS-fingerprint challenge — `curl` and stock Go
   `net/http` get the "Just a moment…" interstitial. The printed CLI uses
   [Surf](https://github.com/imroc/req) with Chrome impersonation so every
   request looks like a real Chrome handshake. No clearance cookie, no
   resident browser, no JS execution.
2. **Typesense search-only key** is auto-discovered. Food52's recipe search
   is powered by a public Typesense cluster. The CLI fetches the
   `/_app.js` bundle on first use, parses out the host + search-only key,
   and caches them locally. The bundle hash rotates on every Food52 deploy
   — `food52-pp-cli doctor` re-runs the discovery if a cached key starts
   returning auth errors. You never see this happen.

If `doctor` reports `unreachable` or `Vercel Security Checkpoint`, the
transport is the problem — see [Troubleshooting](#troubleshooting).

## Quick Start

```bash
# Live Typesense search; sub-second results.
food52-pp-cli recipes search "brownies" --limit 5 --json


# Full structured recipe (ingredients, steps, ratings, kitchen notes) for one recipe.
food52-pp-cli recipes get sarah-fennel-s-best-lunch-lady-brownie-recipe --json


# Seed the local store with two tags worth of recipes — required for offline search and pantry match.
food52-pp-cli sync recipes chicken vegetarian --limit 100


# Tell the CLI what's in your kitchen.
food52-pp-cli pantry add chicken garlic lemon thyme


# Find synced recipes you can mostly make right now.
food52-pp-cli pantry match --min-coverage 0.6 --json


# Clean cooking-friendly view, ready to pipe to lp or paste.
food52-pp-cli print sarah-fennel-s-best-lunch-lady-brownie-recipe

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local state that compounds

- **`pantry match`** — Find Food52 recipes whose ingredients overlap your local pantry, ranked by coverage.

  _Reach for this when the user asks 'what can I make with what I have' rather than searching for one dish at a time._

  ```bash
  food52-pp-cli pantry match --min-coverage 0.7 --json
  ```
- **`search`** — Full-text search across every recipe and article you have synced, with type filtering.

  _Use this for lookups that can't justify a Typesense round trip or when offline._

  ```bash
  food52-pp-cli search "miso" --type recipe --json
  ```
- **`sync recipes`** — Pull recipes for one or more tags into the local FTS-indexed store.

  _Run before pantry match or offline search to seed the local cookbook._

  ```bash
  food52-pp-cli sync recipes chicken vegetarian --limit 100
  ```
- **`articles for-recipe`** — Find synced articles that mention a given recipe in their relatedReading.

  _Use when the user wants the editorial context behind a recipe they've found._

  ```bash
  food52-pp-cli articles for-recipe sarah-fennel-s-best-lunch-lady-brownie-recipe
  ```

### Editorial signals others ignore

- **`recipes top`** — Show only Food52 Test-Kitchen-approved recipes for a tag, with a rating floor.

  _Pick this over a broad search when the user wants 'a recipe Food52's editors signed off on,' not just any community recipe._

  ```bash
  food52-pp-cli recipes top chicken --min-rating 4 --limit 5 --json
  ```

### Recipe transforms

- **`scale`** — Scale a recipe's ingredients to a different number of servings using its Schema.org recipeYield.

  _Use when the user is cooking for a different headcount than the recipe's default yield._

  ```bash
  food52-pp-cli scale mom-s-japanese-curry-chicken-with-radish-and-cauliflower --servings 8 --json
  ```
- **`print`** — Render a recipe as ingredients + numbered steps with no nav, no images, no ads, no comments — ready to pipe to lp or paste into notes.

  _Use when the user wants to actually cook from the recipe rather than browse it._

  ```bash
  food52-pp-cli print sarah-fennel-s-best-lunch-lady-brownie-recipe
  ```

## Usage

Run `food52-pp-cli --help` for the full command reference and flag list.

## Commands

### Recipes

- **`food52-pp-cli recipes browse <tag>`** — Browse recipes filtered by tag (chicken, breakfast, vegetarian, dessert, …)
- **`food52-pp-cli recipes get <slug-or-url>`** — Full structured recipe (ingredients, steps, ratings, kitchen notes)
- **`food52-pp-cli recipes search <query>`** — Live Typesense search across the whole site
- **`food52-pp-cli recipes top <tag>`** — Test-Kitchen-approved + rating-floored browse
- **`food52-pp-cli scale <slug-or-url> --servings N`** — Rewrite a recipe's quantities to a different yield
- **`food52-pp-cli print <slug-or-url>`** — Clean ingredients + numbered steps, ready to pipe to `lp`

### Articles

- **`food52-pp-cli articles browse <vertical>`** — Latest articles in `food` or `life`
- **`food52-pp-cli articles browse-sub <vertical> <subvertical>`** — Drill into a subvertical (e.g. `food baking`)
- **`food52-pp-cli articles get <slug-or-url>`** — Full article body + author + dek
- **`food52-pp-cli articles for-recipe <slug-or-url>`** — Reverse index: which synced articles mention this recipe?

### Local store and pantry

- **`food52-pp-cli sync recipes <tag> [<tag>...]`** — Pull recipes for one or more tags into the local FTS-indexed store
- **`food52-pp-cli sync articles <vertical> [<subvertical>]`** — Pull article listings into the local store
- **`food52-pp-cli search <query>`** — Offline FTS5 search across synced recipes and articles
- **`food52-pp-cli pantry add <ingredient> [<ingredient>...]`** — Add ingredients to the local pantry
- **`food52-pp-cli pantry list`** — Show the pantry
- **`food52-pp-cli pantry remove <ingredient>`** — Remove an ingredient
- **`food52-pp-cli pantry match`** — Find synced recipes that match (or mostly match) the pantry

### Discovery and utilities

- **`food52-pp-cli tags list`** — Discover recipe tags from the homepage navigation
- **`food52-pp-cli open <slug-or-url>`** — Resolve a slug to its canonical URL (prints by default; pair with `--launch` to actually open in the browser)
- **`food52-pp-cli which "<capability>"`** — Find the right command from a natural-language description
- **`food52-pp-cli doctor`** — Verify transport, auth, and cache health
- **`food52-pp-cli agent-context`** — Emit a structured JSON description of this CLI for agents
- **`food52-pp-cli profile {save,use,list,show,delete}`** — Reusable flag presets
- **`food52-pp-cli feedback "..."`** — Record a one-line note about the CLI (local by default)
- **`food52-pp-cli {export,import}`** — Move synced data in/out as JSONL


## Output Formats

```bash
# Human-readable text (default in terminal, JSON when piped)
food52-pp-cli articles get best-mothers-day-gift-ideas

# JSON for scripting and agents
food52-pp-cli articles get best-mothers-day-gift-ideas --json

# Project just the fields you need (dotted paths descend; arrays traverse element-wise)
food52-pp-cli recipes search chicken --json --select 'hits.title,hits.slug,hits.average_rating'

# Dry run — show the URL the CLI would hit, without making the request
food52-pp-cli recipes get sarah-fennel-s-best-lunch-lady-brownie-recipe --dry-run

# Agent mode — turns on --json, --compact, --no-input, --no-color, --yes in one flag
food52-pp-cli articles get best-mothers-day-gift-ideas --agent
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

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Use as MCP Server

This CLI ships a companion MCP server for use with Claude Desktop, Cursor, and other MCP-compatible tools.

### Claude Code

```bash
claude mcp add food52 food52-pp-mcp
```

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "food52": {
      "command": "food52-pp-mcp"
    }
  }
}
```

## Health Check

```bash
$ food52-pp-cli doctor
  OK Config: ok
  OK Auth: not required
  OK API: reachable
  config_path: ~/.config/food52-pp-cli/config.toml
  base_url: https://food52.com
  version: 1.0.0
  INFO Cache: unknown
    db_path: ~/.local/share/food52-pp-cli/data.db
    schema_version: 1
    db_bytes: 122880
    stale_after: 6h0m0s
    hint: sync_state is empty; run 'food52-pp-cli sync' to hydrate.
```

Verifies the Surf-Chrome transport is clearing the Vercel challenge, that
the Typesense discovery key is reachable, and reports cache status. Run
`food52-pp-cli doctor --fail-on=stale` in cron to gate scheduled syncs.

## Configuration

Config file: `~/.config/food52-pp-cli/config.toml` (override with `--config <path>`).

Environment variables:

| Var | Effect |
|-----|--------|
| `FOOD52_CONFIG` | Override config-file path |
| `FOOD52_BASE_URL` | Override the Food52 base URL (used by tests / mocks; you don't need this) |
| `FOOD52_FEEDBACK_ENDPOINT` | If set, `feedback` may POST entries to this URL (with `--send` or `FOOD52_FEEDBACK_AUTO_SEND=true`). Default is local-only. |
| `FOOD52_FEEDBACK_AUTO_SEND` | When `true`, every `feedback` entry is sent to the endpoint above. |
| `NO_COLOR`, `TERM=dumb` | Disable ANSI colors even with `--human-friendly`. |

Local data:

| Path | Contents |
|------|----------|
| `~/.local/share/food52-pp-cli/data.db` | SQLite store (synced recipes/articles, pantry, FTS index) |
| `~/.food52-pp-cli/feedback.jsonl` | `feedback` entries (append-only, local) |

## Cookbook

```bash
# Find a TK-approved chicken recipe under 5 ingredients, project just what an agent needs
food52-pp-cli recipes search chicken --tag 5-ingredients-or-fewer --json \
  --select 'hits.title,hits.slug,hits.test_kitchen_approved,hits.average_rating'

# Pull the structured recipe for piping into a meal planner
food52-pp-cli recipes get mom-s-japanese-curry-chicken-with-radish-and-cauliflower --json \
  --select 'title,ingredients,instructions,average_rating,yield'

# One-shot offline cookbook for a flight
food52-pp-cli sync recipes weeknight quick-and-easy 30-minutes-or-fewer
food52-pp-cli search 'weeknight' --json

# What can I make with what I have?
food52-pp-cli pantry add chicken garlic onion ginger lemon
food52-pp-cli sync recipes chicken --limit 50
food52-pp-cli pantry match --min-coverage 0.6 --json \
  --select 'matches.title,matches.coverage,matches.missing_ingredients'

# Cook from a recipe (no nav, no images, no comments)
food52-pp-cli print sarah-fennel-s-best-lunch-lady-brownie-recipe

# Scale a recipe to a different headcount
food52-pp-cli scale mom-s-japanese-curry-chicken-with-radish-and-cauliflower --servings 8 --json

# What's the editorial context behind a recipe?
food52-pp-cli sync articles food
food52-pp-cli articles for-recipe sarah-fennel-s-best-lunch-lady-brownie-recipe --json
```

## Troubleshooting

**Not found errors (exit code 3)**
- Check the slug is correct. Food52 occasionally renames slugs — run `recipes search '<title-fragment>'` to find the current canonical slug.

### API-specific

- **HTTP 429 or 'Vercel Security Checkpoint' on every request** — You are not using the printed CLI's transport. `curl` and stock Go `net/http` cannot clear Vercel's TLS-fingerprint challenge. Run requests through `food52-pp-cli` itself (which uses Surf + Chrome impersonation) or rebuild from source if your binary is older than the Surf integration.
- **recipes search returns 'Typesense key discovery failed'** — Run `food52-pp-cli doctor` — it re-fetches the public `/_app.js` bundle. Food52 rotates the bundle hash on each deploy; the CLI auto-recovers on the next call.
- **recipes get returns 404 for a slug that exists** — Food52 occasionally renames slugs. Try `recipes search '<title-fragment>'` to find the current canonical slug, or `open <slug>` to confirm in a browser.
- **scale fails with "no recipeYield"** — Some recipes ship a pan size instead of a serving count (e.g. `sarah-fennel-s-best-lunch-lady-brownie-recipe` returns "1 9x13" pan"). Scale is a no-op for those — pick a recipe whose yield reads "Serves: N" or "Makes N servings".
- **pantry match returns nothing** — Run `sync recipes <tag>` first. `pantry match` joins the local store, not the live site. After syncing, run `food52-pp-cli search` to confirm rows landed.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**hhursev/recipe-scrapers**](https://github.com/hhursev/recipe-scrapers) — Python (1900 stars)
- [**imRohan/food52-cli**](https://github.com/imRohan/food52-cli) — Ruby

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
