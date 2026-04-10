---
title: "fix: Resolve 12 CLI quality bugs found during dogfooding"
type: fix
status: active
date: 2026-04-10
---

# fix: Resolve 12 CLI quality bugs found during dogfooding

## Overview

The slack-pp-cli has 12 bugs discovered during real-world usage, ranging from fatal crashes to misleading output. This plan fixes all of them in the printing-press-library repo.

## Problem Frame

During hands-on testing of every command with a real Slack workspace, we found 2 crash bugs (panics/deadlocks), several data display issues (nil channel names, false credential validation), missing functionality (sync doesn't pull DMs/private channels, export only hits API), and input validation gaps (FTS5 query injection, negative limits).

## Requirements Trace

- R1. No command should panic or deadlock under any user input
- R2. Channel names should display as human-readable names, not nil or raw IDs
- R3. `doctor` should actually validate credentials against the API
- R4. `sync` should pull DMs, group DMs, and private channels
- R5. `search` should handle all user input without SQL errors
- R6. `export` should work from local data, not just live API
- R7. Invalid flag values (negative limits, bad types) should produce clear errors
- R8. Health grading should reflect realistic workspace activity levels

## Scope Boundaries

- Fixes target the slack CLI in the library repo only, not the printing-press generator templates
- No new features - only fixing broken existing behavior
- No changes to the MCP server

## Context & Research

### Relevant Code and Patterns

- `internal/store/store.go:236-248`: `Get()` does a `QueryRow` that conflicts with open cursors when `MaxOpenConns=1`
- `internal/cli/search.go:220`: Slice bounds crash on negative limit
- `internal/store/store.go:275-285`: FTS5 MATCH receives raw user input without escaping
- `internal/cli/funny.go:51,75`: Reads `msg["channel"]` from message JSON, but channel is stored in the `channel_id` DB column, not in the message JSON body
- `internal/cli/threads_stale.go:52`: Same channel resolution pattern as funny
- `internal/cli/response_times.go:50-67`: Iterates `rows.Next()` while calling `resourceName()` which opens a nested query - deadlocks because `MaxOpenConns=1`
- `internal/cli/doctor.go:103`: GETs the base URL instead of calling `auth.test`
- `internal/cli/sync.go:490`: `conversations.list` called without `types=` param
- `internal/cli/export.go:37,65`: Always creates an API client, never reads from local store
- `internal/cli/health.go:198-207`: Grading thresholds assume very high-traffic channels
- `internal/cli/analytics.go:106`: `obj[field]` on nonexistent field returns nil, printed as `<nil>`

### Institutional Learnings

- FTS5 query sanitization is a known gap in printing-press templates - no `sanitizeFTSQuery` exists anywhere. Standard fix: wrap each token in double quotes to neutralize FTS5 operators.
- Belt-and-suspenders pattern from filepath traversal solution: sanitize input AND handle errors gracefully.
- SQLite deadlock from nested queries with `MaxOpenConns=1` is a classic Go/SQLite pitfall. Fix: collect results into a slice before running dependent queries.

## Key Technical Decisions

- **FTS5 escaping via double-quoting**: Wrap each whitespace-delimited token in double quotes before passing to MATCH. This neutralizes `AND`, `OR`, `NOT`, `NEAR`, `*`, `(`, `)` without losing phrase matching. Chosen over stripping special chars because it preserves search intent.
- **response-times deadlock fix via result collection**: Collect all rows into a slice first, close the cursor, then call `resourceName()`. Chosen over increasing MaxOpenConns because it's the correct Go pattern for SQLite.
- **Channel name fix via channel_id column**: The message JSON body doesn't contain a `channel` field. The `channel_id` is stored in the messages table column. Fix the funny/threads-stale commands to extract channel_id from the DB row rather than from the JSON body.
- **doctor credential validation via auth.test**: Slack's `auth.test` endpoint is the canonical way to validate a token. Replace the base URL GET with a POST to `auth.test`.
- **sync types parameter**: Pass `types=public_channel,private_channel,mpim,im` to `conversations.list` so all conversation types are synced.
- **Export from local data**: Add `--data-source` flag to export, defaulting to local. Read from the store when local, hit API when live.
- **Health grading recalibration**: Lower thresholds to match typical small/medium workspace activity. A channel with 10+ msgs/day and 5+ posters should be an A, not a C.

## Implementation Units

- [ ] **Unit 1: Fix FTS5 query escaping in store**

**Goal:** Prevent raw user input from crashing FTS5 MATCH queries

**Requirements:** R1, R5

**Dependencies:** None

**Files:**
- Modify: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Approach:**
- Add a `sanitizeFTSQuery(query string) string` function that splits on whitespace, wraps each token in double quotes (escaping any embedded double quotes by doubling them), and rejoins
- Empty queries should return `""` which the caller handles by returning empty results
- Apply sanitization in `Search()`, `SearchUsergroups()`, and `SearchFiles()` before passing to MATCH

**Patterns to follow:**
- Existing `Search()` method at line 275

**Test scenarios:**
- Happy path: search "hello world" returns matching results
- Edge case: empty string query returns empty results without error
- Edge case: query with FTS5 operators "OR 1=1" is treated as literal text search
- Edge case: query with special chars "a]b[c" is treated as literal text search
- Edge case: query with semicolons "DROP TABLE;" is treated as literal text search
- Edge case: query with embedded quotes 'say "hello"' properly escapes the quotes

**Verification:**
- `slack-pp-cli search "" --data-source local` returns "No results" instead of SQL error
- `slack-pp-cli search "OR 1=1" --data-source local` returns results or "No results" without error

- [ ] **Unit 2: Fix search panic on negative limit**

**Goal:** Prevent slice bounds panic when limit is negative

**Requirements:** R1, R7

**Dependencies:** None

**Files:**
- Modify: `internal/cli/search.go`

**Approach:**
- In `outputSearchResults()` at line 220, guard the slice operation: if limit <= 0, skip the truncation
- Validate limit in the command's RunE before calling search

**Test scenarios:**
- Edge case: `--limit -1` does not panic, returns results or error
- Edge case: `--limit 0` returns empty results or all results

**Verification:**
- `slack-pp-cli search "hello" --data-source local --limit -1` exits cleanly

- [ ] **Unit 3: Fix response-times deadlock**

**Goal:** Eliminate goroutine deadlock from nested DB queries

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `internal/cli/response_times.go`

**Approach:**
- Collect all row data (channelID, avg, threads) into a struct slice during `rows.Next()` loop
- Close rows (or let defer handle it)
- Loop over the collected results to call `resourceName()` after the cursor is closed
- The current code at lines 60-68 already builds an `out` slice but calls `resourceName` inside the scan loop - move the `resourceName` call to a second pass

**Patterns to follow:**
- The threads_stale.go pattern where items are collected first, then processed

**Test scenarios:**
- Happy path: `response-times` returns channel response time data without deadlock
- Edge case: no thread data returns a helpful error message

**Verification:**
- `slack-pp-cli response-times` completes without panic or deadlock

- [ ] **Unit 4: Fix channel name resolution in funny and threads-stale**

**Goal:** Display channel names instead of `<nil>` in funny and threads-stale output

**Requirements:** R2

**Dependencies:** None

**Files:**
- Modify: `internal/cli/funny.go`
- Modify: `internal/cli/threads_stale.go`

**Approach:**
- Root cause: message JSON from `m.data` doesn't contain a `channel` field. The channel_id is stored in the `channel_id` column of the messages table, not in the JSON body.
- For funny.go: `TopReactedMessages` returns `m.data` which lacks channel. Either modify `TopReactedMessages` to also return `m.channel_id`, or have `TopReactedMessages` inject `channel_id` into the returned JSON before returning.
- For threads-stale.go: same pattern - `StaleThreads` returns `m.data` without channel. Check that method too.
- Simpler fix: modify `TopReactedMessages` and `StaleThreads` in store.go to SELECT `json_set(m.data, '$.channel', m.channel_id)` instead of plain `m.data`. This injects channel_id into the JSON at query time.

**Files:**
- Modify: `internal/store/store.go` (TopReactedMessages and StaleThreads methods)
- Modify: `internal/cli/funny.go` (verify msg["channel"] works after store fix)
- Modify: `internal/cli/threads_stale.go` (verify msg["channel"] works after store fix)

**Patterns to follow:**
- SQLite's `json_set()` function to inject fields into JSON at query time

**Test scenarios:**
- Happy path: `funny` output shows `#channel-name` instead of `<nil>`
- Happy path: `threads-stale` output shows `#channel-name` instead of `<nil>`

**Verification:**
- `slack-pp-cli funny --period month` shows channel names in the Channel column
- `slack-pp-cli threads-stale` shows channel names in the Channel column

- [ ] **Unit 5: Fix doctor credential validation**

**Goal:** Make `doctor` actually verify credentials work against the Slack API

**Requirements:** R3

**Dependencies:** None

**Files:**
- Modify: `internal/cli/doctor.go`

**Approach:**
- Replace the GET to base URL (line 103) with a POST to `/auth.test`, which is Slack's canonical credential validation endpoint
- Parse the JSON response for `"ok": true` vs `"ok": false`
- On `"ok": false`, report the error field from the response (e.g., "invalid_auth", "token_revoked")
- Keep the existing reachability check (HEAD to base URL) for the API status line

**Test scenarios:**
- Happy path: valid token reports "Credentials: valid"
- Error path: invalid/garbage token reports "Credentials: invalid (invalid_auth)"
- Error path: empty token reports "Auth: not configured"

**Verification:**
- `SLACK_USER_TOKEN=bad-token slack-pp-cli doctor` shows credentials as invalid, not valid

- [ ] **Unit 6: Add DM/private channel support to sync**

**Goal:** Sync DMs, group DMs, and private channels alongside public channels

**Requirements:** R4

**Dependencies:** None

**Files:**
- Modify: `internal/cli/sync.go`

**Approach:**
- In `syncResourcePath()` at line 490, change the conversations path from `/conversations.list` to `/conversations.list?types=public_channel,private_channel,mpim,im`
- Alternatively, modify the sync caller to pass the types parameter as a query param
- The simpler approach is appending to the path since the sync infrastructure passes this directly to the API client

**Test scenarios:**
- Happy path: `sync` pulls DM and private channel conversations into the local store
- Happy path: after sync, `search` finds content from DMs and private channels

**Verification:**
- After `slack-pp-cli sync`, the resources table contains conversations with `is_im: true` and `is_private: true`

- [ ] **Unit 7: Add local data support to export**

**Goal:** Allow export to read from synced local data instead of requiring API access

**Requirements:** R6

**Dependencies:** None

**Files:**
- Modify: `internal/cli/export.go`

**Approach:**
- Add `--data-source` flag matching the pattern used by other commands
- When data-source is "local" or "auto" with local data available, read from `db.List(resource, limit)` instead of `c.Get(path, nil)`
- When data-source is "live", keep the existing API path
- Import the store package and use `openAnalyticsStore()` for local access

**Patterns to follow:**
- The search command's data-source routing pattern

**Test scenarios:**
- Happy path: `export messages --data-source local` exports locally synced messages
- Happy path: `export messages --data-source live` hits the API as before
- Error path: `export messages --data-source local` with no synced data returns a clear error

**Verification:**
- `slack-pp-cli export messages --data-source local --format jsonl` outputs JSONL from local store

- [ ] **Unit 8: Recalibrate health grading thresholds**

**Goal:** Make health grades reflect realistic workspace activity levels

**Requirements:** R8

**Dependencies:** None

**Files:**
- Modify: `internal/cli/health.go`

**Approach:**
- Current thresholds (A: 100+ msgs/day or 25+ users, B: 40+/12+, C: 10+/5+, D: >0) are calibrated for very large workspaces
- Recalibrate to: A: 10+ msgs/day AND 3+ unique posters, B: 5+ msgs/day AND 2+ posters, C: 1+ msgs/day, D: >0 but <1/day, F: no activity
- Change OR logic to AND for A/B grades - a channel needs both volume and participation to be healthy
- Keep single-poster channels capped at C regardless of volume (bot channels)

**Test scenarios:**
- Happy path: a channel with 12 msgs/day and 8 posters gets grade A
- Edge case: a channel with 100 msgs/day but 1 poster (bot channel) gets grade C
- Edge case: a channel with 0 messages gets grade F

**Verification:**
- `slack-pp-cli health` shows a distribution of grades, not all C/D

- [ ] **Unit 9: Validate analytics group-by field and limit edge cases**

**Goal:** Reject invalid group-by fields and handle edge case flag values

**Requirements:** R7

**Dependencies:** None

**Files:**
- Modify: `internal/cli/analytics.go`
- Modify: `internal/cli/funny.go`

**Approach:**
- In `runGroupBy()`, after extracting values, check if all keys are `<nil>` - if so, the field doesn't exist. Return an error: "field %q not found in %s records"
- In funny.go, validate that `--limit 0` and `--limit -1` are rejected with a clear error or clamped to 1

**Test scenarios:**
- Error path: `analytics --type messages --group-by nonexistent` returns "field not found" error
- Edge case: `funny --limit 0` returns error or clamps to minimum
- Edge case: `funny --limit -1` returns error or clamps to minimum

**Verification:**
- `slack-pp-cli analytics --type messages --group-by fake_field` shows a clear error

- [ ] **Unit 10: Fix conversations history limit validation**

**Goal:** Validate numeric flags before sending to API

**Requirements:** R7

**Dependencies:** None

**Files:**
- Modify: `internal/cli/promoted_conversations.go`

**Approach:**
- The `--limit` flag for `conversations history` is a string type. Validate it's a valid positive integer before making the API call
- Return a clear error like "invalid --limit: expected a positive integer" instead of passing garbage to the API

**Test scenarios:**
- Error path: `conversations history --channel C123 --limit abc` returns validation error
- Edge case: `conversations history --channel C123 --limit -1` returns validation error

**Verification:**
- `slack-pp-cli conversations history --channel C123 --limit abc` shows a local validation error, not an API error

- [ ] **Unit 11: Handle missing scopes gracefully**

**Goal:** Show user-friendly messages when Slack API returns missing_scope errors

**Requirements:** R7

**Dependencies:** None

**Files:**
- Modify: `internal/cli/helpers.go` or wherever `classifyAPIError` lives

**Approach:**
- When the API returns `{"ok": false, "error": "missing_scope", "needed": "emoji:read"}`, the CLI currently dumps the raw JSON
- Parse the response and show: "Missing Slack scope: emoji:read. Add it in your app's OAuth & Permissions settings and reinstall."
- Apply this to the common response handling path so all commands benefit

**Test scenarios:**
- Error path: command requiring missing scope shows human-readable error with the needed scope name

**Verification:**
- `slack-pp-cli emoji` (without emoji:read scope) shows "Missing Slack scope: emoji:read" instead of raw JSON

- [ ] **Unit 12: Fix export hitting API with wrong method name**

**Goal:** Fix export so it constructs proper Slack API paths

**Requirements:** R6

**Dependencies:** Unit 7

**Files:**
- Modify: `internal/cli/export.go`

**Approach:**
- Currently export constructs path as `"/" + resource` (e.g., `/messages`), but Slack API endpoints use dot notation (e.g., `/conversations.history`)
- For the live API path, reuse the `syncResourcePath()` mapping so export uses the same endpoints as sync
- This ensures `export conversations` hits `/conversations.list`, not `/conversations`

**Test scenarios:**
- Happy path: `export conversations --data-source live` hits the correct Slack API endpoint
- Error path: `export nonexistent --data-source live` returns a clear "unknown resource" error

**Verification:**
- `slack-pp-cli export conversations` returns conversation data instead of `unknown_method`

## System-Wide Impact

- **Interaction graph:** The FTS sanitization fix in store.go affects search, funny (indirectly via TopReactedMessages), and any future command that uses FTS. The channel name fix via json_set affects funny, threads-stale, and response-times.
- **Error propagation:** Missing scope errors are currently swallowed as raw JSON - Unit 11 makes them user-friendly across all commands.
- **State lifecycle risks:** The sync types change (Unit 6) will pull more data on next sync. Existing local data is unaffected - new data is additive.
- **Unchanged invariants:** The MCP server is not modified. All API write commands (messages post, reactions add, etc.) are unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| json_set may not be available in all SQLite builds | It's part of the JSON1 extension which is compiled in by default since SQLite 3.38.0 (2022). Go's mattn/go-sqlite3 includes it. |
| Sync pulling DMs may hit rate limits on large workspaces | Slack's conversations.list handles all types in one paginated call, so no additional API calls beyond what sync already makes. |
| Health grade recalibration may surprise users who rely on current grades | This is a cosmetic change with no downstream effects. Document the new thresholds in the help text. |

## Sources & References

- Related bugs: found during hands-on dogfooding session
- Slack API docs: auth.test endpoint for credential validation
- SQLite FTS5 docs: tokenize and MATCH query syntax
- Go database/sql docs: cursor behavior with MaxOpenConns=1
