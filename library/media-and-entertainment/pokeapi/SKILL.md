---
name: pp-pokeapi
description: "Printing Press CLI for Pokeapi. All the Pokémon data you'll ever need in one place, easily accessible through a modern free open-source RESTful..."
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata: '{"openclaw":{"requires":{"bins":["pokeapi-pp-cli"]},"install":[{"id":"go","kind":"shell","command":"go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pokeapi/cmd/pokeapi-pp-cli@latest","bins":["pokeapi-pp-cli"],"label":"Install via go install"}]}}'
---

# Pokeapi — Printing Press CLI

All the Pokémon data you'll ever need in one place, easily accessible through a modern free open-source RESTful API.

## What is this?

This is a full RESTful API linked to an extensive database detailing everything about the Pokémon main game series.

We've covered everything from Pokémon to Berry Flavors.

## Where do I start?

We have awesome [documentation](https://pokeapi.co/docs/v2) on how to use this API. It takes minutes to get started.

This API will always be publicly available and will never require any extensive setup process to consume.

Created by [**Paul Hallett**](https://github.com/phalt) and other [**PokéAPI contributors***](https://github.com/PokeAPI/pokeapi#contributing) around the world. Pokémon and Pokémon character names are trademarks of Nintendo.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Command Reference

**v2** — Manage v2

- `pokeapi-pp-cli v2 ability-list` — Abilities provide passive effects for Pokémon in battle or in the overworld. Pokémon have multiple possible...
- `pokeapi-pp-cli v2 ability-retrieve` — Abilities provide passive effects for Pokémon in battle or in the overworld. Pokémon have multiple possible...
- `pokeapi-pp-cli v2 berry-firmness-list` — List berry firmness
- `pokeapi-pp-cli v2 berry-firmness-retrieve` — Get berry by firmness
- `pokeapi-pp-cli v2 berry-flavor-list` — List berry flavors
- `pokeapi-pp-cli v2 berry-flavor-retrieve` — Get berries by flavor
- `pokeapi-pp-cli v2 berry-list` — List berries
- `pokeapi-pp-cli v2 berry-retrieve` — Get a berry
- `pokeapi-pp-cli v2 characteristic-list` — List charecterictics
- `pokeapi-pp-cli v2 characteristic-retrieve` — Get characteristic
- `pokeapi-pp-cli v2 contest-effect-list` — List contest effects
- `pokeapi-pp-cli v2 contest-effect-retrieve` — Get contest effect
- `pokeapi-pp-cli v2 contest-type-list` — List contest types
- `pokeapi-pp-cli v2 contest-type-retrieve` — Get contest type
- `pokeapi-pp-cli v2 egg-group-list` — List egg groups
- `pokeapi-pp-cli v2 egg-group-retrieve` — Get egg group
- `pokeapi-pp-cli v2 encounter-condition-list` — List encounter conditions
- `pokeapi-pp-cli v2 encounter-condition-retrieve` — Get encounter condition
- `pokeapi-pp-cli v2 encounter-condition-value-list` — List encounter condition values
- `pokeapi-pp-cli v2 encounter-condition-value-retrieve` — Get encounter condition value
- `pokeapi-pp-cli v2 encounter-method-list` — List encounter methods
- `pokeapi-pp-cli v2 encounter-method-retrieve` — Get encounter method
- `pokeapi-pp-cli v2 evolution-chain-list` — List evolution chains
- `pokeapi-pp-cli v2 evolution-chain-retrieve` — Get evolution chain
- `pokeapi-pp-cli v2 evolution-trigger-list` — List evolution triggers
- `pokeapi-pp-cli v2 evolution-trigger-retrieve` — Get evolution trigger
- `pokeapi-pp-cli v2 gender-list` — List genders
- `pokeapi-pp-cli v2 gender-retrieve` — Get gender
- `pokeapi-pp-cli v2 generation-list` — List genrations
- `pokeapi-pp-cli v2 generation-retrieve` — Get genration
- `pokeapi-pp-cli v2 growth-rate-list` — List growth rates
- `pokeapi-pp-cli v2 growth-rate-retrieve` — Get growth rate
- `pokeapi-pp-cli v2 item-attribute-list` — List item attributes
- `pokeapi-pp-cli v2 item-attribute-retrieve` — Get item attribute
- `pokeapi-pp-cli v2 item-category-list` — List item categories
- `pokeapi-pp-cli v2 item-category-retrieve` — Get item category
- `pokeapi-pp-cli v2 item-fling-effect-list` — List item fling effects
- `pokeapi-pp-cli v2 item-fling-effect-retrieve` — Get item fling effect
- `pokeapi-pp-cli v2 item-list` — List items
- `pokeapi-pp-cli v2 item-pocket-list` — List item pockets
- `pokeapi-pp-cli v2 item-pocket-retrieve` — Get item pocket
- `pokeapi-pp-cli v2 item-retrieve` — Get item
- `pokeapi-pp-cli v2 language-list` — List languages
- `pokeapi-pp-cli v2 language-retrieve` — Get language
- `pokeapi-pp-cli v2 location-area-list` — List location areas
- `pokeapi-pp-cli v2 location-area-retrieve` — Get location area
- `pokeapi-pp-cli v2 location-list` — List locations
- `pokeapi-pp-cli v2 location-retrieve` — Get location
- `pokeapi-pp-cli v2 machine-list` — List machines
- `pokeapi-pp-cli v2 machine-retrieve` — Get machine
- `pokeapi-pp-cli v2 move-ailment-list` — List move meta ailments
- `pokeapi-pp-cli v2 move-ailment-retrieve` — Get move meta ailment
- `pokeapi-pp-cli v2 move-battle-style-list` — List move battle styles
- `pokeapi-pp-cli v2 move-battle-style-retrieve` — Get move battle style
- `pokeapi-pp-cli v2 move-category-list` — List move meta categories
- `pokeapi-pp-cli v2 move-category-retrieve` — Get move meta category
- `pokeapi-pp-cli v2 move-damage-class-list` — List move damage classes
- `pokeapi-pp-cli v2 move-damage-class-retrieve` — Get move damage class
- `pokeapi-pp-cli v2 move-learn-method-list` — List move learn methods
- `pokeapi-pp-cli v2 move-learn-method-retrieve` — Get move learn method
- `pokeapi-pp-cli v2 move-list` — List moves
- `pokeapi-pp-cli v2 move-retrieve` — Get move
- `pokeapi-pp-cli v2 move-target-list` — List move targets
- `pokeapi-pp-cli v2 move-target-retrieve` — Get move target
- `pokeapi-pp-cli v2 nature-list` — List natures
- `pokeapi-pp-cli v2 nature-retrieve` — Get nature
- `pokeapi-pp-cli v2 pal-park-area-list` — List pal park areas
- `pokeapi-pp-cli v2 pal-park-area-retrieve` — Get pal park area
- `pokeapi-pp-cli v2 pokeathlon-stat-list` — List pokeathlon stats
- `pokeapi-pp-cli v2 pokeathlon-stat-retrieve` — Get pokeathlon stat
- `pokeapi-pp-cli v2 pokedex-list` — List pokedex
- `pokeapi-pp-cli v2 pokedex-retrieve` — Get pokedex
- `pokeapi-pp-cli v2 pokemon-color-list` — List pokemon colors
- `pokeapi-pp-cli v2 pokemon-color-retrieve` — Get pokemon color
- `pokeapi-pp-cli v2 pokemon-encounters-retrieve` — Get pokemon encounter
- `pokeapi-pp-cli v2 pokemon-form-list` — List pokemon forms
- `pokeapi-pp-cli v2 pokemon-form-retrieve` — Get pokemon form
- `pokeapi-pp-cli v2 pokemon-habitat-list` — List pokemom habitas
- `pokeapi-pp-cli v2 pokemon-habitat-retrieve` — Get pokemom habita
- `pokeapi-pp-cli v2 pokemon-list` — List pokemon
- `pokeapi-pp-cli v2 pokemon-retrieve` — Get pokemon
- `pokeapi-pp-cli v2 pokemon-shape-list` — List pokemon shapes
- `pokeapi-pp-cli v2 pokemon-shape-retrieve` — Get pokemon shape
- `pokeapi-pp-cli v2 pokemon-species-list` — List pokemon species
- `pokeapi-pp-cli v2 pokemon-species-retrieve` — Get pokemon species
- `pokeapi-pp-cli v2 region-list` — List regions
- `pokeapi-pp-cli v2 region-retrieve` — Get region
- `pokeapi-pp-cli v2 stat-list` — List stats
- `pokeapi-pp-cli v2 stat-retrieve` — Get stat
- `pokeapi-pp-cli v2 super-contest-effect-list` — List super contest effects
- `pokeapi-pp-cli v2 super-contest-effect-retrieve` — Get super contest effect
- `pokeapi-pp-cli v2 type-list` — List types
- `pokeapi-pp-cli v2 type-retrieve` — Get types
- `pokeapi-pp-cli v2 version-group-list` — List version groups
- `pokeapi-pp-cli v2 version-group-retrieve` — Get version group
- `pokeapi-pp-cli v2 version-list` — List versions
- `pokeapi-pp-cli v2 version-retrieve` — Get version


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
pokeapi-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

Set your API key via environment variable:

```bash
export POKÉAPI_BASIC_AUTH="<your-key>"
```

Or persist it in `~/.config/pokéapi-pp-cli/config.toml`.

Run `pokeapi-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  pokeapi-pp-cli v2 ability-list --agent --select id,name,status
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
pokeapi-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
pokeapi-pp-cli feedback --stdin < notes.txt
pokeapi-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.pokeapi-pp-cli/feedback.jsonl`. They are never POSTed unless `POKEAPI_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `POKEAPI_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
pokeapi-pp-cli profile save briefing --json
pokeapi-pp-cli --profile briefing v2 ability-list
pokeapi-pp-cli profile list --json
pokeapi-pp-cli profile show briefing
pokeapi-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `pokeapi-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → CLI installation
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## CLI Installation

1. Check Go is installed: `go version` (requires Go 1.23+)
2. Install:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pokeapi/cmd/pokeapi-pp-cli@latest
   ```
3. Verify: `pokeapi-pp-cli --version`
4. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pokeapi/cmd/pokeapi-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add pokeapi-pp-mcp -- pokeapi-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which pokeapi-pp-cli`
   If not found, offer to install (see CLI Installation above).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   pokeapi-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `pokeapi-pp-cli <command> --help`.
