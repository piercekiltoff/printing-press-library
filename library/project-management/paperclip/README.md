# Paperclip CLI

**Complete control-plane CLI for Paperclip — fleet status, approvals, costs, and issue management in one fast binary.**

paperclip-pp-cli gives operators a daily-driver terminal interface to the Paperclip AI agent management platform. Manage your agent fleet, triage the approval queue, monitor costs by agent or project, and drive issues through their lifecycle — all with --json output and agent-native flags. Covers every endpoint the MCP server and TypeScript CLI expose, plus cross-endpoint fleet intelligence commands no other tool has.

## Install

### Go

```
go install github.com/mvanhorn/printing-press-library/library/project-management/paperclip/cmd/paperclip-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Authentication

Run `paperclip-pp-cli auth login` to open the browser-based CLI auth challenge flow. Or set PAPERCLIP_API_KEY to a board API key and PAPERCLIP_URL to your instance URL.

## Quick Start

```bash
# Authenticate via browser challenge flow
paperclip-pp-cli auth login


# Set your active company so you don't need --company-id everywhere
paperclip-pp-cli context use <companyId>


# Get a live fleet status overview
paperclip-pp-cli fleet


# Get all open issue identifiers
paperclip-pp-cli issues list --status todo --json | jq '.[].identifier'


# See everything waiting for human review
paperclip-pp-cli approvals queue


# Check spending by agent this month
paperclip-pp-cli costs by-agent --json

```

## Unique Features

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

## Usage

Run `paperclip-pp-cli --help` for the full command reference and flag list.

## Commands

### adapters

Manage adapters

- **`paperclip-pp-cli adapters create`** - Install an adapter
- **`paperclip-pp-cli adapters delete`** - Delete an adapter
- **`paperclip-pp-cli adapters list`** - List all adapters
- **`paperclip-pp-cli adapters update`** - Enable or disable an adapter

### admin

Manage admin

- **`paperclip-pp-cli admin create`** - Demote a user from instance admin
- **`paperclip-pp-cli admin create-users`** - Promote a user to instance admin
- **`paperclip-pp-cli admin get`** - Get company access for a user (admin)
- **`paperclip-pp-cli admin list`** - List all users (admin)
- **`paperclip-pp-cli admin update`** - Set company access for a user (admin)

### agents

Manage agents

- **`paperclip-pp-cli agents delete`** - Delete an agent
- **`paperclip-pp-cli agents get`** - Get an agent
- **`paperclip-pp-cli agents list`** - Get the current agent
- **`paperclip-pp-cli agents list-me`** - Get current agent inbox (lite)
- **`paperclip-pp-cli agents list-me-2`** - Get current agent assigned inbox items
- **`paperclip-pp-cli agents update`** - Update an agent

### approvals

Manage approvals

- **`paperclip-pp-cli approvals get`** - Get an approval

### assets

Manage assets


### attachments

Manage attachments

- **`paperclip-pp-cli attachments delete`** - Delete an attachment

### auth

Manage auth

- **`paperclip-pp-cli auth list`** - Get current session
- **`paperclip-pp-cli auth list-profile`** - Get current user profile
- **`paperclip-pp-cli auth update`** - Update current user profile

### board-claim

Manage board claim

- **`paperclip-pp-cli board-claim get`** - Get board claim details by token

### cli-auth

Manage cli auth

- **`paperclip-pp-cli cli-auth create`** - Create a CLI auth challenge
- **`paperclip-pp-cli cli-auth create-cliauth`** - Revoke current CLI auth session
- **`paperclip-pp-cli cli-auth create-cliauth-2`** - Approve a CLI auth challenge
- **`paperclip-pp-cli cli-auth create-cliauth-3`** - Cancel a CLI auth challenge
- **`paperclip-pp-cli cli-auth get`** - Get a CLI auth challenge
- **`paperclip-pp-cli cli-auth list`** - Get current CLI auth session

### companies

Manage companies

- **`paperclip-pp-cli companies create`** - Create a company
- **`paperclip-pp-cli companies create-import`** - Apply a company import (legacy route)
- **`paperclip-pp-cli companies create-import-2`** - Preview a company import (legacy route)
- **`paperclip-pp-cli companies delete`** - Delete a company
- **`paperclip-pp-cli companies get`** - Get a company
- **`paperclip-pp-cli companies list`** - List companies
- **`paperclip-pp-cli companies list-issues`** - Legacy — returns error directing to correct issues path
- **`paperclip-pp-cli companies list-stats`** - Company stats
- **`paperclip-pp-cli companies update`** - Update a company

### environment-leases

Manage environment leases

- **`paperclip-pp-cli environment-leases get`** - Get an environment lease

### environments

Manage environments

- **`paperclip-pp-cli environments delete`** - Delete an environment
- **`paperclip-pp-cli environments get`** - Get an environment
- **`paperclip-pp-cli environments update`** - Update an environment

### execution-workspaces

Manage execution workspaces

- **`paperclip-pp-cli execution-workspaces get`** - Get an execution workspace
- **`paperclip-pp-cli execution-workspaces update`** - Update an execution workspace

### feedback-traces

Manage feedback traces

- **`paperclip-pp-cli feedback-traces get`** - Get a feedback trace

### goals

Manage goals

- **`paperclip-pp-cli goals delete`** - Delete a goal
- **`paperclip-pp-cli goals get`** - Get a goal
- **`paperclip-pp-cli goals update`** - Update a goal

### health

Manage health

- **`paperclip-pp-cli health list`** - Health check

### heartbeat-runs

Manage heartbeat runs

- **`paperclip-pp-cli heartbeat-runs get`** - Get a heartbeat run

### instance

Manage instance

- **`paperclip-pp-cli instance create`** - Trigger a database backup
- **`paperclip-pp-cli instance list`** - List scheduler heartbeats
- **`paperclip-pp-cli instance list-settings`** - Get experimental instance settings
- **`paperclip-pp-cli instance list-settings-2`** - Get general instance settings
- **`paperclip-pp-cli instance update`** - Update experimental instance settings
- **`paperclip-pp-cli instance update-settings`** - Update general instance settings

### invites

Manage invites

- **`paperclip-pp-cli invites get`** - Get an invite by token

### issues

Manage issues

- **`paperclip-pp-cli issues delete`** - Delete an issue
- **`paperclip-pp-cli issues get`** - Get an issue
- **`paperclip-pp-cli issues list`** - Legacy — returns error directing to /api/companies/{companyId}/issues
- **`paperclip-pp-cli issues update`** - Update an issue

### join-requests

Manage join requests


### labels

Manage labels

- **`paperclip-pp-cli labels delete`** - Delete a label

### llms

Manage llms

- **`paperclip-pp-cli llms get`** - Get agent configuration for a specific adapter type
- **`paperclip-pp-cli llms list`** - Get agent configuration as plain text (for LLM context)
- **`paperclip-pp-cli llms list-agenticonstxt`** - Get agent icon names as plain text

### openapi-json

Manage openapi json

- **`paperclip-pp-cli openapi-json list`** - Get the generated OpenAPI document

### plugins

Manage plugins

- **`paperclip-pp-cli plugins create`** - Install a plugin
- **`paperclip-pp-cli plugins create-tools`** - Execute a plugin tool
- **`paperclip-pp-cli plugins delete`** - Delete a plugin
- **`paperclip-pp-cli plugins get`** - Get a plugin
- **`paperclip-pp-cli plugins list`** - List installed plugins
- **`paperclip-pp-cli plugins list-examples`** - List example plugins
- **`paperclip-pp-cli plugins list-tools`** - List plugin tools
- **`paperclip-pp-cli plugins list-uicontributions`** - List plugin UI contributions

### projects

Manage projects

- **`paperclip-pp-cli projects delete`** - Delete a project
- **`paperclip-pp-cli projects get`** - Get a project
- **`paperclip-pp-cli projects update`** - Update a project

### routine-triggers

Manage routine triggers

- **`paperclip-pp-cli routine-triggers create`** - Fire a public routine trigger
- **`paperclip-pp-cli routine-triggers delete`** - Delete a routine trigger
- **`paperclip-pp-cli routine-triggers update`** - Update a routine trigger

### routines

Manage routines

- **`paperclip-pp-cli routines get`** - Get a routine
- **`paperclip-pp-cli routines update`** - Update a routine

### secrets

Manage secrets

- **`paperclip-pp-cli secrets delete`** - Delete a secret
- **`paperclip-pp-cli secrets update`** - Update a secret

### sidebar-preferences

Manage sidebar preferences

- **`paperclip-pp-cli sidebar-preferences list`** - Get current user sidebar preferences
- **`paperclip-pp-cli sidebar-preferences update`** - Update current user sidebar preferences

### skills

Manage skills

- **`paperclip-pp-cli skills get`** - Get a skill by name
- **`paperclip-pp-cli skills list`** - List available skills
- **`paperclip-pp-cli skills list-index`** - Get skills index

### work-products

Manage work products

- **`paperclip-pp-cli work-products delete`** - Delete a work product
- **`paperclip-pp-cli work-products update`** - Update a work product

### workspace-operations

Manage workspace operations



## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
paperclip-pp-cli adapters list

# JSON for scripting and agents
paperclip-pp-cli adapters list --json

# Filter to specific fields
paperclip-pp-cli adapters list --json --select id,name,status

# Dry run — show the request without sending
paperclip-pp-cli adapters list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
paperclip-pp-cli adapters list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Retryable** - creates return "already exists" on retry, deletes return "already deleted"
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use as MCP Server

This CLI ships a companion MCP server for use with Claude Desktop, Cursor, and other MCP-compatible tools.

### Claude Code

```bash
claude mcp add paperclip paperclip-pp-mcp -e PAPERCLIP_BOARD_SESSION_AUTH=<your-key>
```

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "paperclip": {
      "command": "paperclip-pp-mcp",
      "env": {
        "PAPERCLIP_BOARD_SESSION_AUTH": "<your-key>"
      }
    }
  }
}
```

## Health Check

```bash
paperclip-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/paperclip-pp-cli/config.toml`

Environment variables:
- `PAPERCLIP_BOARD_SESSION_AUTH`

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `paperclip-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $PAPERCLIP_BOARD_SESSION_AUTH`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **401 Unauthorized** — Run `paperclip-pp-cli auth login` or export PAPERCLIP_API_KEY=<your-key>
- **Cannot connect to server** — Export PAPERCLIP_URL=http://your-instance:3100 — default is localhost:3100
- **Wrong company shown** — Run `paperclip-pp-cli context use <companyId>` to switch active company

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**paperclipai/paperclip mcp-server**](https://github.com/paperclipai/paperclip) — TypeScript
- [**paperclipai/paperclip cli**](https://github.com/paperclipai/paperclip) — TypeScript
- [**Wizarck/paperclip-mcp**](https://github.com/Wizarck/paperclip-mcp) — TypeScript

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
