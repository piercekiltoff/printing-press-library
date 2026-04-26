# Pagliacci Polish Report (Phase 5.5)

## Delta
| Metric | Before | After |
|--------|--------|-------|
| Scorecard | 71 (B) | **80 (A)** |
| Verify pass-rate | 26/27 (96%) | 26/27 (96%) (unchanged — `which` failure is verify-framework artifact) |
| Dogfood verdict | WARN | **PASS** |
| go vet | 0 | 0 |

## Fixes applied
- Removed dead helper `extractResponseData` (~26 lines, never called)
- Removed dead helper `wrapResultsWithFreshness` (~10 lines, never called) and its docstring reference in `auto_refresh.go`
- Removed 6 ghost `*search` entries from `readCommandResources` freshness registry (`address|credit|customer|gifts|orders|store search`) — those subcommands were never registered with cobra
- Removed ghost `customer list` entry from freshness registry — `customer` has no `list` subcommand
- Removed corresponding ghost paths from README "Covered command paths" list (24 → 17 entries)
- Removed boilerplate "Retryable" claim from README Agent Usage section — CLI does not implement idempotent create/delete on retry
- Replaced empty `Config file:` value in README with real path `~/.config/pagliacci-pp-cli/config.json` and added env vars (`PAGLIACCI_CONFIG`, `PAGLIACCI_BASE_URL`, `PAGLIACCI_NO_AUTO_REFRESH`)
- Fixed `--agent --select` interaction in two SKILL recipes (`orders summary` and `address list`) — swapped `--agent` for `--json` because `--agent` enables `--compact` which strips fields before `--select` can pick them

## Skipped (with rationale)
- **`which` verify failure (1/27)**: scorer artifact, not a CLI bug. The verifier auto-derives a positional arg from the `[query]` placeholder, passes synthetic `mock-query`, gets "no match" with exit 2, counts that as failure. The command is correct: empty arg lists the index, matched arg returns matches, unmatched arg correctly exits 2.
- **live_check 0/6 in `scorecard --live-check`**: all 6 are environmental, not CLI bugs (token-matching heuristic, no seeded address labeled `home`, upstream API state, empty test account, SQLITE_BUSY from concurrent runs).
- **insight 4/10 dimension**: would require deeper analytics (trends, anomalies). Out of polish scope.

## Sanity verification (post-polish)
- `go build ./cmd/pagliacci-pp-cli/` — exit 0
- `go vet ./...` — clean
- `go test ./...` — all pass (cli, cliutil, store packages)
- `slices today --json` — 92 rows (unchanged)
- `rewards card --json` — returns card data (auth still wired)
- `orders reorder --last --clone-only --json` — returns clone of last order (Phase 5 fixes preserved)

## Verdict: ship
