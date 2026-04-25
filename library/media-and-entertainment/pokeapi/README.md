# Pokeapi CLI

All the Pokémon data you'll ever need in one place, easily accessible through a modern free open-source RESTful API.

## What is this?

This is a full RESTful API linked to an extensive database detailing everything about the Pokémon main game series.

We've covered everything from Pokémon to Berry Flavors.

## Where do I start?

We have awesome [documentation](https://pokeapi.co/docs/v2) on how to use this API. It takes minutes to get started.

This API will always be publicly available and will never require any extensive setup process to consume.

Created by [**Paul Hallett**](https://github.com/phalt) and other [**PokéAPI contributors***](https://github.com/PokeAPI/pokeapi#contributing) around the world. Pokémon and Pokémon character names are trademarks of Nintendo.

Learn more at [Pokeapi](https://pokeapi.co/docs/v2).

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pokeapi/cmd/pokeapi-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Set Up Credentials

Get your API key from your API provider's developer portal. The key typically looks like a long alphanumeric string.

```bash
export POKÉAPI_BASIC_AUTH="<paste-your-key>"
```

You can also persist this in your config file at `~/.config/pokéapi-pp-cli/config.toml`.

### 3. Verify Setup

```bash
pokeapi-pp-cli doctor
```

This checks your configuration and credentials.

### 4. Try Your First Command

```bash
pokeapi-pp-cli v2 ability-list
```

## Usage

Run `pokeapi-pp-cli --help` for the full command reference and flag list.

## Commands

### v2

Manage v2

- **`pokeapi-pp-cli v2 ability-list`** - Abilities provide passive effects for Pokémon in battle or in the overworld. Pokémon have multiple possible abilities but can have only one ability at a time. Check out [Bulbapedia](http://bulbapedia.bulbagarden.net/wiki/Ability) for greater detail.
- **`pokeapi-pp-cli v2 ability-retrieve`** - Abilities provide passive effects for Pokémon in battle or in the overworld. Pokémon have multiple possible abilities but can have only one ability at a time. Check out [Bulbapedia](http://bulbapedia.bulbagarden.net/wiki/Ability) for greater detail.
- **`pokeapi-pp-cli v2 berry-firmness-list`** - List berry firmness
- **`pokeapi-pp-cli v2 berry-firmness-retrieve`** - Get berry by firmness
- **`pokeapi-pp-cli v2 berry-flavor-list`** - List berry flavors
- **`pokeapi-pp-cli v2 berry-flavor-retrieve`** - Get berries by flavor
- **`pokeapi-pp-cli v2 berry-list`** - List berries
- **`pokeapi-pp-cli v2 berry-retrieve`** - Get a berry
- **`pokeapi-pp-cli v2 characteristic-list`** - List charecterictics
- **`pokeapi-pp-cli v2 characteristic-retrieve`** - Get characteristic
- **`pokeapi-pp-cli v2 contest-effect-list`** - List contest effects
- **`pokeapi-pp-cli v2 contest-effect-retrieve`** - Get contest effect
- **`pokeapi-pp-cli v2 contest-type-list`** - List contest types
- **`pokeapi-pp-cli v2 contest-type-retrieve`** - Get contest type
- **`pokeapi-pp-cli v2 egg-group-list`** - List egg groups
- **`pokeapi-pp-cli v2 egg-group-retrieve`** - Get egg group
- **`pokeapi-pp-cli v2 encounter-condition-list`** - List encounter conditions
- **`pokeapi-pp-cli v2 encounter-condition-retrieve`** - Get encounter condition
- **`pokeapi-pp-cli v2 encounter-condition-value-list`** - List encounter condition values
- **`pokeapi-pp-cli v2 encounter-condition-value-retrieve`** - Get encounter condition value
- **`pokeapi-pp-cli v2 encounter-method-list`** - List encounter methods
- **`pokeapi-pp-cli v2 encounter-method-retrieve`** - Get encounter method
- **`pokeapi-pp-cli v2 evolution-chain-list`** - List evolution chains
- **`pokeapi-pp-cli v2 evolution-chain-retrieve`** - Get evolution chain
- **`pokeapi-pp-cli v2 evolution-trigger-list`** - List evolution triggers
- **`pokeapi-pp-cli v2 evolution-trigger-retrieve`** - Get evolution trigger
- **`pokeapi-pp-cli v2 gender-list`** - List genders
- **`pokeapi-pp-cli v2 gender-retrieve`** - Get gender
- **`pokeapi-pp-cli v2 generation-list`** - List genrations
- **`pokeapi-pp-cli v2 generation-retrieve`** - Get genration
- **`pokeapi-pp-cli v2 growth-rate-list`** - List growth rates
- **`pokeapi-pp-cli v2 growth-rate-retrieve`** - Get growth rate
- **`pokeapi-pp-cli v2 item-attribute-list`** - List item attributes
- **`pokeapi-pp-cli v2 item-attribute-retrieve`** - Get item attribute
- **`pokeapi-pp-cli v2 item-category-list`** - List item categories
- **`pokeapi-pp-cli v2 item-category-retrieve`** - Get item category
- **`pokeapi-pp-cli v2 item-fling-effect-list`** - List item fling effects
- **`pokeapi-pp-cli v2 item-fling-effect-retrieve`** - Get item fling effect
- **`pokeapi-pp-cli v2 item-list`** - List items
- **`pokeapi-pp-cli v2 item-pocket-list`** - List item pockets
- **`pokeapi-pp-cli v2 item-pocket-retrieve`** - Get item pocket
- **`pokeapi-pp-cli v2 item-retrieve`** - Get item
- **`pokeapi-pp-cli v2 language-list`** - List languages
- **`pokeapi-pp-cli v2 language-retrieve`** - Get language
- **`pokeapi-pp-cli v2 location-area-list`** - List location areas
- **`pokeapi-pp-cli v2 location-area-retrieve`** - Get location area
- **`pokeapi-pp-cli v2 location-list`** - List locations
- **`pokeapi-pp-cli v2 location-retrieve`** - Get location
- **`pokeapi-pp-cli v2 machine-list`** - List machines
- **`pokeapi-pp-cli v2 machine-retrieve`** - Get machine
- **`pokeapi-pp-cli v2 move-ailment-list`** - List move meta ailments
- **`pokeapi-pp-cli v2 move-ailment-retrieve`** - Get move meta ailment
- **`pokeapi-pp-cli v2 move-battle-style-list`** - List move battle styles
- **`pokeapi-pp-cli v2 move-battle-style-retrieve`** - Get move battle style
- **`pokeapi-pp-cli v2 move-category-list`** - List move meta categories
- **`pokeapi-pp-cli v2 move-category-retrieve`** - Get move meta category
- **`pokeapi-pp-cli v2 move-damage-class-list`** - List move damage classes
- **`pokeapi-pp-cli v2 move-damage-class-retrieve`** - Get move damage class
- **`pokeapi-pp-cli v2 move-learn-method-list`** - List move learn methods
- **`pokeapi-pp-cli v2 move-learn-method-retrieve`** - Get move learn method
- **`pokeapi-pp-cli v2 move-list`** - List moves
- **`pokeapi-pp-cli v2 move-retrieve`** - Get move
- **`pokeapi-pp-cli v2 move-target-list`** - List move targets
- **`pokeapi-pp-cli v2 move-target-retrieve`** - Get move target
- **`pokeapi-pp-cli v2 nature-list`** - List natures
- **`pokeapi-pp-cli v2 nature-retrieve`** - Get nature
- **`pokeapi-pp-cli v2 pal-park-area-list`** - List pal park areas
- **`pokeapi-pp-cli v2 pal-park-area-retrieve`** - Get pal park area
- **`pokeapi-pp-cli v2 pokeathlon-stat-list`** - List pokeathlon stats
- **`pokeapi-pp-cli v2 pokeathlon-stat-retrieve`** - Get pokeathlon stat
- **`pokeapi-pp-cli v2 pokedex-list`** - List pokedex
- **`pokeapi-pp-cli v2 pokedex-retrieve`** - Get pokedex
- **`pokeapi-pp-cli v2 pokemon-color-list`** - List pokemon colors
- **`pokeapi-pp-cli v2 pokemon-color-retrieve`** - Get pokemon color
- **`pokeapi-pp-cli v2 pokemon-encounters-retrieve`** - Get pokemon encounter
- **`pokeapi-pp-cli v2 pokemon-form-list`** - List pokemon forms
- **`pokeapi-pp-cli v2 pokemon-form-retrieve`** - Get pokemon form
- **`pokeapi-pp-cli v2 pokemon-habitat-list`** - List pokemom habitas
- **`pokeapi-pp-cli v2 pokemon-habitat-retrieve`** - Get pokemom habita
- **`pokeapi-pp-cli v2 pokemon-list`** - List pokemon
- **`pokeapi-pp-cli v2 pokemon-retrieve`** - Get pokemon
- **`pokeapi-pp-cli v2 pokemon-shape-list`** - List pokemon shapes
- **`pokeapi-pp-cli v2 pokemon-shape-retrieve`** - Get pokemon shape
- **`pokeapi-pp-cli v2 pokemon-species-list`** - List pokemon species
- **`pokeapi-pp-cli v2 pokemon-species-retrieve`** - Get pokemon species
- **`pokeapi-pp-cli v2 region-list`** - List regions
- **`pokeapi-pp-cli v2 region-retrieve`** - Get region
- **`pokeapi-pp-cli v2 stat-list`** - List stats
- **`pokeapi-pp-cli v2 stat-retrieve`** - Get stat
- **`pokeapi-pp-cli v2 super-contest-effect-list`** - List super contest effects
- **`pokeapi-pp-cli v2 super-contest-effect-retrieve`** - Get super contest effect
- **`pokeapi-pp-cli v2 type-list`** - List types
- **`pokeapi-pp-cli v2 type-retrieve`** - Get types
- **`pokeapi-pp-cli v2 version-group-list`** - List version groups
- **`pokeapi-pp-cli v2 version-group-retrieve`** - Get version group
- **`pokeapi-pp-cli v2 version-list`** - List versions
- **`pokeapi-pp-cli v2 version-retrieve`** - Get version


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
pokeapi-pp-cli v2 ability-list

# JSON for scripting and agents
pokeapi-pp-cli v2 ability-list --json

# Filter to specific fields
pokeapi-pp-cli v2 ability-list --json --select id,name,status

# Dry run — show the request without sending
pokeapi-pp-cli v2 ability-list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
pokeapi-pp-cli v2 ability-list --agent
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
claude mcp add pokeapi pokeapi-pp-mcp -e POKÉAPI_BASIC_AUTH=<your-key>
```

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "pokeapi": {
      "command": "pokeapi-pp-mcp",
      "env": {
        "POKÉAPI_BASIC_AUTH": "<your-key>"
      }
    }
  }
}
```

## Health Check

```bash
pokeapi-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/pokéapi-pp-cli/config.toml`

Environment variables:
- `POKÉAPI_BASIC_AUTH`

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `pokeapi-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $POKÉAPI_BASIC_AUTH`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
