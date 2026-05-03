---
name: pp-paperclip
description: "Complete control-plane CLI for Paperclip — fleet status, approvals, costs, and issue management in one fast binary. Trigger phrases: `check my agent fleet status`, `list pending approvals in Paperclip`, `show Paperclip costs by agent`, `find stale issues`, `manage Paperclip routines`, `use paperclip`, `run paperclip cli`."
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata: '{"openclaw":{"requires":{"bins":["paperclip-pp-cli"]},"install":[{"id":"go","kind":"shell","command":"go install github.com/mvanhorn/printing-press-library/library/project-management/paperclip/cmd/paperclip-pp-cli@latest","bins":["paperclip-pp-cli"],"label":"Install via go install"}]}}'
---

# Paperclip — Printing Press CLI

paperclip-pp-cli gives operators a daily-driver terminal interface to the Paperclip AI agent management platform. Manage your agent fleet, triage the approval queue, monitor costs by agent or project, and drive issues through their lifecycle — all with --json output and agent-native flags. Covers every endpoint the MCP server and TypeScript CLI expose, plus cross-endpoint fleet intelligence commands no other tool has.

## When to Use This CLI

Use paperclip-pp-cli when you need to manage or inspect a Paperclip AI agent management platform instance from the terminal. It is the right tool for fleet monitoring, approval triage, cost analysis, and issue lifecycle operations. It is not a replacement for agents running tasks — it is the operator control plane.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Fleet intelligence

- **`fleet`** — See live status, costs, active issues, and idle time for every agent at once.

  _Use this instead of calling individual agent endpoints when you need a situational overview of your entire agent fleet._

  ```bash
  paperclip-pp-cli fleet --agent
  ```
- **`approvals queue`** — Shows pending approvals with their linked issues, waiting agent, and time-in-queue.

  _Use when you need to triage all pending human-approval gates across a company without clicking through the UI._

  ```bash
  paperclip-pp-cli approvals queue --json
  ```
- **`issues stale`** — Finds in-progress issues with no agent activity in N days.

  _Use to detect stuck or forgotten work that an agent checked out but stopped progressing._

  ```bash
  paperclip-pp-cli issues stale --days 3 --agent
  ```
- **`agents timeline`** — Chronological view of an agent's runs, comments, and sessions in one stream.

  _Use to understand what an agent has been doing and why issues have stalled._

  ```bash
  paperclip-pp-cli agents timeline <agentId> --limit 20 --agent
  ```

### Cost intelligence

- **`costs anomalies`** — Flags agents spending significantly more than their 30-day average.

  _Use to catch runaway agents before they blow the budget._

  ```bash
  paperclip-pp-cli costs anomalies --threshold 2.0 --agent
  ```

### Automation health

- **`routines health`** — Shows which routines have consecutive failures, are overdue, or have high error rates.

  _Use to find broken scheduled automation before users notice._

  ```bash
  paperclip-pp-cli routines health --agent
  ```

## Command Reference

**adapters** — Manage adapters

- `paperclip-pp-cli adapters create` — Install an adapter
- `paperclip-pp-cli adapters delete` — Delete an adapter
- `paperclip-pp-cli adapters list` — List all adapters
- `paperclip-pp-cli adapters update` — Enable or disable an adapter

**admin** — Manage admin

- `paperclip-pp-cli admin create` — Demote a user from instance admin
- `paperclip-pp-cli admin create-users` — Promote a user to instance admin
- `paperclip-pp-cli admin get` — Get company access for a user (admin)
- `paperclip-pp-cli admin list` — List all users (admin)
- `paperclip-pp-cli admin update` — Set company access for a user (admin)

**agents** — Manage agents

- `paperclip-pp-cli agents delete` — Delete an agent
- `paperclip-pp-cli agents get` — Get an agent
- `paperclip-pp-cli agents list` — Get the current agent
- `paperclip-pp-cli agents list-me` — Get current agent inbox (lite)
- `paperclip-pp-cli agents list-me-2` — Get current agent assigned inbox items
- `paperclip-pp-cli agents update` — Update an agent

**approvals** — Manage approvals

- `paperclip-pp-cli approvals get` — Get an approval

**assets** — Manage assets


**attachments** — Manage attachments

- `paperclip-pp-cli attachments delete` — Delete an attachment

**auth** — Manage auth

- `paperclip-pp-cli auth list` — Get current session
- `paperclip-pp-cli auth list-profile` — Get current user profile
- `paperclip-pp-cli auth update` — Update current user profile

**board-claim** — Manage board claim

- `paperclip-pp-cli board-claim get` — Get board claim details by token

**cli-auth** — Manage cli auth

- `paperclip-pp-cli cli-auth create` — Create a CLI auth challenge
- `paperclip-pp-cli cli-auth create-cliauth` — Revoke current CLI auth session
- `paperclip-pp-cli cli-auth create-cliauth-2` — Approve a CLI auth challenge
- `paperclip-pp-cli cli-auth create-cliauth-3` — Cancel a CLI auth challenge
- `paperclip-pp-cli cli-auth get` — Get a CLI auth challenge
- `paperclip-pp-cli cli-auth list` — Get current CLI auth session

**companies** — Manage companies

- `paperclip-pp-cli companies create` — Create a company
- `paperclip-pp-cli companies create-import` — Apply a company import (legacy route)
- `paperclip-pp-cli companies create-import-2` — Preview a company import (legacy route)
- `paperclip-pp-cli companies delete` — Delete a company
- `paperclip-pp-cli companies get` — Get a company
- `paperclip-pp-cli companies list` — List companies
- `paperclip-pp-cli companies list-issues` — Legacy — returns error directing to correct issues path
- `paperclip-pp-cli companies list-stats` — Company stats
- `paperclip-pp-cli companies update` — Update a company

**environment-leases** — Manage environment leases

- `paperclip-pp-cli environment-leases get` — Get an environment lease

**environments** — Manage environments

- `paperclip-pp-cli environments delete` — Delete an environment
- `paperclip-pp-cli environments get` — Get an environment
- `paperclip-pp-cli environments update` — Update an environment

**execution-workspaces** — Manage execution workspaces

- `paperclip-pp-cli execution-workspaces get` — Get an execution workspace
- `paperclip-pp-cli execution-workspaces update` — Update an execution workspace

**feedback-traces** — Manage feedback traces

- `paperclip-pp-cli feedback-traces get` — Get a feedback trace

**goals** — Manage goals

- `paperclip-pp-cli goals delete` — Delete a goal
- `paperclip-pp-cli goals get` — Get a goal
- `paperclip-pp-cli goals update` — Update a goal

**health** — Manage health

- `paperclip-pp-cli health list` — Health check

**heartbeat-runs** — Manage heartbeat runs

- `paperclip-pp-cli heartbeat-runs get` — Get a heartbeat run

**instance** — Manage instance

- `paperclip-pp-cli instance create` — Trigger a database backup
- `paperclip-pp-cli instance list` — List scheduler heartbeats
- `paperclip-pp-cli instance list-settings` — Get experimental instance settings
- `paperclip-pp-cli instance list-settings-2` — Get general instance settings
- `paperclip-pp-cli instance update` — Update experimental instance settings
- `paperclip-pp-cli instance update-settings` — Update general instance settings

**invites** — Manage invites

- `paperclip-pp-cli invites get` — Get an invite by token

**issues** — Manage issues

- `paperclip-pp-cli issues delete` — Delete an issue
- `paperclip-pp-cli issues get` — Get an issue
- `paperclip-pp-cli issues list` — Legacy — returns error directing to /api/companies/{companyId}/issues
- `paperclip-pp-cli issues update` — Update an issue

**join-requests** — Manage join requests


**labels** — Manage labels

- `paperclip-pp-cli labels delete` — Delete a label

**llms** — Manage llms

- `paperclip-pp-cli llms get` — Get agent configuration for a specific adapter type
- `paperclip-pp-cli llms list` — Get agent configuration as plain text (for LLM context)
- `paperclip-pp-cli llms list-agenticonstxt` — Get agent icon names as plain text

**openapi-json** — Manage openapi json

- `paperclip-pp-cli openapi-json list` — Get the generated OpenAPI document

**plugins** — Manage plugins

- `paperclip-pp-cli plugins create` — Install a plugin
- `paperclip-pp-cli plugins create-tools` — Execute a plugin tool
- `paperclip-pp-cli plugins delete` — Delete a plugin
- `paperclip-pp-cli plugins get` — Get a plugin
- `paperclip-pp-cli plugins list` — List installed plugins
- `paperclip-pp-cli plugins list-examples` — List example plugins
- `paperclip-pp-cli plugins list-tools` — List plugin tools
- `paperclip-pp-cli plugins list-uicontributions` — List plugin UI contributions

**projects** — Manage projects

- `paperclip-pp-cli projects delete` — Delete a project
- `paperclip-pp-cli projects get` — Get a project
- `paperclip-pp-cli projects update` — Update a project

**routine-triggers** — Manage routine triggers

- `paperclip-pp-cli routine-triggers create` — Fire a public routine trigger
- `paperclip-pp-cli routine-triggers delete` — Delete a routine trigger
- `paperclip-pp-cli routine-triggers update` — Update a routine trigger

**routines** — Manage routines

- `paperclip-pp-cli routines get` — Get a routine
- `paperclip-pp-cli routines update` — Update a routine

**secrets** — Manage secrets

- `paperclip-pp-cli secrets delete` — Delete a secret
- `paperclip-pp-cli secrets update` — Update a secret

**sidebar-preferences** — Manage sidebar preferences

- `paperclip-pp-cli sidebar-preferences list` — Get current user sidebar preferences
- `paperclip-pp-cli sidebar-preferences update` — Update current user sidebar preferences

**skills** — Manage skills

- `paperclip-pp-cli skills get` — Get a skill by name
- `paperclip-pp-cli skills list` — List available skills
- `paperclip-pp-cli skills list-index` — Get skills index

**work-products** — Manage work products

- `paperclip-pp-cli work-products delete` — Delete a work product
- `paperclip-pp-cli work-products update` — Update a work product

**workspace-operations** — Manage workspace operations



### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
paperclip-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Triage stuck agents

```bash
paperclip-pp-cli issues stale --days 2 --agent
```

Find all in-progress issues with no agent activity in 48+ hours

### Approve a batch of pending approvals

```bash
paperclip-pp-cli approvals queue --json | jq '.[].id' | xargs -I{} paperclip-pp-cli approvals decide {} approve
```

Pipe the approval queue into a bulk approve loop

### Cost overview for current month

```bash
paperclip-pp-cli costs by-agent --json --select agentName,spendCents --agent
```

Get per-agent spending in structured output for downstream processing

### Wake up all idle agents

```bash
paperclip-pp-cli agents list --status idle --json | jq '.[].id' | xargs -I{} paperclip-pp-cli agents wakeup {}
```

Fan out a wakeup signal to every idle agent

### Monitor routine health

```bash
paperclip-pp-cli routines health --agent
```

Scan all routines for consecutive failures and overdue schedules

## Auth Setup

Run `paperclip-pp-cli auth login` to open the browser-based CLI auth challenge flow. Or set PAPERCLIP_API_KEY to a board API key and PAPERCLIP_URL to your instance URL.

Run `paperclip-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  paperclip-pp-cli adapters list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag

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
paperclip-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
paperclip-pp-cli feedback --stdin < notes.txt
paperclip-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.paperclip-pp-cli/feedback.jsonl`. They are never POSTed unless `PAPERCLIP_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `PAPERCLIP_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
paperclip-pp-cli profile save briefing --json
paperclip-pp-cli --profile briefing adapters list
paperclip-pp-cli profile list --json
paperclip-pp-cli profile show briefing
paperclip-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `paperclip-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → CLI installation
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## CLI Installation

1. Check Go is installed: `go version` (requires Go 1.23+)
2. Install:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/project-management/paperclip/cmd/paperclip-pp-cli@latest
   ```
3. Verify: `paperclip-pp-cli --version`
4. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/project-management/paperclip/cmd/paperclip-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add paperclip-pp-mcp -- paperclip-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which paperclip-pp-cli`
   If not found, offer to install (see CLI Installation above).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   paperclip-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `paperclip-pp-cli <command> --help`.
