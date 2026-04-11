---
title: "feat: agent-hints on stderr when archive-is-pp-cli is called non-interactively"
type: feat
status: completed
date: 2026-04-11
---

# Agent Hints on Stderr

**Target repo:** archive-is-pp-cli at `~/printing-press/library/archive-is/`. All file paths in this plan are relative to that directory.

## Overview

When `archive-is-pp-cli` runs inside an agent loop (Claude Code calling it via a Bash tool), the interactive terminal UX from the previous plan — the `Open in browser? [Y/n/q]` prompt, the `[o]/[t]/[r]/[q]` menu — is completely invisible. The calling agent sees only bare stdout:

```
https://archive.md/20260411080659/https://www.nytimes.com/...
  captured: 2026-04-11 08:06:59
  mirror:   https://archive.ph
  backend:  archive-is
```

Nothing tells the agent that `open <url>` would show the article to the user, or that `archive-is-pp-cli tldr <orig>` would summarize it, or that `archive-is-pp-cli get <orig>` would return the full text. The agent has to guess.

This plan adds machine-readable next-step hints to stderr when the CLI runs in non-interactive mode. The interactive human flow is unchanged — the prompt still fires in a real terminal. The hints are purely additive for agent callers.

## Problem Frame

Matt runs every CLI through Claude Code. Claude Code invokes CLIs via its Bash tool, which pipes stdout back for the model to read — never attaches a real TTY. The `isInteractive()` guard on the prompt correctly returns false, so no prompt appears. Result: the CLI's "dogfood user" is the model, not Matt, and the model currently receives zero guidance about what to do with the result.

Symptom observed tonight: Matt asked the CLI to read an article. The model returned the archive URL and stopped. Matt had to explicitly say "can you open in my browser" for the model to run `open <url>` — something a terminal user would have gotten via the prompt, and something the model would have known to do if the CLI's output had said so.

The fix is small: when the CLI detects it's not in an interactive terminal, emit a short `NEXT:` block on stderr listing the actions the caller can take, with the exact commands. Agents reading stderr get guidance. Humans running with `| cat` or `--quiet` can opt out. Scripts and `--json` consumers get a parallel `next_actions` field in the JSON payload.

## Requirements Trace

- **R1.** When `read`, `save`, or `request --wait` completes successfully in non-interactive mode, the CLI emits a short list of suggested next-step commands on stderr.
- **R2.** The hints include, at minimum: open-in-browser (platform-appropriate), summarize (`tldr`), extract-text (`get`), and list-history (`history`). Each hint is a full runnable command line.
- **R3.** When `--json` is set, the same actions are exposed as a `next_actions` array in the JSON payload instead of on stderr. The caller can parse one or the other, never both.
- **R4.** Zero impact on the existing interactive menu flow. When the CLI detects a real TTY, the prompt still fires as today — hints are skipped in interactive mode.
- **R5.** `--quiet` suppresses the hints entirely (both stderr and JSON). The quiet mode contract is "minimal output" and that wins.
- **R6.** `--agent` flag also emits hints, because `--agent` is the explicit "an agent is driving me" signal. Matt's Claude Code workflow is the canonical agent path.
- **R7.** Hints are generated for `read`, `save`, and `request --wait`. Not for `get` (its stdout is already the article text), `history` (multi-result, hints would be ambiguous), `bulk` (batch, hints would fire N times), `tldr` (its stdout is already the summary), or `request` without `--wait` (terminal state unknown).

## Scope Boundaries

- **In scope:** Additive changes to `read`, `save`, and `request --wait` result paths in `internal/cli/read.go`. New helper file `internal/cli/agent_hints.go`.
- **Out of scope:** Generalizing the pattern to the Printing Press generator so every future printed CLI gets agent hints for free. That's a follow-up plan against the `cli-printing-press` repo and belongs alongside the URL-shortcut generalization from the previous session's notes.
- **Out of scope:** New subcommands. No `archive-is-pp-cli explain` or similar meta-command. The hints are inline with the existing commands.
- **Out of scope:** Teaching the MCP server to expose these hints. The MCP already ships rich tool descriptions — agents connecting via MCP get structured context directly and don't need stderr parsing. Wiring the MCP into Claude Code's settings is option 3 from the conversation and tracked separately.
- **Out of scope:** Rewriting `--quiet` or `--json` semantics. Both are preserved as-is.

## Context & Research

### Relevant Code and Patterns

- `internal/cli/interactive.go` — `isInteractive(flags *rootFlags) bool`. Already the single source of truth for "should we prompt the user". The hints use the inverse of this check.
- `internal/cli/read.go` — `renderMemento()`, `maybePromptOpen()`, `newReadCmd()`, `newSaveCmd()`, `newRequestCmd()`, `renderRequestResult()`. These are the result-rendering paths that need a new call to the hint helper.
- `internal/cli/read.go` — `memento` struct has `OriginalURL` and `MementoURL`; hints need both.
- `internal/cli/root.go` — `rootFlags` struct holds `asJSON`, `quiet`, `agent`, `noPrompt`. The hint check reads these.
- `cmd/archive-is-pp-cli/main.go` — `rewriteURLShortcut()` + `knownCommands` map. No change needed unless we add a new top-level subcommand, which we are not.
- Previous session's `.notes/improvements.md` — item #1 mentions this pattern under "Conversational post-action UX" but scoped it to interactive terminals only. This plan is the non-interactive complement.

### Institutional Learnings

- The menu/prompt UX shipped yesterday (`maybePromptOpen` + `promptMenu`) is the right model for humans. It deliberately gates on `isInteractive()`. That gate is now doing double duty: "suppress menu" AND "emit hints instead". Keep the gate single-purpose-for-the-user but let the hint path branch off it.
- PostHog's agent-first rules map cleanly here. Rule 3 ("front-load universal context") says the agent needs to know what it can do with a result. Rule 4 ("skills are human knowledge") says we encode our opinion about the right next step — for archive-is-pp-cli, that's "open in browser" for the 80% case, with tldr / get / history as escape valves.

### External References

- PostHog blog post on agent-first product engineering (2026-04-09, @posthog on X) — informs the design principle that the CLI's primary user is the agent, not the human.

## Key Technical Decisions

- **Use the existing `isInteractive()` check, inverted.** When `isInteractive()` returns false AND `flags.quiet` is false, emit hints. No new detection logic, no new env vars, no new flag. The same rules that suppress the menu are the rules that enable hints.
- **Hints go to stderr for non-JSON callers and into `next_actions` for JSON callers.** Never both. This keeps stdout clean for piping and keeps JSON consumers from having to parse two sources.
- **One central helper, not inline.** All three commands (`read`, `save`, `request --wait`) call the same `agentHintsFor(memento, cmd.ErrOrStderr())` helper so the format and action set stay consistent. Also: adding a new hint later means editing one file.
- **Hint format is a `NEXT:` prefix + tab-separated label and command.** Greppable, compact, agent-friendly. Example:
  ```
  NEXT: open in browser          open "https://archive.md/..."
  NEXT: summarize with LLM       archive-is-pp-cli tldr "https://www.nytimes.com/..."
  NEXT: get article text         archive-is-pp-cli get "https://www.nytimes.com/..."
  NEXT: list historical snapshots  archive-is-pp-cli history "https://www.nytimes.com/..."
  ```
  No decoration, no color, no fancy Unicode. LLMs parse this cleanly and humans scanning stderr see exactly what would run.
- **Platform-specific open command.** The hint uses `open` on macOS, `xdg-open` on Linux, `cmd /c start ""` on Windows. The existing `openInBrowser()` helper in `internal/cli/interactive.go` already does the detection internally, but hints need to print the command the agent should literally run, so the detection is duplicated in the hint generator. Acceptable — one extra `runtime.GOOS` switch.
- **JSON action shape.** Each entry has `action` (machine-readable tag), `command` (full command string), and `description` (short human-readable label). This mirrors the stderr `NEXT:` format while being structured.
- **Hints are added after existing output, not before.** The result URL + metadata goes first (unchanged) so any existing parser that looked at the first line of stdout still works. Hints are purely additive stderr noise for anyone not expecting them.
- **No hints for bulk, history, get, tldr, `request` without --wait.** These commands either produce the payload directly on stdout (get, tldr), return many results (bulk, history), or have no terminal state yet (request without wait). Hints would be either redundant or ambiguous.

## Open Questions

### Resolved During Planning

- **Should the hints fire when `--no-prompt` is set?** Yes. `--no-prompt` suppresses the interactive menu, which it must because the agent hasn't asked for a prompt. But it does not suppress hints — the agent still wants to know what actions exist. `--quiet` is the flag that suppresses hints.
- **Should the hints fire in `--agent` mode?** Yes. `--agent` is the explicit "I am an agent" flag. Hints are the whole point.
- **Should hints suggest commands or also show expected output shapes?** Commands only, for v1. Expected-output hints would bloat the stderr block without adding much value; agents can run the commands and see the output shape empirically.
- **Should we cap the number of hints?** Yes — max 4 per command. More than 4 is noise. The four are: open, tldr, get, history.

### Deferred to Implementation

- **Exact wording of each hint label.** The strings above ("open in browser", "summarize with LLM", "get article text", "list historical snapshots") are starting points. Tune during implementation for clarity and consistency.
- **Whether to include a trailing blank line before the NEXT: block.** Aesthetic choice; decide when looking at real output.
- **Whether the `description` field in JSON should match the stderr label verbatim or be more verbose.** Start by matching; diverge only if a specific JSON consumer needs more context.

## Implementation Units

- [ ] **Unit 1: Add the `agentHintsFor` helper**

**Goal:** Single central function that generates the hint list for a given memento. Used by all downstream callers.

**Requirements:** R1, R2, R5, R6

**Dependencies:** None.

**Files:**
- Create: `internal/cli/agent_hints.go`
- Test: `internal/cli/agent_hints_test.go`

**Approach:**
- Define a `hintAction` struct with `Tag`, `Command`, `Description` fields.
- `agentHintsFor(memento *memento) []hintAction` returns the standard four actions: open, tldr, get, history. The `open` command is built from `runtime.GOOS`. The three CLI commands use the memento's `OriginalURL` as the argument.
- `writeAgentHints(w io.Writer, actions []hintAction)` renders the `NEXT:` prefix format to the given writer. Used when emitting to stderr.
- Helper is pure — no I/O except the writer. Easy to unit test.

**Patterns to follow:**
- `copyToClipboard()` in `internal/cli/read.go` — cross-platform `runtime.GOOS` switch.
- `openInBrowser()` in `internal/cli/interactive.go` — the existing browser-launch command mapping, which the hint's `open` line mirrors.

**Test scenarios:**
- Happy path: `agentHintsFor` with a memento whose OriginalURL is `https://example.com/article` returns four actions with correct commands.
- Happy path: `writeAgentHints` writes exactly four `NEXT:` lines to the buffer with the expected prefixes.
- Edge case: memento with empty OriginalURL falls back to MementoURL for the tldr/get/history hints (they still work against an archive URL, just less ideal).
- Edge case: memento with empty MementoURL returns no hints (the `open` action has nothing to open).
- Platform: on macOS, the `open` command is `open "<url>"`. On Linux, `xdg-open "<url>"`. Test via `runtime.GOOS` override or a helper variable.

**Verification:**
- Unit tests pass.
- `go vet` clean.

---

- [ ] **Unit 2: Emit hints from `read`, `save`, and `request --wait` result paths**

**Goal:** Wire the helper into the three commands that produce a single terminal-state memento result. Hints appear on stderr when non-interactive + not --quiet; interactive flow is untouched.

**Requirements:** R1, R4, R5, R6, R7

**Dependencies:** Unit 1.

**Files:**
- Modify: `internal/cli/read.go` — `newReadCmd()`, `newSaveCmd()`, `renderRequestResult()` (for the ready/existing cases).
- Test: `internal/cli/read.go` callers exercised indirectly via new integration tests in `internal/cli/agent_hints_integration_test.go`.

**Approach:**
- Create a small wrapper `maybeEmitHints(cmd, flags, memento)` that checks `!isInteractive(flags) && !flags.quiet && !flags.asJSON` and, if true, writes the hints to `cmd.ErrOrStderr()`. If any of the three guards is false, it's a no-op.
- Call `maybeEmitHints()` immediately after `renderMemento()` in the three result paths, before (or alongside) the `maybePromptOpen()` call. The two are mutually exclusive via the `isInteractive()` check: interactive → menu, non-interactive → hints.
- In `renderRequestResult()`, call `maybeEmitHints()` for the `ready` and `existing` cases. Not for `pending`, `failed`, or `timeout` — those are incomplete states without a URL to act on.
- Do NOT call from `newGetCmd()`, `newHistoryCmd()`, `newBulkCmd()`, `newTldrCmd()`, or the bare `request` (no --wait) path.

**Patterns to follow:**
- `maybePromptOpen()` in `internal/cli/read.go` — sibling helper, same shape, same gating, different purpose.
- `renderMemento()` — existing render-then-branch pattern.

**Test scenarios:**
- Integration: run `read` with a mocked non-TTY stdin/stdout → stderr contains `NEXT:` lines, stdout has the URL + metadata, no prompt attempted.
- Integration: run `read` with `--quiet` → stdout has the URL only, stderr has NO hints, no prompt.
- Integration: run `read` with `--json` → stdout has JSON, stderr has NO `NEXT:` lines (hints move into JSON per Unit 3).
- Integration: run `save` non-interactive → same stderr hints as `read`.
- Integration: run `read` with `--agent` → stderr has hints (agent mode forces non-interactive but does not suppress hints).
- Integration: run `read` in a real TTY (terminal mode helper or mock) → no hints; prompt fires instead. Assert stderr is empty of NEXT: lines.
- Edge case: `request check <url>` with a `ready` state from the state file → hints appear alongside the READY line.
- Edge case: `request <url>` without --wait returning PENDING → no hints.
- Error path: `read` fails to find a snapshot and submit fails → apiErr returned, no hints emitted (nothing to act on).

**Verification:**
- Running `archive-is-pp-cli read https://www.nytimes.com/` through a non-TTY (e.g., `| cat`) emits four `NEXT:` lines on stderr plus the existing URL output on stdout.
- Running the same command with `--quiet` produces no `NEXT:` lines.
- Running in a real terminal shows the existing prompt and no `NEXT:` lines.

---

- [ ] **Unit 3: Include `next_actions` in `--json` output**

**Goal:** When the caller asks for JSON, hints travel inside the payload as a structured field instead of on stderr. Parsers get everything from one source.

**Requirements:** R3, R5

**Dependencies:** Unit 1.

**Files:**
- Modify: `internal/cli/read.go` — `renderMemento()` for `read`/`save`, `renderRequestResult()` for `request`/`request check`.
- Test: `internal/cli/agent_hints_test.go` (extended from Unit 1).

**Approach:**
- When `flags.asJSON` is true and hints would otherwise fire (non-interactive, not --quiet), include the hint list as a `next_actions` array in the JSON object.
- Each array entry: `{"action": "open_browser", "command": "open \"https://...\"", "description": "open in browser"}`.
- The JSON path SUPPRESSES the stderr emission from Unit 2 — callers get the same information once, not twice.
- `--quiet --json` still suppresses hints (both locations). Quiet wins over everything.

**Patterns to follow:**
- Existing `flags.asJSON` JSON rendering in `renderMemento()` — just add one field to the map.
- Existing `renderRequestResult()` JSON branch — same pattern.

**Test scenarios:**
- Happy path: `read <url> --json` returns JSON with `next_actions` array containing four entries.
- Happy path: `save <url> --json` same.
- Happy path: `request <url> --wait --json` same, for the ready case.
- Edge case: `--json --quiet` → JSON does NOT include `next_actions` (quiet rules).
- Edge case: interactive TTY + `--json` → `--json` forces non-interactive semantics, so JSON includes `next_actions`. (Consistent with Unit 2's rule: JSON always implies non-interactive.)
- JSON parseability: assert that the output is valid JSON and that `next_actions[0].command` is a non-empty string.

**Verification:**
- `archive-is-pp-cli read <url> --json | jq '.next_actions | length'` returns 4.
- `archive-is-pp-cli read <url> --json --quiet | jq '.next_actions // "none"'` returns `"none"`.

## System-Wide Impact

- **Interaction graph:** Unit 2 touches `newReadCmd()`, `newSaveCmd()`, and `renderRequestResult()`. Unit 3 touches `renderMemento()` and `renderRequestResult()`. Everything else is unchanged. Dependencies flow: Unit 1 → Unit 2 + Unit 3.
- **Error propagation:** Hints fire only on success paths. Error paths (rate-limited submit, CAPTCHA fallback failure, network error) return their existing typed errors. No hint emitted when there's no URL to act on.
- **State lifecycle risks:** None. Hints are stateless output — nothing to persist, nothing to GC.
- **API surface parity:** The CLI's human flags (`--quiet`, `--json`, `--agent`, `--no-prompt`) keep their existing semantics. One new implicit behavior: non-interactive + not-quiet now emits stderr hints. This is additive for every existing flag combination except `--json` where hints move inside the payload.
- **Integration coverage:** The three-way matrix (interactive TTY, non-interactive, non-interactive + --json) needs explicit integration tests. Unit tests alone won't catch the "hints show up in the wrong channel" bug class. At least one integration test per channel.
- **Unchanged invariants:**
  - Interactive terminal flow is identical to before: prompt fires, menu appears, default is open.
  - `--quiet` still produces minimal output.
  - `--agent` still sets `--json`, `--compact`, `--no-input`, `--yes`, `--no-prompt`. The new addition is that agent mode ALSO gets hints inside the JSON payload.
  - Typed exit codes unchanged.
  - The MCP server (`archive-is-pp-mcp`) is not touched.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Stderr noise breaks a script that parsed stderr for specific error strings. | Hints are prefixed with `NEXT:` — easy to grep out. Document the convention in the README and in the `--help` footer. |
| LLM parses the stderr hints literally and executes them verbatim without checking that the paths exist. | Not a meaningful risk — the commands are well-formed and resolve correctly on the caller's machine. If `open` or `xdg-open` is absent, the caller gets a clear error from the shell, same as today. |
| JSON consumers ignore `next_actions` and continue using stderr hints. | Both channels are valid; neither breaks. The JSON is a nicer interface but parsing stderr `NEXT:` lines is also fine. |
| Someone adds a new hint action and forgets to update tests. | Unit 1's tests pin the expected action count. Breaking the contract fails CI. |
| Future Printing Press machine regeneration wipes the `agent_hints.go` file. | Not in scope here, but flagged in Scope Boundaries: the generalization plan should add the helper as a template so regenerated CLIs preserve the behavior. |

## Documentation / Operational Notes

- Update `README.md` with a new "Agent Integration" section under Usage. Example: running from a shell agent, what to expect on stderr, how to parse `next_actions` from JSON.
- Update `archive-is-pp-cli --help` footer (or the `read`/`save` long descriptions) to mention the `NEXT:` convention.
- No rollout concerns. Ship behind the normal `go install` path. Users on the prior version see no change; users on this version get the new stderr lines.

## Sources & References

- **Origin conversation:** session on 2026-04-11 after the archive-is-pp-cli polish work. Matt asked "should it have given YOU better instructions? I only run CLIs through Claude Code." This plan is option 2 from that exchange.
- Previous session plan: `docs/plans/2026-04-10-001-feat-archive-is-polish-improvements-plan.md` — established `isInteractive()`, `maybePromptOpen()`, `promptMenu()`.
- Previous session notes: `.notes/improvements.md` — item #1 ("Conversational post-action UX") captured the interactive side; this plan is the non-interactive complement.
- PostHog blog post: "The golden rules of agent-first product engineering" (2026-04-09) — Rule 3 (front-load context) and Rule 4 (skills are human knowledge) map directly to this feature.
- Related published work: [printing-press-library PR #37](https://github.com/mvanhorn/printing-press-library/pull/37) — the initial archive-is publish.
