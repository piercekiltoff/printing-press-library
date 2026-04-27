---
name: pp-food52
description: "Search, browse, and read Food52 from your terminal — with offline FTS, pantry matching, recipe scaling, and the editorial signals other tools throw away. Trigger phrases: `find me a food52 recipe for X`, `scale this food52 recipe to N servings`, `what can I cook from food52 with what's in my pantry`, `use food52`, `run food52`."
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata: '{"openclaw":{"requires":{"bins":["food52-pp-cli"]},"install":[{"id":"go","kind":"shell","command":"go install github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/cmd/food52-pp-cli@latest","bins":["food52-pp-cli"],"label":"Install via go install"}]}}'
---

# Food52 — Printing Press CLI

Every recipe and article on Food52, queryable without a browser. Ships with `pantry match` (find recipes from what you already have), `search` (offline FTS over your synced cookbook), `recipes top` (Test-Kitchen approved + rating-floored), and `scale` (resize ingredient lists via JSON-LD). The only existing Food52 CLI is a 2018-era Ruby HTML scraper that no longer runs against today's Vercel-protected site; this is a clean rebuild on Surf with Chrome TLS impersonation.

## When to Use This CLI

Reach for this CLI when an agent needs Food52-quality recipes (community-curated, editor-tested) without rendering a browser or scraping HTML. The pantry match and offline FTS commands turn it into an edit-friendly cookbook the agent can keep on disk between sessions. Use the live `recipes search` for one-off lookups; sync + offline `search` for repeated queries against the same recipes.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

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

## HTTP Transport

Food52 sits behind Vercel's TLS-fingerprint bot challenge. The CLI clears it
at the transport layer using Surf with Chrome impersonation — no resident
browser, no clearance cookie, no env var. The Typesense search-only key for
`recipes search` is auto-discovered from Food52's public `/_app.js` bundle on
first use and re-discovered automatically when Food52 deploys a new bundle.

## Command Reference

**articles** — Browse and read Food52 stories (articles) from the food and life verticals

- `food52-pp-cli articles browse` — Browse the latest Food52 articles in a vertical (food, life)
- `food52-pp-cli articles get` — Get a Food52 article (story) by slug

**recipes** — Browse Food52 recipes by tag and fetch single recipe details (data extracted from SSR __NEXT_DATA__)

- `food52-pp-cli recipes browse` — Browse Food52 recipes filtered by a tag (e.g. chicken, breakfast, vegetarian)
- `food52-pp-cli recipes get` — Get full structured details for a single Food52 recipe by slug


**Hand-written commands**

- `food52-pp-cli recipes search <query>` — Search Food52 recipes via Typesense (host + search-only key auto-discovered from the public JS bundle)
- `food52-pp-cli articles browse-sub <vertical> <subvertical>` — Browse a Food52 article subvertical (e.g. food baking, food drinks, life travel)
- `food52-pp-cli tags list` — List Food52 recipe tags discovered from the homepage navigation (chicken, breakfast, vegetarian, dessert, pasta, ...)
- `food52-pp-cli sync recipes <tag> [<tag>...]` — Pull recipes for one or more tags into the local FTS-indexed store
- `food52-pp-cli sync articles <vertical> [<subvertical>]` — Pull article listings for a vertical (or subvertical) into the local store
- `food52-pp-cli search <query>` — Search the local store across recipes and articles (FTS5, offline)
- `food52-pp-cli pantry add <ingredient> [<ingredient>...]` — Add ingredients to your local pantry
- `food52-pp-cli pantry list` — Show your local pantry
- `food52-pp-cli pantry remove <ingredient>` — Remove an ingredient from your pantry
- `food52-pp-cli pantry match` — Find synced recipes whose ingredients match (or mostly match) your pantry
- `food52-pp-cli scale <slug-or-url>` — Scale a recipe's ingredients to a different number of servings (parses recipeYield from the recipe's JSON-LD)
- `food52-pp-cli print <slug-or-url>` — Print a clean cooking-friendly view of a recipe (ingredients + numbered steps, no nav, no images, ready to paste or...
- `food52-pp-cli open <slug-or-url>` — Resolve a Food52 recipe or article slug to its canonical URL (prints by default; pair with `--launch` to actually open in the browser — agents should leave `--launch` off)


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
food52-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Find a TK-approved chicken recipe under 5 ingredients

```bash
food52-pp-cli recipes search chicken --tag 5-ingredients-or-fewer --json --select 'hits.title,hits.slug,hits.test_kitchen_approved,hits.average_rating'
```

Typesense search filtered by tag, projecting just the fields an agent needs to pick a recipe.

### Get the full recipe in JSON for downstream meal planning

```bash
food52-pp-cli recipes get mom-s-japanese-curry-chicken-with-radish-and-cauliflower --json --select 'title,ingredients,instructions,average_rating,yield'
```

Pulls structured ingredients + steps for piping into a meal planner or shopping list builder.

### Build an offline weeknight cookbook in one shot

```bash
food52-pp-cli sync recipes weeknight quick-and-easy 30-minutes-or-fewer && food52-pp-cli search 'weeknight' --json
```

Pulls three high-signal tags into the local store, then queries via FTS5 — works on a plane.

### What can I make right now?

```bash
food52-pp-cli pantry add chicken garlic onion ginger lemon
food52-pp-cli sync recipes chicken --limit 50
food52-pp-cli pantry match --min-coverage 0.6 --json --select 'matches.title,matches.coverage,matches.missing_ingredients'
```

Seeds the pantry, syncs a tag's worth of recipes into the local store, then joins
the two — returning the ones you can mostly make. The first two commands are
one-time setup; rerun `pantry match` whenever the pantry changes.

### Cook from a recipe in one piped step

```bash
food52-pp-cli print sarah-fennel-s-best-lunch-lady-brownie-recipe | lp
```

Prints just the ingredients and numbered steps — no nav, no images, no comments —
ready to send to a printer or paste into notes.

### Scale a recipe to a different headcount

```bash
food52-pp-cli scale mom-s-japanese-curry-chicken-with-radish-and-cauliflower --servings 8 --json
```

Reads recipeYield from the page's JSON-LD and rewrites every quantity. Some
recipes don't ship a structured yield (e.g. `sarah-fennel-s-best-lunch-lady-brownie-recipe`
gives a pan size, not servings) and `scale` will surface that as an actionable error.

### Reverse-look-up: which articles cite this recipe?

```bash
food52-pp-cli sync articles food
food52-pp-cli articles for-recipe sarah-fennel-s-best-lunch-lady-brownie-recipe --json
```

Recipes link out to articles in `relatedReading`, but the site never shows the
reverse. After syncing the article corpus, this builds the reverse index locally.

## Auth Setup

No authentication required. No Food52 sign-in, no API key, no env var. The
CLI's Surf-Chrome transport clears Vercel's bot challenge automatically and
the Typesense search key is auto-discovered from Food52's public JS bundle.

Run `food52-pp-cli doctor` to verify the transport is working.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  food52-pp-cli articles get best-mothers-day-gift-ideas --agent --select title,author_name,published_at
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
food52-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
food52-pp-cli feedback --stdin < notes.txt
food52-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.food52-pp-cli/feedback.jsonl`. They are never POSTed unless `FOOD52_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `FOOD52_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
food52-pp-cli profile save briefing --json
food52-pp-cli --profile briefing articles get best-mothers-day-gift-ideas
food52-pp-cli profile list --json
food52-pp-cli profile show briefing
food52-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `food52-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → CLI installation
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## CLI Installation

1. Check Go is installed: `go version` (requires Go 1.25+)
2. Install:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/cmd/food52-pp-cli@latest
   ```
3. Verify: `food52-pp-cli --version`
4. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/cmd/food52-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add food52-pp-mcp -- food52-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which food52-pp-cli`
   If not found, offer to install (see CLI Installation above).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   food52-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `food52-pp-cli <command> --help`.
