---
title: "fix: swap Wayback availability API for CDX, fix submit timeout and visibility"
type: fix
status: completed
date: 2026-04-11
completed: 2026-04-11
origin: docs/tests/2026-04-11-archive-capture-report.md
---

# Fix Wayback CDX + Submit Reliability

**Target repo:** archive-is-pp-cli at `~/printing-press/library/archive-is/`. All file paths are relative to that directory. This is a standalone Go module (not a git repo).

## Overview

Today's capture-matrix test (`docs/tests/2026-04-11-archive-capture-report.md`) surfaced two bugs in the CLI's reliability story. Both are small, well-scoped, and high-impact:

1. **`waybackLookup` uses the unreliable `availability` API.** The Phase A test ran `tldr` against 20 URLs. 15 failed with "no wayback snapshot available" — but Wayback's CDX API confirms those URLs DO have captures. The availability endpoint is returning empty for indexed URLs. This single bug is responsible for 75% of tldr failures in the test matrix.

2. **`submitCapture`'s per-request timeout doesn't fire, and there's no progress visibility.** During Phase B retry of the WSJ article after cooldown cleared, the submit hung for 9+ minutes with CPU 0.0 before I killed it. Go's `http.Client.Timeout` isn't enforcing the per-request limit when archive.is holds a TCP connection open without responding. The user sits in silence with zero feedback for the entire duration. The fix isn't to make the CLI give up earlier — **archive.is might take longer sometimes, and might get faster in the future, and the CLI shouldn't pre-decide what "too long" means for the user**. The fix is: make per-request timeouts actually work, make the submit workflow visible, let the user Ctrl-C cleanly, and make the overall budget configurable with a sane default.

This plan fixes both. Two implementation units, both small, land in either order. Expected impact on the same test matrix:
- `tldr` pass rate: **5/20 → ~17/20** (CDX fix alone)
- Submit UX: live progress on stderr, per-request timeouts that actually work, user can Ctrl-C cleanly, user can override the budget with a flag if they want to wait longer

## Problem Frame

The archive-is CLI has a clear hero workflow (`read` → find or create archive) and a content-extraction workflow (`tldr` / `get` → fetch body and summarize). The hero workflow is genuinely production-quality: today's test showed 19/20 URLs returned working archive.is timegate snapshots.

The content-extraction workflow is broken, but not where we thought:
- **Not because archive.is CAPTCHA'd body fetches** — the CLI already detects that and falls back to Wayback. Working as designed.
- **Not because Wayback lacks snapshots** — we verified via CDX API that BBC News (since 1999), Simon Willison (since 2005), Wikipedia (since 2004), and many others ARE archived.
- **Because the availability API returns empty for URLs that ARE in Wayback.** Inconsistently. Sometimes returns the snapshot, sometimes doesn't, for the same URL within seconds. It's an upstream bug in Internet Archive's availability endpoint.

The submit path bug is separate: when archive.is's scraper is slow-responding (which it often is for DataDome-protected sites like WSJ), my CLI sits on an open TCP connection until the process is killed. No user feedback. No progress. No give-up. The 180-second http.Client.Timeout doesn't fire because Go's client timeout has known edge cases when the server is slow-reading but not closing the connection.

Matt's principle for this CLI is "try loudly, fail clearly, offer paths forward." The current submit behavior is none of those — it's silent, indefinite, and leaves the user with nothing to do.

## Requirements Trace

- **R1.** When `tldr`, `get`, or `read` needs to query Wayback for a snapshot, the CLI uses the CDX API (not the availability API) as its source of truth. CDX is Wayback's canonical search endpoint and is reliable.
- **R2.** When the CDX query returns results, the CLI constructs the memento URL from the most recent snapshot's timestamp and original URL, and returns it the same way the existing `waybackLookup` does today.
- **R3.** When CDX returns no results, the CLI retries with a canonicalized form of the URL (strip trailing slash, try without `www.` prefix) before giving up. URL canonicalization is cheap and covers most edge cases.
- **R4.** `submitCapture` uses `context.WithTimeout` as the authoritative per-request cancellation mechanism, not `http.Client.Timeout`. Per-request timeouts fire reliably even when archive.is is slow-responding.
- **R5.** The overall submit budget is **configurable**, not hard-coded. A new `--submit-timeout` flag (default: 10 minutes) lets the user override it. Rationale: archive.is's typical capture time varies and may change over time — the CLI shouldn't pre-decide what "too long" means. Users who want to wait 30 minutes should be able to; users on a CI box that needs a quick exit should be able to set 2 minutes.
- **R6.** During a submit that takes longer than ~10 seconds, the CLI prints live progress to stderr so the user sees what's happening. At minimum: elapsed time ticker every 10 seconds, plus the predictive archive.ph snapshot URL immediately at the start so the user can refresh it in their browser manually.
- **R7.** Ctrl-C during a submit cancels cleanly — the context cancellation propagates, the progress ticker stops, the function returns in under a second with exit code 130. The user is always in control.
- **R8.** `waybackLookup`'s public signature is preserved. `submitCapture` gains a context parameter (internal, no public surface area exposed beyond the new `--submit-timeout` flag). The rate-limit cooldown state file, the submit error formatter, and the agent-hints output are unchanged.
- **R9.** All existing tests continue to pass.

## Scope Boundaries

- **In scope:** Changes to `internal/cli/read.go` (waybackLookup + submitCapture + tryMirrorWithBackoff) and possibly `internal/cli/http_client.go` (context helpers). Tests for each change.
- **Out of scope:** Changes to `tldr.go`, `menu.go`, `interactive.go`, or the MCP server. The tldr pipeline calls `waybackLookup` — fixing that helper is enough; no changes needed in the callers.
- **Out of scope:** Caching CDX responses. CDX is fast; caching is a premature optimization.
- **Out of scope:** Parallelizing Wayback queries across multiple URL forms. Sequential with early-exit is simpler and fast enough.
- **Out of scope:** Predictive URL for `request` command (the async-submit path). That has its own workflow (state file, polling) and is covered by the notes-item #5 follow-up, not this plan.
- **Out of scope:** Exploring or fixing Internet Archive's availability API. We just route around it.
- **Out of scope:** Any change to `read` command's timegate-first behavior. `read` works correctly today — it's the body-fetch and submit paths that this plan fixes.

## Context & Research

### Relevant Code and Patterns

- `internal/cli/read.go` — `waybackLookup(origURL, timeout)` at ~line 312. Single function, ~45 lines. The whole fix for Unit 1 lives here.
- `internal/cli/read.go` — `submitCapture()`, `tryMirrorWithBackoff()`, `tryMirrorOnce()`. The whole fix for Unit 2 lives in these three.
- `internal/cli/http_client.go` — `newArchiveHTTPClient()`, `backoffSchedule()`. Unit 2 may add a context-aware variant here.
- `internal/cli/rate_limit.go` — `CooldownError`, `classifySubmitError()`, cooldown state file. Not modified by this plan; referenced only to make sure the new budget-exhausted path returns a compatible error.
- `internal/cli/submit_error.go` — `SubmitFailureError`. Unit 2 adds a new failure mode ("budget exhausted") that this type should be able to format cleanly.
- `internal/cli/read.go` — `newReadCmd()`, `newSaveCmd()`, `newRequestCmd()`. These call `submitCapture`/`waybackLookup`. No code changes here, but verify none of them passes contexts that would interfere.
- `internal/cli/tldr.go` — `runTldr()`. Calls `waybackLookup` in the body-fetch fallback path. No changes; will benefit from Unit 1 automatically.

### Institutional Learnings

- **Unit 6's URL canonicalization lesson from yesterday's polish plan:** yesterday we fixed the "Wayback expects unencoded URLs" bug by passing the URL directly (without `url.QueryEscape`). That fix is still in place. This plan extends it with additional canonicalization (trim trailing slash, strip www) as a fallback path.
- **Unit 8's silent redirect detection from yesterday:** uses a HEAD request with a separate `http.Client`. Unit 2 of this plan should follow the same pattern — dedicated client for each concern, context-based cancellation for reliability.
- **Matt's principle** (documented in `.notes/improvements.md` section #8): "try loudly, fail clearly, offer paths forward." The existing silent hang on submit violates this directly. Unit 2's progress output and budget cap restore the principle.

### External References

- [Wayback CDX Server API docs](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server) — the canonical documentation for the CDX endpoint. Supports `output=json`, `limit=-N` for most recent N, `filter=statuscode:200`, `fl=field1,field2` for output field selection.
- [Go `context` package — context.WithTimeout](https://pkg.go.dev/context#WithTimeout) — standard pattern for request cancellation with a hard deadline. Used throughout the Go ecosystem.
- [Dave Cheney — "Why I don't rely on http.Client.Timeout"](https://dave.cheney.net/2014/09/14/go-network-transport-timeouts) — articulates the known edge cases with `http.Client.Timeout` vs context-based cancellation. Matches exactly the bug we observed today.

## Key Technical Decisions

- **Use CDX API, not a combined availability+CDX flow.** Yesterday I considered "try availability first, fall back to CDX on empty" but that's more code for no reason. CDX is reliable; availability isn't. Just use CDX.
- **Keep the public signature of `waybackLookup` identical.** `waybackLookup(origURL string, timeout time.Duration) (*memento, error)`. Same inputs, same outputs. Only the internal implementation changes. This keeps every caller site unchanged.
- **CDX query uses `limit=-1&filter=statuscode:200&fl=timestamp,original`.** Returns the single most recent 200-status snapshot with just the fields we need. Minimal response body, clean parsing.
- **Parse CDX JSON as `[][]string`** (array of arrays of strings) — CDX returns CSV-like JSON. First row is the field header. Subsequent rows are snapshots. Use Go's `encoding/json` with `[][]string` target.
- **URL canonicalization order on CDX miss:** original URL → strip trailing slash → strip `www.` prefix → give up. Three attempts max. Stops at the first hit.
- **End-to-end submit budget is configurable via `--submit-timeout`.** Default value: 10 minutes. Rationale: matt's principle is "don't pre-decide things for the user." Archive.is's current 9-minute stall might be fixed tomorrow, or it might get worse, or a specific URL might legitimately take 8 minutes to capture. The CLI's job is to make cancellation work correctly (context propagation, Ctrl-C, progress visibility) and let the user choose how long to wait. A 10-minute default is generous enough to cover the observed pathology, short enough that automated usage in CI doesn't hang indefinitely, and override-able for power users.
- **Why 10 minutes, not "no budget":** something has to cap the worst case. A CLI that never times out can leave a CI job running for hours. 10 minutes is the "reasonable default" — way beyond archive.today's documented 30-120 second typical, generous enough that a genuinely slow-but-succeeding submit will complete, but not so long that a forgotten script hangs a terminal overnight. Users with specific needs override it.
- **Context derivation for submit:** `submitCapture` creates one parent context using `--submit-timeout` as the deadline. Each `tryMirrorOnce` gets a sub-context derived from the parent with its own per-request timeout (default 180s per request, also reuses `--timeout` if the user set one). When the parent expires, all in-flight mirror attempts are cancelled. When the user hits Ctrl-C, signal handling cancels the parent immediately and the cancellation propagates to any in-flight request.
- **Progress output goes to stderr, not stdout.** Same pattern as existing agent-hints + menu output. Keeps stdout clean for piping the result.
- **Progress format:** simple. On submit start, print "Submitting... watch https://archive.ph/<url> for the snapshot" to stderr. Then every 10 seconds print "...still waiting (<elapsed>)". On give-up, print "...took too long. Archive.today may still be processing in the background. Check https://archive.ph/<url> in a few minutes." No colors, no unicode, no box-drawing. Machine-parseable where possible.
- **Progress ticker uses `time.Tick()` in a goroutine with a done channel.** Standard Go pattern. Cancels cleanly when the main goroutine returns.
- **Do NOT change the rate-limit cooldown logic.** The existing `cooldownError()` short-circuit stays. If we detect cooldown at the start of submit, we never get to the progress ticker — the user gets the cooldown error immediately, which is correct.
- **Budget exhaustion returns a `SubmitFailureError` with a new field.** Add `BudgetExhausted bool` to the struct. The formatter renders it differently from the per-mirror 429 case: "Archive.today took too long to respond across all mirrors (5 min budget exhausted). The capture may still complete on their side..."

## Open Questions

### Resolved During Planning

- **Should we add a `--submit-timeout` flag?** No. Sane default first. Flag later if needed.
- **Should CDX queries be rate-limited?** No. CDX is fast and Internet Archive has no documented per-IP quota for this endpoint. If we hit one, the CLI surfaces the HTTP error naturally.
- **Should progress output be suppressed with `--quiet`?** Yes. Same gating as other stderr output — `--quiet`, `--agent`, `--json`, or piped stdout disables the ticker. `--no-prompt` does NOT disable it (non-interactive agents still benefit from seeing elapsed time).
- **Can we reuse the progress ticker for `request --wait`?** Not now. Different flow, different state. Possible follow-up but out of scope.
- **What if CDX returns an error status?** Treat as "no snapshot found" — same as empty result. CDX is generally reliable; errors there mean we can't determine what's archived, and Wayback as a fallback is exhausted. The caller handles the "no snapshot" case already.

### Deferred to Implementation

- **Exact CDX response parsing edge cases** — the CDX docs claim the response is always `[][]string` but production responses may include malformed rows. Implementation should handle gracefully and fall through to the canonicalization fallback.
- **Should the progress ticker show a spinner character?** Maybe. Look at what reads well in iTerm vs piped output. Decide at implementation.
- **Whether to include the count of mirrors tried in the progress output** — "[mirror 2/6, attempt 3/4] 45s elapsed" might be useful or noisy. Decide empirically.

## Implementation Units

- [ ] **Unit 1: Replace Wayback availability API with CDX API in `waybackLookup`**

**Goal:** Make `waybackLookup` actually return snapshots that exist. The current implementation queries an unreliable endpoint; switching to CDX is expected to flip 12 out of 15 tldr failures to successes on today's test matrix.

**Requirements:** R1, R2, R3, R7, R8

**Dependencies:** None. Independent.

**Files:**
- Modify: `internal/cli/read.go` — `waybackLookup()` implementation (function signature unchanged)
- Test: `internal/cli/read_test.go` (create if absent) — CDX response parser tests

**Approach:**
- Replace the body of `waybackLookup` with a CDX-based lookup. Query `https://web.archive.org/cdx/search/cdx?url=<origURL>&output=json&limit=-1&filter=statuscode:200&fl=timestamp,original`.
- Parse the response as `[][]string`. First row is a header (`["timestamp", "original"]`). Subsequent rows are snapshots. With `limit=-1` there should be at most one snapshot row. Defensive: if there are multiple, take the last (CDX is chronological).
- Extract the timestamp and original URL from the snapshot row. Construct the memento URL as `https://web.archive.org/web/<timestamp>/<original>`.
- Build a `memento` struct with `OriginalURL=origURL`, `MementoURL=<constructed>`, `CapturedAt=<parsed from timestamp>`, `Mirror="web.archive.org"`, `Backend=string(backendWayback)`. Return it.
- **URL canonicalization fallback:** if the first CDX query returns empty (header row only, no snapshots), retry with variants:
  1. Original URL as given
  2. Original with trailing slash stripped (`strings.TrimRight(s, "/")`)
  3. Original with `www.` prefix stripped from hostname
  If any variant returns a snapshot, use it. If all three fail, return the existing `"no wayback snapshot available"` error for backwards compatibility with callers that match on that string.
- Use the passed-in `timeout` for the HTTP request. Simple `http.Client{Timeout: timeout}` is fine here — CDX is a small, fast endpoint and doesn't have the slow-response pathology that afflicts archive.is submits. Unit 2's context-based approach is for the submit path specifically.
- Keep the User-Agent header the same as before (the `userAgent` constant).

**Patterns to follow:**
- Yesterday's `waybackLookup` structure — same arguments, same return type, same error shape.
- `timegateLookup` in the same file — similar cross-service HTTP client usage.
- Go's `encoding/json` — standard `json.Unmarshal` into `[][]string`.

**Test scenarios:**
- Happy path: CDX returns one snapshot row, waybackLookup returns a valid memento with correct timestamp, memento URL, and backend field.
- Happy path: CDX returns multiple rows (header + 3 snapshots), waybackLookup picks the last one.
- Happy path: CDX returns empty for the original URL but hits on URL-without-trailing-slash canonicalization. Returns a valid memento.
- Happy path: CDX returns empty for original and no-slash, but hits on no-www variant. Returns a valid memento.
- Edge case: CDX returns only the header row (no data), all three canonicalization variants return empty → function returns the "no wayback snapshot available" error.
- Edge case: CDX returns a row with empty timestamp or empty original — defensive skip, fall through to canonicalization variants.
- Edge case: CDX timestamp fails to parse as 14-digit format — defensive use zero time, still return the memento.
- Edge case: origURL is already without `www.` prefix — canonicalization skips the www-stripping step.
- Edge case: origURL is a path-less root (e.g., `https://example.com` without trailing slash) — trailing-slash canonicalization is a no-op; still tries other variants.
- Error path: HTTP 500 from CDX — return an error ("cdx HTTP 500") distinct from "no snapshot available" so callers can distinguish outage from miss.
- Error path: network failure (DNS, TCP refused) — return the underlying error wrapped.
- Error path: malformed JSON response — return a parse error wrapped.
- Integration (live): run against a known-archived URL like `https://simonwillison.net/` and confirm a real memento URL comes back. This is the test that catches regressions on the CDX endpoint format.

**Verification:**
- `go test ./internal/cli/ -run TestWaybackLookup` passes all scenarios above.
- Running `archive-is-pp-cli tldr https://simonwillison.net/` (a URL from yesterday's test matrix that previously failed) produces a real summary via Wayback.
- Re-running Phase A of yesterday's test matrix shows tldr pass rate jumping from 5/20 to 15+/20.

---

- [ ] **Unit 2: Context-based per-request timeout + configurable budget + progress visibility**

**Goal:** Make submit behave like a well-mannered CLI command. Per-request timeouts that actually fire, context cancellation that propagates through the whole flow, live stderr progress so the user sees what's happening, a **configurable** overall budget with a sane default, Ctrl-C that works, and a clear give-up message with next steps. The CLI does NOT pre-decide what "too long" means — the user does.

**Requirements:** R4, R5, R6, R7, R8, R9

**Dependencies:** None. Independent of Unit 1.

**Files:**
- Modify: `internal/cli/read.go` — `submitCapture()`, `tryMirrorWithBackoff()`, `tryMirrorOnce()`. Signatures change to accept `context.Context`.
- Modify: `internal/cli/root.go` — register new `--submit-timeout` persistent flag (or command-scoped, decide at implementation).
- Modify: `internal/cli/http_client.go` — possibly add a context-aware helper if needed.
- Modify: `internal/cli/submit_error.go` — add a `BudgetExhausted bool` field to `SubmitFailureError`, update the `Error()` formatter to render the new case.
- Test: `internal/cli/submit_capture_test.go` (create if absent) — tests for context cancellation, budget exhaustion, progress output, Ctrl-C handling, flag behavior.

**Approach:**
- **`--submit-timeout` flag.** Duration type, default `10m`. Registered as persistent (applies to `save`, `request --wait`, and any other submit-path command). Accessible via `flags.submitTimeout` on the rootFlags struct.
- **Parent context from flag.** At the top of `submitCapture`, create a parent `context.Context` with `context.WithTimeout(parentCtx, flags.submitTimeout)`. If `flags.submitTimeout` is zero or negative (someone explicitly disabled it), use `context.Background()` with no deadline — genuinely unbounded, only cancellable by signal.
- **Per-request timeout as sub-context.** `tryMirrorOnce` takes the parent context and wraps it with its own sub-context using the per-request timeout (reuses `flags.timeout` which defaults to 180s for the submit HTTP call, or a sensible override). The sub-context is applied to the HTTP request via `req.WithContext(ctx)`. This is the authoritative cancellation — Go's runtime respects context at every I/O boundary, including the slow-response case that bit us today. When the per-request timeout fires OR the parent deadline fires, the request is cancelled cleanly.
- **Remove reliance on `http.Client.Timeout` as the load-bearing mechanism.** The client may still set a very generous timeout as belt-and-suspenders, but the context is the authority.
- **Signal handling.** At the top of `submitCapture` (or better, at the CLI entry point), wire `signal.NotifyContext(context.Background(), os.Interrupt)` so the parent context is cancelled when the user hits Ctrl-C. Propagation to in-flight requests is automatic once the context is connected end-to-end. Exit code 130 on interrupt.
- **Backoff loop checks context.** Before each retry, `tryMirrorWithBackoff` checks if the parent context is still valid (`ctx.Err() == nil`). If it's expired or cancelled, give up immediately — no further retries, no more time.Sleep.
- **Backoff sleep respects context.** Replace `time.Sleep(delay)` with `select { case <-time.After(delay): case <-ctx.Done(): return ctx.Err() }`. This ensures backoff waits don't outlive the parent deadline or survive a Ctrl-C.
- **Progress output:** before the mirror loop starts, print the predictive URL message to stderr once:
  ```
  Submitting to archive.today... (budget: 10m, override with --submit-timeout)
    You can watch for the snapshot at: https://archive.ph/<orig_url>
  ```
  The budget string shows whatever `flags.submitTimeout` resolved to, so users see what they've got.
- **Live ticker:** run a goroutine alongside the mirror loop that prints an elapsed-time line to stderr every 10 seconds:
  ```
  ...still waiting (0:45 elapsed, 9:15 remaining)
  ...still waiting (0:55 elapsed, 9:05 remaining)
  ...still waiting (1:05 elapsed, 8:55 remaining)
  ```
  The ticker goroutine uses `time.Ticker` and the parent context's Done channel. Main goroutine closes a done channel on return. Ticker exits cleanly on either signal.
- **Gating progress output:** only show the predictive URL and ticker when `isInteractive(flags)` would return true — i.e., not `--quiet`, not `--agent`, not `--json`, not `--no-prompt`, stdout is a TTY. For non-interactive callers, structured error reporting covers the same ground.
- **Budget exhausted error.** When the parent context's deadline fires AND we didn't succeed, `submitCapture` returns a `SubmitFailureError` with `BudgetExhausted: true`, any per-mirror attempt results collected so far, and the budget value used. The `Error()` formatter renders it as:
  ```
  submit failed: archive.today did not respond within the submit budget (10m, override with --submit-timeout)
    archive.ph:   last seen mid-request at 5:30
    archive.md:   429 Too Many Requests (attempts: 4)
    archive.is:   not reached (budget exhausted first)
    ...

  The capture may still be processing on archive.today's side.
  Check https://archive.ph/<orig_url> in a few minutes — if it lands, `archive-is-pp-cli read <url>` will find it.

  Or retry with a longer budget:  archive-is-pp-cli save <url> --submit-timeout 20m
  Or submit manually at https://archive.ph/ in your browser.
  ```
  Gives the user three concrete paths forward: wait for async completion, retry with a longer budget, browser handoff. Matches Matt's principle of "try loudly, fail clearly, offer alternatives."
- **classifySubmitError** already handles `SubmitFailureError`. Extend it so budget-exhausted cases return `apiErr` (exit code 5) — different from rate-limited (exit code 7) because the user's remediation is different. A rate-limited user waits; a budget-exhausted user might just retry with a longer budget. Exit code 5 is "API error" which is the correct semantic — archive.today didn't give us a usable response in time.
- **Context cancellation vs. budget exhaustion.** Distinguish them:
  - `ctx.Err() == context.DeadlineExceeded` AND we initiated that deadline → budget exhausted, return `SubmitFailureError{BudgetExhausted: true}`
  - `ctx.Err() == context.Canceled` (usually from signal) → return `context.Canceled` directly, caller exits with 130
  Both paths respect the user's decision. The budget-exhaust is "the CLI gave up because the configured budget ran out." The cancellation is "the user hit Ctrl-C."

**Technical design:** *(directional, not implementation spec)*

```
submitCapture(parentCtx, origURL, anyway, flags):
  // existing cooldown short-circuit — keep as-is
  if inCooldown(): return cooldownError

  // Budget from flag. Zero/negative = unbounded (signal-only cancellation).
  budget := flags.submitTimeout
  if budget > 0:
    ctx, cancel := context.WithTimeout(parentCtx, budget)
    defer cancel
  else:
    ctx := parentCtx  // unbounded

  if isInteractive(flags):
    print "Submitting... (budget: <budget>, override with --submit-timeout)" to stderr
    print "Watch https://archive.ph/<orig_url> for snapshot" to stderr
    doneCh := start progress ticker goroutine(ctx, start_time, budget)
    defer close(doneCh)

  client := newArchiveHTTPClient  // client.Timeout is a generous belt-and-suspenders

  collect per-mirror failures
  for each mirror:
    if ctx.Err() != nil: break  // budget or user cancel

    mementoOrErr := tryMirrorWithBackoff(ctx, client, mirror, origURL, perRequestTimeout)
    if mementoOrErr is a memento: return memento
    collect the failure

  // Distinguish budget-exhaust from user-cancel from all-mirrors-failed
  switch:
  case ctx.Err() == context.Canceled:  // user Ctrl-C
    return ctx.Err()  // exits 130
  case ctx.Err() == context.DeadlineExceeded:  // budget ran out
    return SubmitFailureError{ Attempts: ..., BudgetExhausted: true, Budget: budget }
  case allAttemptsWere429:
    writeCooldown(...)
    return SubmitFailureError{ Attempts: ..., Cooldown: ... }
  default:
    return SubmitFailureError{ Attempts: ... }

tryMirrorWithBackoff(ctx, client, mirror, origURL, perRequestTimeout):
  attempts := 0
  for each backoff delay in [5s, 15s, 60s]:
    if ctx.Err() != nil: return ctx.Err()
    reqCtx, cancel := context.WithTimeout(ctx, perRequestTimeout)
    m, err := tryMirrorOnce(reqCtx, client, mirror, origURL)
    cancel()
    attempts++
    if m != nil: return m, attempts, nil
    if !is429(err): return nil, attempts, err  // non-429 errors don't retry

    // Wait before retry, but respect parent context.
    select:
      case <-time.After(delay):  // continue to next retry
      case <-ctx.Done(): return ctx.Err()
  return nil, attempts, lastErr

// ticker goroutine — prints elapsed/remaining every 10s
progressTicker(ctx, start, budget):
  ticker := time.Tick(10s)
  for:
    select:
      case <-ticker:
        elapsed := time.Since(start)
        remaining := budget - elapsed  // negative or N/A if budget is 0
        print "still waiting (<elapsed> elapsed<, remaining>)"
      case <-ctx.Done(): return
      case <-doneCh: return
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

**Patterns to follow:**
- Go stdlib: `context.WithTimeout`, `req.WithContext`, `time.Ticker`, goroutine + done channel.
- Existing `tryMirrorWithBackoff` shape — extend it to take a `ctx` argument as the first parameter (Go convention).
- Existing `SubmitFailureError` formatter for budget-exhausted rendering. Adding a new field is additive and safe for existing callers.

**Test scenarios:**
- Happy path: submit succeeds on first mirror in <30 seconds. No progress ticker output (< 10s elapsed). Returns memento cleanly.
- Happy path: submit succeeds on third mirror after 60 seconds. Ticker has printed several "still waiting" lines. Returns memento cleanly. Budget was never a factor.
- Happy path: user sets `--submit-timeout 20m`, submit succeeds at 12 minutes (slower than default 10m would allow). Verifies the flag is honored and non-default budgets work.
- Happy path: user sets `--submit-timeout 0` (unbounded), submit runs without deadline and only exits on success, per-request error, or user Ctrl-C.
- Edge case: user hits Ctrl-C after 30 seconds of waiting. Context cancellation propagates, ticker exits cleanly, main function returns context.Canceled, CLI exits 130 in under 1 second.
- Edge case: `--submit-timeout 2m` set on CI, archive.today is slow. Budget fires at 2 minutes, returns SubmitFailureError{BudgetExhausted: true, Budget: 2m}. Error message mentions the 2m budget and suggests `--submit-timeout 20m` for retry.
- Edge case: all 6 mirrors return 429 quickly (total wall time < 1 min). The existing rate-limit-cooldown path runs, no budget exhaustion, cooldown state written, existing behavior preserved.
- Edge case: `--quiet` set — no predictive URL line, no ticker output. Only errors go to stderr.
- Edge case: `--json` set — no predictive URL line, no ticker output. JSON error body on failure includes the `budget_exhausted` field if applicable.
- Edge case: `--agent` set — same as quiet+json. No progress noise.
- Error path: archive.is slow-responds for 10 minutes with default budget (like today's WSJ case would have been, but bounded). Parent context fires at 10 min mark, returns SubmitFailureError{BudgetExhausted: true}, `classifySubmitError` wraps with exit code 5 (apiErr, not rateLimitErr). Total wall time is ~10:05, not infinite.
- Error path: per-request timeout fires on mirror 1 (archive.ph takes >180s), mirror 2 responds successfully at 3 minutes total — returns memento from mirror 2. Ticker showed 3 minutes of progress. Per-request context cancellation worked even though the parent context is still valid.
- Error path: budget fires mid-backoff (e.g., after mirror 2's second retry, while sleeping in the 60s backoff). The context-aware sleep returns immediately, function returns budget-exhausted error. Attempted-mirror list includes mirrors 1 and 2 with their partial progress. No post-budget sleeps.
- Integration: run against a real but obscure URL (e.g., a fresh Hacker News item that hasn't been archived). Should either succeed within the default 10-minute budget OR fail with a clear budget-exhausted error that tells the user exactly how to retry with a longer budget.
- Integration: run with `--submit-timeout 30s` against a URL known to take longer. Confirms the CLI gives up at 30s, not later. Exit code 5, helpful error message, Ctrl-C not needed.
- Error path (unchanged): cooldown file says IP is throttled — returns cooldownError in <10ms, no ticker, no predictive URL hint (nothing to hint at). Budget-exhaust code path never runs.
- Ticker output format: with default 10m budget, ticker shows `still waiting (0:45 elapsed, 9:15 remaining)`. With `--submit-timeout 0` (unbounded), ticker shows `still waiting (0:45 elapsed)` without the remaining field.

**Verification:**
- `go test ./internal/cli/ -run TestSubmitCapture` passes all scenarios above.
- Running `archive-is-pp-cli save <fresh-url>` with default budget when archive.is is slow-responding exits within ~10:10 (the default budget), not 9+ minutes silently.
- Running with `--submit-timeout 2m` exits at ~2:05 with a clear budget-exhausted message.
- Running with `--submit-timeout 30m` lets the submit run up to 30 minutes if archive.is is that slow.
- Running with `--submit-timeout 0` runs unbounded — only Ctrl-C or a successful response ends the call.
- Stderr shows the predictive URL on start and periodic elapsed-time lines during the wait.
- `archive-is-pp-cli save <fresh-url> --quiet` shows NO progress output (quiet contract preserved).
- `archive-is-pp-cli save <fresh-url> --json` puts the error in the JSON payload with the new `budget_exhausted: true` field, no stderr progress.
- Ctrl-C during a long submit exits in <1 second with exit code 130. Works with any budget value including unbounded.
- `archive-is-pp-cli --help` shows the new `--submit-timeout` flag with its default and description.

## System-Wide Impact

- **Interaction graph:** Unit 1 touches `waybackLookup` only. Callers (`runTldr`, `newReadCmd` via backend iteration, `newGetCmd` via body-fetch fallback) see the same function signature. No ripple.
- **Unit 2 touches `submitCapture`, `tryMirrorWithBackoff`, `tryMirrorOnce`.** `tryMirrorWithBackoff` and `tryMirrorOnce` gain a `ctx` parameter — this is an internal API change. Callers within the same file are updated. Not exported, no downstream consumers.
- **Error propagation:** Unit 1 preserves the exact error strings (`"no wayback snapshot available"`) so existing match-on-string checks in callers keep working. Unit 2 adds a `BudgetExhausted` field to `SubmitFailureError`. Existing callers that check `isAllRateLimited()` or read `.Cooldown` continue to work. The budget-exhausted case goes through `classifySubmitError` → `rateLimitErr` exit code 7, which is correct behavior.
- **State lifecycle risks:** Unit 2's goroutine for progress ticker must be cleanly torn down on every return path (success, failure, cancellation). Use `defer close(doneCh)` at the goroutine start site and have the ticker goroutine select on `<-doneCh` to exit. No orphaned goroutines.
- **API surface parity:** No change to the CLI's flag set, command set, or output contract. All changes are internal. The only user-visible behavior changes are: (1) `tldr` works more often, (2) `save` shows progress and gives up at 5 minutes.
- **Integration coverage:** Unit 2's progress ticker is hard to unit-test faithfully. Include at least one integration test that runs `save` against a mock HTTP server configured to slow-respond, and asserts the budget fires at the expected time with the expected error type. Use `httptest.NewServer` with a deliberately slow handler.
- **Unchanged invariants:**
  - `read` command behavior is completely unchanged.
  - The rate-limit cooldown state file (`rate-limit.json`) is unchanged.
  - Exit codes 0 / 2 / 3 / 4 / 5 / 7 / 10 retain their existing meanings.
  - `--no-prompt`, `--quiet`, `--agent`, `--json`, `--no-clipboard`, `--copy-text` flag semantics are unchanged.
  - The MCP server (`archive-is-pp-mcp`) is not touched.
  - Yesterday's polish improvements (menu, hints, paywall warnings, silent-redirect detection, agent hints) all continue to work.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| CDX API has its own rate limits or outages we haven't hit yet | Unit 1's error path handles HTTP errors distinctly from "no snapshot." If CDX itself goes down, we surface the error message to the caller and existing Wayback-fallback paths in `runTldr` can still try the memento URL directly. The CLI degrades gracefully, doesn't crash. |
| CDX response format changes — Internet Archive rewrites the endpoint in the future | Unit 1's integration test against a live known-archived URL will catch this on first run. Not great but acceptable — archive-is-pp-cli is already a CLI that depends on unofficial APIs, and the maintenance model is "update when it breaks." |
| Go's `context.WithTimeout` has an edge case we haven't considered | Unlikely. `context.WithTimeout` is battle-tested across the Go ecosystem. Unit 2's test matrix covers the main cases (cancellation, deadline, mid-request, at-start). If a bug shows up in production, fix it as a follow-up. |
| The 10-minute default budget is too short for a genuinely slow but eventually-successful archive.is submit | The budget is user-configurable via `--submit-timeout`. The error message explicitly tells users how to retry with a longer budget. The default is generous (5-20x archive.today's documented typical) but not infinite, which prevents stuck CI jobs from hanging terminals indefinitely. Users who hit the default regularly can set a higher default in their shell or alias. |
| Users set `--submit-timeout 0` (unbounded) and archive.is never responds | The only exit path is Ctrl-C or a response. That's by design — the user explicitly asked for unbounded behavior. The progress ticker continues to fire so at least the user sees activity and can decide when to give up. |
| Signal handling conflicts with Cobra's default interrupt behavior | Cobra doesn't install signal handlers by default; we're free to wire `signal.NotifyContext` at the CLI entry point. Standard pattern. |
| Progress ticker flickering in some terminals | Not a real risk — we're just printing plain text lines to stderr, not doing cursor-up-and-clear redraws. |
| Goroutine leak if done channel close is missed | Standard Go risk. Deferred close at function top. The pattern is well-known and easy to review. |
| Existing tests depend on the "no wayback snapshot available" error string | Unit 1 preserves that string exactly. Grep for the string across test files before implementation and update any hard-coded matches if they exist (they probably don't, since waybackLookup's error path wasn't previously tested). |
| CDX canonicalization surprises — `www.` strip produces a URL CDX can't find even though the original works | The three-variant fallback tries the original first. If the original works, we never hit the variants. The variants are only attempted when CDX already returned empty for the original. |

## Documentation / Operational Notes

- **README update:** not strictly required — the public surface is unchanged. Consider adding a one-line note under Troubleshooting: "If `tldr` or `get` can't find a Wayback snapshot, the CLI will automatically try variants of the URL (trailing slash, www prefix). This usually succeeds."
- **CHANGELOG-style note in `.notes/improvements.md`:** mark items #10 and #11 as resolved with a date.
- **Rollout:** none. Ship via `go install` like before. No migration, no config changes.
- **Re-run the test matrix after implementation:** Phase A from `docs/plans/2026-04-11-002-test-archive-capture-matrix-plan.md`. Expected: tldr pass rate jumps from 5/20 to 15+/20. Update `docs/tests/2026-04-11-archive-capture-report.md` with the new numbers as before/after.

## Sources & References

- **Origin:** `docs/tests/2026-04-11-archive-capture-report.md` — the Phase A test report that surfaced both bugs
- **Notes entries:** `.notes/improvements.md` sections #10 (submit timeout) and #11 (CDX API swap) — detailed write-ups of both bugs with evidence
- **Previous session plan:** `docs/plans/2026-04-10-001-feat-archive-is-polish-improvements-plan.md` — established the submit flow, rate-limit cooldown, and mirror backoff that Unit 2 extends
- **Test matrix plan:** `docs/plans/2026-04-11-002-test-archive-capture-matrix-plan.md` — the smoke test that exposed both bugs
- **Related code:**
  - `internal/cli/read.go` — `waybackLookup`, `submitCapture`, `tryMirrorWithBackoff`, `tryMirrorOnce`
  - `internal/cli/http_client.go` — `newArchiveHTTPClient`, `backoffSchedule`
  - `internal/cli/submit_error.go` — `SubmitFailureError` type to extend
  - `internal/cli/rate_limit.go` — `classifySubmitError` for exit code routing
- **External:**
  - [Wayback CDX Server API docs](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server)
  - [Dave Cheney — Go network timeouts](https://dave.cheney.net/2014/09/14/go-network-transport-timeouts)
  - [Go `context` package](https://pkg.go.dev/context#WithTimeout)
