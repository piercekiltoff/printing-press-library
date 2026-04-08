# linear-pp-cli

CLI for the Linear project management GraphQL API

## Install

### Homebrew

```
brew install user/tap/linear-pp-cli
```

### Go

```
go install github.com/mvanhorn/printing-press-library/library/project-management/linear/cmd/linear-pp-cli@latest
```

### Binary

Download from [Releases](https://github.com/mvanhorn/printing-press-library/releases).

## Quick Start

```bash
# 1. Set your API credentials
export LINEAR_API_KEY="your-key-here"

# 2. Verify everything works
linear-pp-cli doctor

# 3. Start using it
linear-pp-cli attachments --help
```

## Usage

```bash
# Sync your workspace to local SQLite (enables offline commands)
linear-pp-cli workflow archive

# List issues with filters
linear-pp-cli issues list --teamid <uuid> --first 20

# Create an issue
linear-pp-cli issues create --title "Fix login bug" --teamid <uuid> --priority 1

# Sprint analytics
linear-pp-cli sprint status
linear-pp-cli sprint burndown --team ENG
linear-pp-cli sprint velocity -n 8

# Team leaderboard
linear-pp-cli leaderboard --weeks 4 --sort closed

# Find duplicates before creating
linear-pp-cli similar "login page broken"

# Cross-entity search (issues + docs + projects)
linear-pp-cli search "auth timeout" --scope all

# Dependency graph
linear-pp-cli deps LIN-123

# Bottleneck detection
linear-pp-cli bottleneck --team ENG

# Bulk operations
linear-pp-cli bulk update-state --stateid <uuid> ID1 ID2 ID3

# Raw GraphQL
linear-pp-cli graphql --query '{ viewer { id name } }'
```

## Commands

### attachments

Operations on issue attachments

- **`linear-pp-cli attachments create`** - Create an attachment on an issue
- **`linear-pp-cli attachments delete`** - Delete an attachment
- **`linear-pp-cli attachments get`** - Get an attachment by ID
- **`linear-pp-cli attachments list`** - List attachments
- **`linear-pp-cli attachments update`** - Update an attachment

### comments

Operations on issue comments

- **`linear-pp-cli comments create`** - Create a comment on an issue
- **`linear-pp-cli comments delete`** - Delete a comment
- **`linear-pp-cli comments get`** - Get a single comment by ID
- **`linear-pp-cli comments list`** - List comments on an issue
- **`linear-pp-cli comments update`** - Update a comment

### cycles

Operations on cycles (sprints)

- **`linear-pp-cli cycles create`** - Create a new cycle
- **`linear-pp-cli cycles get`** - Get a cycle by ID
- **`linear-pp-cli cycles list`** - List cycles
- **`linear-pp-cli cycles update`** - Update a cycle

### documents

Operations on documents

- **`linear-pp-cli documents create`** - Create a document
- **`linear-pp-cli documents get`** - Get a document by ID
- **`linear-pp-cli documents list`** - List documents
- **`linear-pp-cli documents update`** - Update a document

### favorites

Operations on favorites

- **`linear-pp-cli favorites create`** - Create a favorite
- **`linear-pp-cli favorites delete`** - Delete a favorite
- **`linear-pp-cli favorites list`** - List favorites for the authenticated user

### issue_relations

Operations on issue relations (blocks, duplicates, related)

- **`linear-pp-cli issue_relations create`** - Create an issue relation
- **`linear-pp-cli issue_relations delete`** - Delete an issue relation
- **`linear-pp-cli issue_relations list`** - List issue relations

### issues

Operations on issues

- **`linear-pp-cli issues create`** - Create a new issue
- **`linear-pp-cli issues delete`** - Delete an issue (archive)
- **`linear-pp-cli issues get`** - Get a single issue by ID
- **`linear-pp-cli issues list`** - List issues with optional filtering
- **`linear-pp-cli issues search`** - Search issues by term
- **`linear-pp-cli issues update`** - Update an existing issue

### labels

Operations on issue labels

- **`linear-pp-cli labels create`** - Create a new label
- **`linear-pp-cli labels delete`** - Delete a label
- **`linear-pp-cli labels get`** - Get a label by ID
- **`linear-pp-cli labels list`** - List issue labels
- **`linear-pp-cli labels update`** - Update a label

### notifications

Operations on notifications

- **`linear-pp-cli notifications archive`** - Archive a notification
- **`linear-pp-cli notifications list`** - List notifications for the authenticated user

### organization

Operations on the organization

- **`linear-pp-cli organization get`** - Get the current organization

### project_updates

Operations on project updates

- **`linear-pp-cli project_updates create`** - Create a project update
- **`linear-pp-cli project_updates get`** - Get a project update by ID
- **`linear-pp-cli project_updates list`** - List project updates

### projects

Operations on projects

- **`linear-pp-cli projects create`** - Create a new project
- **`linear-pp-cli projects delete`** - Delete (archive) a project
- **`linear-pp-cli projects get`** - Get a project by ID
- **`linear-pp-cli projects list`** - List projects
- **`linear-pp-cli projects update`** - Update a project

### teams

Operations on teams

- **`linear-pp-cli teams create`** - Create a new team
- **`linear-pp-cli teams get`** - Get a team by ID
- **`linear-pp-cli teams list`** - List all teams
- **`linear-pp-cli teams update`** - Update a team

### users

Operations on users

- **`linear-pp-cli users get`** - Get a user by ID
- **`linear-pp-cli users list`** - List users in the organization
- **`linear-pp-cli users me`** - Get the authenticated user

### webhooks

Operations on webhooks

- **`linear-pp-cli webhooks create`** - Create a webhook
- **`linear-pp-cli webhooks delete`** - Delete a webhook
- **`linear-pp-cli webhooks get`** - Get a webhook by ID
- **`linear-pp-cli webhooks list`** - List webhooks
- **`linear-pp-cli webhooks update`** - Update a webhook

### workflow_states

Operations on workflow states

- **`linear-pp-cli workflow_states create`** - Create a workflow state
- **`linear-pp-cli workflow_states get`** - Get a workflow state by ID
- **`linear-pp-cli workflow_states list`** - List workflow states
- **`linear-pp-cli workflow_states update`** - Update a workflow state


### bottleneck

Detect who/what is blocking the most issues (local data)

- **`linear-pp-cli bottleneck`** - Show top blocking issues and people
- Flags: `--team`, `--limit`

### bulk

Bulk operations on issues

- **`linear-pp-cli bulk update-state`** - Batch-update workflow state
- **`linear-pp-cli bulk assign`** - Batch-assign to a user
- **`linear-pp-cli bulk label`** - Batch-add labels

### deps

Show dependency graph for an issue (local data)

- **`linear-pp-cli deps <issue-id>`** - Recursive block chain traversal
- Flags: `--depth`

### graphql

Execute raw GraphQL queries/mutations

- **`linear-pp-cli graphql --query '...'`** - Inline query
- **`linear-pp-cli graphql --file query.graphql`** - From file
- Supports `--variables` and stdin piping

### leaderboard

Rank team members by issue activity (local data)

- **`linear-pp-cli leaderboard`** - Team leaderboard
- Flags: `--weeks`, `--team`, `--user`, `--sort` (score/closed/created/assigned)

### search

Cross-entity full-text search (local data)

- **`linear-pp-cli search "query"`** - Search across issues, docs, projects
- Flags: `--scope` (all/issues/documents/projects/labels)

### similar

Find potentially duplicate issues using FTS5 (local data)

- **`linear-pp-cli similar "text"`** - Find similar issues before creating

### sprint

Sprint analytics (local data)

- **`linear-pp-cli sprint status`** - Current sprint overview with progress bar
- **`linear-pp-cli sprint burndown`** - ASCII burndown chart
- **`linear-pp-cli sprint velocity`** - Velocity across recent sprints
- **`linear-pp-cli sprint carry-over`** - Incomplete issues from last sprint

### stale

Find items with no updates in N days (local data)

- **`linear-pp-cli stale --days 14`** - Forgotten backlog items

### triage

Triage workflow

- **`linear-pp-cli triage list`** - List unassigned issues
- **`linear-pp-cli triage claim <id>`** - Assign to yourself

### workflow

Data sync and local store management

- **`linear-pp-cli workflow archive`** - Sync all resources to local SQLite
- **`linear-pp-cli workflow status`** - Show sync state

## Output Formats

```bash
# Human-readable table (default)
linear-pp-cli attachments list

# JSON for scripting and agents
linear-pp-cli attachments list --json

# Filter specific fields
linear-pp-cli attachments list --json --select id,name,status

# Plain tab-separated for piping
linear-pp-cli attachments list --plain

# Dry run (show request without sending)
linear-pp-cli attachments list --dry-run
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Retryable** - creates return "already exists" on retry, deletes return "already deleted"
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - `echo '{"key":"value"}' | linear-pp-cli <resource> create --stdin`
- **Cacheable** - GET responses cached for 5 minutes, bypass with `--no-cache`
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set
- **Progress events** - paginated commands emit NDJSON events to stderr in default mode

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
linear-pp-cli doctor
```

<!-- DOCTOR_OUTPUT -->

## Configuration

Config file: `~/.config/linear-pp-cli/config.toml`

Environment variables:
- `LINEAR_API_KEY`

## Troubleshooting

**Authentication errors (exit code 4)**
- Run `linear-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $LINEAR_API_KEY`

**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

**Rate limit errors (exit code 7)**
- The CLI auto-retries with exponential backoff
- If persistent, wait a few minutes and try again

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
