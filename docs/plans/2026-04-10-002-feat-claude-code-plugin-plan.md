---
title: "feat: Add Claude Code plugin with unified discovery/install/use skill"
type: feat
status: active
date: 2026-04-10
---

# feat: Add Claude Code plugin with unified discovery/install/use skill

## Overview

Add a Claude Code plugin to the printing-press-library repo containing a unified skill (`/printing-press-library`) that lets users discover, install, and use any Printing Press CLI or MCP server conversationally. The skill acts as a router — parsing user intent and dispatching to discovery, installation, or direct CLI execution.

## Problem Frame

Users currently discover Printing Press tools by browsing `registry.json` or the README manually. There is no way to discover, install, or use these tools from within Claude Code or Codex without leaving the editor. The repo has no plugin infrastructure (`.claude-plugin/`, `skills/`).

The goal is a plugin with one skill that handles all four capabilities — discovery, CLI installation, MCP installation, and direct CLI usage — so the library doesn't need a separate skill per CLI as it scales to hundreds of entries.

## Requirements Trace

- R1. Plugin manifest at `.claude-plugin/plugin.json` with standard metadata
- R2. Single skill invocable as `/printing-press-library` that handles discovery, CLI install, MCP install, and direct CLI use
- R3. Lazy registry loading — read `registry.json` on demand via the Read tool, never inject into context
- R4. CLI binary name derivation from registry entries (handle `-pp-cli` suffix convention and outliers)
- R5. Installation via `go install` with path constructed from registry `path` field
- R6. MCP server installation via `go install` + `claude mcp add` using registry `mcp` block
- R7. Direct CLI usage via `--agent` flag for structured, token-efficient output
- R8. Semantic matching — natural language queries matched against registry descriptions to find the right CLI
- R9. Codex portability — frontmatter uses only portable fields, no Claude Code-only extensions
- R10. No changes to existing files (`registry.json`, `library/`, `README.md`)

## Scope Boundaries

- Plugin infrastructure in this repo only — no code changes to existing files
- SKILL.md body is procedural agent instructions, not end-user documentation
- No per-CLI skills — the unified skill handles all CLIs via registry lookup

### Deferred to Separate Tasks

- `marketplace.json` in cli-printing-press repo: separate PR to add marketplace reference pointing here
- GitHub Actions version-check workflow: future PR to enforce version bumps on registry/skill changes
- README update with plugin installation instructions: future PR
- Codex-specific plugin manifest: once Codex plugin spec stabilizes

## Context & Research

### Relevant Code and Patterns

- `registry.json` (repo root): 12 entries with fields `name`, `category`, `api`, `description`, `path`, and optional `mcp` block containing `binary`, `transport`, `tool_count`, `auth_type`, `env_vars`, `mcp_ready`
- All CLIs are Go modules under `library/<category>/<name>/` with `cmd/<binary>/` subdirectories
- CLI binary naming convention: most CLIs use `<name>-pp-cli` (e.g., `espn-pp-cli`, `slack-pp-cli`), but `agent-capture` uses just `agent-capture` (no `-pp-cli` suffix)
- Registry `name` field is inconsistent — some include the `-pp-cli` suffix (`dominos-pp-cli`, `hubspot-pp-cli`), others don't (`espn`, `linear`, `dub`)
- All CLIs with MCP support have the binary name explicitly in `mcp.binary`; CLI binary names are not explicit in the registry
- Existing plan at `docs/plans/2026-04-10-001-fix-cli-quality-bugs-plan.md` provides reference for plan conventions in this repo

### Plugin Structure Patterns

- `.claude-plugin/plugin.json` requires `name` field; `version`, `description`, `author`, `repository`, `keywords` are recommended
- SKILL.md frontmatter: `name`, `description` (third-person trigger phrases), `argument-hint`, `allowed-tools`
- `${CLAUDE_SKILL_DIR}` resolves to the skill's directory at runtime — use for file references in the skill body
- Plugins are installed via git clone into a cache directory — symlinks are preserved as symlinks within the clone (not resolved to copies), pointing to relative paths within the same cloned repo
- `allowed-tools` accepts comma or space-separated tool names; Bash commands can be filtered with patterns like `Bash(git *)`
- Skill body should be 1,500–5,000 words of imperative agent instructions with progressive disclosure to `references/` for detailed material

## Key Technical Decisions

- **One skill, not many**: A single "mega skill" routes to any CLI via registry lookup. This avoids needing hundreds of per-CLI skills as the library grows. The registry's `description` field enables semantic matching for natural-language queries.

- **Lazy registry loading**: The SKILL.md instructs the agent to `Read ${CLAUDE_SKILL_DIR}/registry.json` only when needed (discovery, name matching). This keeps the context clean — the registry is currently ~5KB (12 entries) but will grow to potentially hundreds.

- **Registry symlink**: `skills/printing-press-library/registry.json` is a symlink to `../../registry.json`. Plugins are installed via git clone, and git preserves symlinks by default on macOS. The symlink points to `../../registry.json` within the cloned repo, so the skill reads the registry from the clone's repo root. Updates come when the plugin is updated (git pull/fetch), not independently.

- **CLI binary name derivation**: The registry lacks an explicit `cli_binary` field (adding one is deferred per R10). The SKILL.md must derive binary names: if `name` already ends in `-pp-cli`, use as-is; otherwise try `<name>-pp-cli` first, then fall back to `<name>`. This is a temporary heuristic — once a `cli_binary` field is added to the registry schema, this derivation logic in SKILL.md should be replaced with a direct field lookup. Two contexts use this differently:
  - **Use mode** (CLI already installed): `which <name>-pp-cli`, then fall back to `which <name>`
  - **Install mode** (CLI not yet installed): try `go install .../cmd/<name>-pp-cli@latest`; if that fails with a "cannot find package" error, retry with `go install .../cmd/<name>@latest`

- **MCP binary name is explicit**: Unlike CLI binaries, MCP server binary names are always available via the `mcp.binary` field in the registry. MCP installation uses this field directly — no derivation needed.

- **`--agent` flag for CLI execution**: All standard Printing Press CLIs support `--agent` which enables `--json --compact --no-input --no-color --yes`. The skill always uses this flag for direct CLI usage to get structured, token-efficient output.

- **`allowed-tools: "Read Bash"`**: Read is needed for registry access. Bash is needed for `go install`, `which`, `--help`, `--version`, CLI execution, and `claude mcp add`. Bash cannot be scoped more narrowly because CLI binary names vary per entry.

- **No `AskUserQuestion` in allowed-tools**: The skill relies on natural conversation flow for disambiguation (ambiguous matches, install offers, API key requests). This keeps the tool set minimal and maintains Codex portability, since `AskUserQuestion` is Claude Code-specific.

## Open Questions

### Resolved During Planning

- **Should the skill use `references/` for detailed flow instructions?** No — the procedural instructions are the skill itself, not reference material. The SKILL.md body should be self-contained at ~2,000–3,000 words, well within the 5,000-word limit. Progressive disclosure to `references/` would fragment the core routing logic.

- **How to handle inconsistent registry `name` fields?** Derive binary names with a two-step check: try `<name>-pp-cli` first, fall back to `<name>`. For install paths, use the same derivation against the `cmd/` subdirectory. This handles both conventions without requiring registry changes.

- **Is `AskUserQuestion` needed?** No — keeping `allowed-tools` to `Read Bash` maintains Codex portability. The agent can ask questions through normal message output.

### Deferred to Implementation

- **Exact wording and formatting of discovery output**: The SKILL.md gives structural guidance (table, grouping by category) but exact formatting is left to the agent at runtime.

- **How semantic matching handles genuinely ambiguous queries**: The SKILL.md instructs the agent to present options when multiple CLIs match. The exact matching heuristic is the agent's judgment at execution time.

## Output Structure

```
.claude-plugin/
  plugin.json
skills/
  printing-press-library/
    SKILL.md
    registry.json → ../../registry.json  (symlink)
```

## Implementation Units

- [x] **Unit 1: Plugin manifest**

  **Goal:** Create the `.claude-plugin/plugin.json` manifest that registers this repo as a Claude Code plugin.

  **Requirements:** R1, R9

  **Dependencies:** None

  **Files:**
  - Create: `.claude-plugin/plugin.json`

  **Approach:**
  - Minimal required fields plus recommended metadata for discoverability
  - `skills` field points to `./skills/` for auto-discovery
  - No `mcpServers` — MCP servers are installed individually by the skill at runtime
  - Use only fields that are portable across Claude Code and Codex (`name`, `version`, `description`, `author`, `repository`, `license`, `keywords`, `skills`)

  **Patterns to follow:**
  - Standard plugin.json schema as documented by Claude Code plugin-dev
  - Existing production plugins (e.g., compound-engineering) for field ordering and conventions

  **Test expectation:** none — static manifest file with no behavioral logic

  **Verification:**
  - `.claude-plugin/plugin.json` exists and is valid JSON
  - Contains `name: "printing-press-library"`, `version: "1.0.0"`, `skills: "./skills/"`

- [x] **Unit 2: Unified SKILL.md**

  **Goal:** Write the skill that handles all four capabilities — discovery, CLI installation, MCP installation, and direct CLI usage — based on argument parsing and registry lookup.

  **Requirements:** R2, R3, R4, R5, R6, R7, R8, R9

  **Dependencies:** None (creates its own directory tree independent of Unit 1)

  **Files:**
  - Create: `skills/printing-press-library/SKILL.md`

  **Approach:**
  - Frontmatter: `name`, `description` (with third-person trigger phrases), `argument-hint`, `allowed-tools: "Read Bash"`
  - Body structure (imperative agent instructions):
    1. Role definition — unified router for the Printing Press CLI library
    2. Registry reference — `${CLAUDE_SKILL_DIR}/registry.json`, read on demand only
    3. Argument parsing rules to determine intent:
       - Empty → Discovery mode
       - Starts with `install` followed by a CLI name → Install mode (CLI or MCP)
       - First word matches a registry entry `name` → Explicit Use mode
       - Anything else → Semantic Use mode (match query against registry descriptions)
    4. Discovery mode — read registry, show catalog (all or filtered by query), suggest install/use commands
    5. CLI installation — read registry, check Go, derive binary name (see Key Technical Decisions), `go install` with path from registry, verify, show auth env vars
    6. MCP installation — read registry, check `mcp` block exists, install MCP binary using explicit `mcp.binary` field, `claude mcp add` with env vars from `mcp.env_vars`
    7. Explicit Use mode — match name, check `which` (try both `<name>-pp-cli` and `<name>`), run `--help` to discover commands, construct invocation with `--agent`
    8. Semantic Use mode — scan descriptions for relevance, single strong match → proceed as explicit use, multiple matches → present options, no match → show catalog
    9. Important notes section: `--agent` flag, binary name derivation rule, Go 1.23+ requirement, exit code meanings
  - Keep body under 3,000 words — all routing logic in one file, no `references/` splitting

  **Patterns to follow:**
  - Existing production SKILL.md files use imperative/infinitive form ("Parse the arguments", not "You should parse")
  - Third-person in description frontmatter ("This skill should be used when...")
  - `${CLAUDE_SKILL_DIR}` for file references within the skill body

  **Test scenarios:**
  - Happy path: Empty invocation reads registry and shows grouped catalog of all 12 CLIs
  - Happy path: `install espn cli` reads registry, runs `go install .../espn/cmd/espn-pp-cli@latest`, verifies with `espn-pp-cli --version`
  - Happy path: `install espn mcp` reads registry `mcp` block, installs `espn-pp-mcp`, registers with `claude mcp add`
  - Happy path: `espn lakers score` matches "espn" to registry, checks `which espn-pp-cli`, runs `--help`, constructs `espn-pp-cli scores basketball nba --agent`
  - Happy path: `lakers score` (semantic) scans descriptions, matches ESPN, proceeds as explicit use
  - Edge case: `install agent-capture mcp` — no `mcp` block in registry, skill explains MCP is not available for this CLI
  - Edge case: `games` (ambiguous) — matches ESPN and Steam Web, skill presents options for user to choose
  - Edge case: `weather` (no match) — no relevant CLI found, skill shows full catalog
  - Edge case: `agent-capture record` — binary is `agent-capture` not `agent-capture-pp-cli`, skill's two-step `which` check handles this
  - Error path: CLI not installed when user tries to use it — skill offers to install first
  - Error path: Go not installed — skill detects missing `go` binary and provides install guidance
  - Error path: CLI execution returns exit code 4 (auth) — skill shows required env vars from registry
  - Integration: Semantic match → install → use flow — user says "lakers score", ESPN not installed, skill offers install, installs, then executes the query

  **Verification:**
  - `skills/printing-press-library/SKILL.md` exists with valid YAML frontmatter
  - Frontmatter contains `name: printing-press-library`, `description` with trigger phrases, `argument-hint`, `allowed-tools: "Read Bash"`
  - Body covers all four modes (discovery, CLI install, MCP install, use) with clear argument-parsing rules
  - Binary name derivation rule is documented (try `-pp-cli` suffix first, fall back to bare name)
  - Body is under 5,000 words

- [x] **Unit 3: Registry symlink**

  **Goal:** Create a symlink from the skill directory to the canonical `registry.json` at repo root, so the installed plugin has access to the registry without maintaining a copy.

  **Requirements:** R3, R10

  **Dependencies:** Unit 2 (skill directory must exist)

  **Files:**
  - Create: `skills/printing-press-library/registry.json` (symlink → `../../registry.json`)

  **Approach:**
  - `cd skills/printing-press-library && ln -s ../../registry.json registry.json`
  - Plugins are installed via git clone, which preserves symlinks on macOS. The relative symlink `../../registry.json` points to the repo root's `registry.json` within the cloned repo
  - The installed plugin reads the registry from the clone, which updates when the plugin is updated (git pull/fetch)
  - Relative symlink path `../../registry.json` reaches from `skills/printing-press-library/` up to repo root

  **Patterns to follow:**
  - Git preserves symlinks by default (`core.symlinks=true` on macOS)

  **Test scenarios:**
  - Happy path: `readlink skills/printing-press-library/registry.json` returns `../../registry.json`
  - Happy path: `cat skills/printing-press-library/registry.json` returns the same content as `cat registry.json`
  - Edge case: Symlink target exists and is valid JSON with `schema_version` and `entries` fields

  **Verification:**
  - `skills/printing-press-library/registry.json` is a symlink pointing to `../../registry.json`
  - Following the symlink yields valid JSON matching repo root `registry.json`

## System-Wide Impact

- **Interaction graph:** The skill invokes external CLIs (`<name>-pp-cli --agent`), `go install`, `which`, and `claude mcp add`. No callbacks, middleware, or observers are affected. The skill is self-contained within its own invocation.
- **Error propagation:** CLI exit codes (0=success, 2=usage, 3=not found, 4=auth, 5=API, 7=rate limited) propagate through Bash tool results. The SKILL.md instructs the agent to interpret these and advise the user.
- **State lifecycle risks:** `go install` modifies `$GOPATH/bin`. `claude mcp add` modifies the user's MCP server configuration. Both are user-visible side effects that should be confirmed before execution.
- **API surface parity:** The plugin is Claude Code/Codex only. No REST API, webhook, or other interface.
- **Unchanged invariants:** `registry.json` schema, `library/` directory structure, README content, and all existing CLI behavior are explicitly unchanged by this plan.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| CLI binary name derivation fails for future CLIs with non-standard naming | Two-step check (try `-pp-cli` suffix, then bare name) handles known cases. Temporary heuristic until `cli_binary` field is added to registry schema |
| `hubspot-pp-cli` registry `path` doesn't match its `go.mod` module path | Pre-existing bug: `go.mod` declares module as `.../hubspot-pp-cli` but registry `path` is `.../hubspot`. Install will fail until the registry entry or go.mod is corrected. Out of scope per R10 — note in SKILL.md error handling |
| Registry grows to hundreds of entries, making Read tool output large | At ~500 bytes per entry, 200 entries ≈ 100KB — still well within a single Read. Monitor and consider category-based filtering if needed |
| `go install` requires Go 1.23+ which user may not have | SKILL.md checks `go version` first and provides install guidance |
| Registry updates require plugin reinstall to propagate | Plugin is installed via git clone; symlink points within the clone. Updates come via marketplace plugin update. Document in future README |
| `agent-capture` has no `-pp-cli` suffix and no MCP block | Binary name derivation fallback handles this. Semantic matching still finds it by description |

## Sources & References

- Registry schema: `registry.json` (repo root, 12 entries, schema_version 1)
- Existing plan convention: `docs/plans/2026-04-10-001-fix-cli-quality-bugs-plan.md`
- Claude Code plugin structure: official plugin-dev documentation
- CLI directory pattern: `library/<category>/<name>/cmd/<binary>/`
