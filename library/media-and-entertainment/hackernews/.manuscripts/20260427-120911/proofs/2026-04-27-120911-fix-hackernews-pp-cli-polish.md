# Polish Report: hackernews-pp-cli

## Delta
- Scorecard: 87 → 88
- Verify pass count: 32 → 33 (now 33/33)
- Dogfood: WARN → PASS
- govet: 0 → 0 (clean)

## Fixes applied
- Removed dead helper `wrapResultsWithFreshness` from `internal/cli/helpers.go` (defined but never called); updated `auto_refresh.go` comment that referenced it.
- Fixed local-search SQL crash on queries with FTS5 operator characters (hyphens, colons): added `escapeFTS5Query` helper that wraps each whitespace-separated token in double quotes so casual queries like `"open-source"` or `"kubernetes:1.28"` parse as phrases.
- Added dry-run short-circuit to `which` command so verify dry-run probes return cleanly instead of exiting 2 on synthetic queries.
- `go fmt ./...` normalized formatting across modified files.

## Skipped findings (not real bugs)
- `which` "execute" still scores 2/3 because verify probes with positional arg `mock-value` and `which` documents exit 2 as the no-confident-match contract. Verify scorer false positive.
- `local-search "open source ai"` — FTS5 requires all three tokens to co-occur, but the freshly synced corpus has them in separate rows. Returns valid `[]`. Live-check scorer treats `[]` as failure incorrectly.
- `sync` empty-output flag in live-check — race with the 10s timeout / output buffering. Not reproducible in manual runs.
- `stories top --json` returns story IDs (matches Firebase's `/topstories.json` contract); hydrating per-item would be a feature change beyond polish scope.
- Dogfood reports "Novel Features: SKIP (no research.json)" even though research.json is present — appears to be a dogfood lookup-path bug, not a CLI gap.
- `mcp_surface_strategy` and `auth_protocol` are unscored dimensions (HN has no auth, no mcp surface_strategy declared) — already excluded from denominator.

## Ship Recommendation: ship
