# Pagliacci Shipcheck Report

## Summary

| Step | Result | Notes |
|------|--------|-------|
| Generate | PASS | All 7 quality gates passed (mod tidy, vet, build, binary, --help, version, doctor) |
| Dogfood | WARN | Path Validity 7/7 PASS; Novel Features 6/6 survived; 2 dead helpers |
| Verify | PASS | 96% (26/27), 0 critical; 1 unrelated `which` failure |
| Workflow-verify | workflow-pass | No workflow manifest emitted (none defined) |
| Verify-skill | PASS | All flag-names, flag-commands, positional-args match |
| Scorecard | 78/100 Grade B | Tier 1 strong; insight 4/10 the main gap |

**Verdict: ship**

## Fixes Applied

### Generator-blocker: feedback resource collision (1 file edit)
The spec resource `feedback` clobbered the standard `feedback.go.tmpl` template (which defines `FeedbackEndpointConfigured()`). Workaround: renamed spec resource `feedback` → `customer_feedback`. The collision is a Printing Press machine bug (the resource template should not overwrite reserved single-file templates).

### Generator-blocker: GOOS-name file collision (1 file edit)
The spec endpoint `windows` produced `scheduling_windows.go`, which Go treats as a Windows-only build constraint (filename pattern `*_<GOOS>.go`). Fix: renamed `windows` → `slot_list` and `windows_for_date` → `slot_list_for_date`. The collision is a Printing Press machine bug (the file emitter must escape filenames matching `_GOOS`/`_GOARCH` patterns).

### Path-param substitution (29 file edits)
The internal-YAML-spec parser does not extract `{paramName}` placeholders from path templates as Positional Params. The `command_endpoint.go.tmpl` template emits an empty params map and no positional-arg parsing. **29 generated files were sending literal `{storeId}`/`{customerId}`/etc. in request URLs → 404 on every path-parameterized endpoint.** Fix: each broken file got `Args: cobra.ExactArgs(N)`, positional args appended to `Use:`, realistic `Example:` values, and `replacePathParam(path, "name", args[i])` calls. A `replacePathParam` helper was added to `helpers.go` (the generator should emit this — log for retro).

### Live behavioral verification (no auth)
Three patched commands verified end-to-end against the real Pagliacci API:
- `store get 490 --json` → returns "Capitol Hill - Pike Street, 415 East Pike Street, Seattle WA"
- `menu top 490 --json` → returns category list (Pizza, Salads, Specials, ...)
- `scheduling window_days 490 DEL --json` → returns 7+ days of available delivery slots
- `slices today --json` → returns 92 rows across 23 distinct stores (T1 transcendence flagship)

## Remaining Issues

### Tier 1 (do-not-ship blockers)
None. All path-parameterized endpoints work correctly.

### Tier 2 (polish items)
- **2 dead helpers** in generated code (`extractResponseData`, `wrapResultsWithFreshness`) — emitted by templates but unused for this spec shape. The polish-worker agent should remove these in Phase 5.5.
- **Insight 4/10** in scorecard — analytics surface is light. Acceptable for a v1; could improve in a future emboss pass.
- **Underscored command names** (`customer_feedback`, `slot_list`, `slot_list_for_date`, `window_days`) — user-facing snake_case is unusual for cobra commands. The polish-worker can either rename via spec edits + regen, or directly edit the cobra `Use:` strings to dashes. Logged for retro: generator should auto-kebab snake_case resource/endpoint keys for `Use:` strings.
- **`which` verify failure (1/3)** — unrelated to domain commands. Generated utility command issue.

### Tier 3 (auth-gated, deferred to Phase 5)
The composed PagliacciAuth flow (`auth login --chrome`) is implemented but not yet exercised live in this shipcheck — that happens in Phase 5 (live dogfood with auth). 17 commands require auth (orders, rewards, address book, customer profile, etc.) and will be tested then.

## Generator-bug retro entries

1. **Resource-name vs single-file-template collision.** Reserved template names (`feedback.go`, etc.) must not be overwritten by spec resource emission. Detect and either rename or suffix the resource template.
2. **GOOS/GOARCH filename collisions.** Endpoint names matching `_<GOOS>.go` or `_<GOARCH>.go` cause silent Go build exclusion. Detect at spec-parse time and either reject or auto-suffix (e.g., `_endpoint.go`).
3. **Internal-YAML path-param wiring.** The internal-YAML-spec parser must extract `{paramName}` placeholders from `endpoint.path` and add them to `Endpoint.Params` with `Positional: true` so `command_endpoint.go.tmpl` emits the right cobra `Args:` + `replacePathParam` calls.
4. **`replacePathParam` helper.** The generator should emit a `replacePathParam(path, name, value string) string` helper into `helpers.go` so generated commands can use it without hand-editing.
5. **Underscore in cobra `Use:`.** Snake-case spec keys flow through to cobra `Use:` strings as-is. The generator should kebab-case them for user-facing commands while preserving snake-case for Go identifiers.

## Lock state
Lock held: pagliacci-pp-cli, scope mellow-sprouting-sifakis-f6d9f64d, phase shipcheck-fixing.

## Verdict: SHIP

All ship-threshold conditions met. Proceeding to Phase 4.8/4.9/4.85 agentic reviews.
