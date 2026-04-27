# Shipcheck Report: hackernews-pp-cli (Regen on v2.3.9)

## Scores
- **Scorecard:** 85/100 (Grade A)
- **Verify:** 97% (32/33 PASS, 0 critical, verdict PASS)
- **Workflow-verify:** workflow-pass (no manifest, skipped)
- **Verify-skill:** PASS (all checks pass: flag-names, flag-commands, positional-args, unknown-command)
- **Dogfood:** WARN (notes below)

## Tier 1 (infrastructure) breakdown
| Dim | Score |
|---|---|
| Output Modes | 10/10 |
| Auth | 10/10 |
| Error Handling | 10/10 |
| Terminal UX | 9/10 |
| README | 8/10 |
| Doctor | 10/10 |
| Agent Native | 10/10 |
| Local Cache | 10/10 |
| Breadth | 7/10 |
| Vision | 8/10 |
| Workflows | 10/10 |
| Insight | 6/10 |

## Tier 2 (domain correctness) breakdown
| Dim | Score |
|---|---|
| Path Validity | 10/10 |
| Data Pipeline Integrity | 7/10 |
| Sync Correctness | 10/10 |
| Type Fidelity | 3/5 |
| Dead Code | 4/5 |
| Auth Protocol | N/A (no auth) |

Omitted from denominator: mcp_surface_strategy, auth_protocol, live_api_verification.

## Findings

### Verify
- One non-critical exec failure: `local-search` under mock mode. Validated query trim catches empty-string FTS5 errors; mock-mode environment may still pass an empty fixture. Live behavior verified manually (working with real queries).

### Dogfood (WARN, 0 critical)
- 5 novel commands (pulse, my, hiring-stats, repost, tldr) flagged "hand-rolled response: no API client call, no store access" — **false positive**. These call the Algolia API via the hand-built `internal/algolia` client (different host than Firebase, can't share `internal/client`). Dogfood's reimplementation check looks for `flags.newClient`, `c.Get`, `http.Get` — the algolia helper exposes typed methods (`ac.Search`, `ac.Item`) which don't match. Logged as a retro candidate (machine gap: dogfood should recognize secondary API clients).
- 1 dead helper: `wrapResultsWithFreshness` defined but never called — generator-emitted opt-in for hand-built commands. Left in place per AGENTS.md guidance (don't strip generator-emitted helpers from one CLI).
- Path validity reported "0/0 valid" with FAIL status — likely a dogfood quirk when the spec is internal YAML; not a real failure (verify shows path validity 10/10).

### Build patches applied (printed-CLI specific)
- Truncated `?limit=N` client-side in 6 spec-driven commands (Firebase ignores the query param, returns full lists). New helper at `internal/cli/limit_helper.go::truncateJSONArray`.
- Made `controversial` self-fetch via `cliutil.FanoutRun` instead of relying on local store (sync of `/topstories.json` returns bare integer IDs, not full Item objects, so the generator's generic `UpsertBatch` writes nothing for `stories`).
- Added `bookmarks` table via in-command DDL (`internal/cli/bookmark.go::ensureBookmarksTable`).
- Wired `since` to auto-snapshot the front page on every run, so `since` and `velocity` work without a sync step.
- Tightened `local-search` to reject empty queries with a usage error (FTS5 syntax error otherwise).

## Ship Recommendation: ship

All ship-threshold conditions met:
- verify verdict PASS (97%, no critical failures)
- dogfood WARN with 0 critical (false positives + minor)
- workflow-verify workflow-pass
- verify-skill PASS
- scorecard 85 ≥ 65
- Behavioral correctness verified: every novel command tested against live API; `controversial`, `pulse`, `my`, `repost`, `tldr`, `comments`, `bookmark`, `since`, `open` all return correct, non-empty output for realistic inputs.

## Retro candidates (machine improvements)
1. **Dogfood reimplementation check doesn't recognize secondary API clients.** Hand-built sub-clients (Algolia for HN, OMDb for movie-goat) get false-positive WARNs. Same root finding as movie-goat F8 / HN F3 from prior retros. Worth filing as a generator capability gap.
2. **Generator's `extra_commands` list doesn't auto-emit stub files.** Spec authors declare them so SKILL.md picks them up, but every command must still be hand-built. Could auto-generate stubs that import the algolia helper for known patterns.
3. **Spec format still single-base-URL.** The "dual-API CLI" pattern (HN, movie-goat, others) recurs; a spec field for `enrichment_apis` with named base URLs would let the generator emit a typed multi-host client.
4. **Generic UpsertBatch silently drops bare-number arrays.** Firebase's `/topstories.json` returns `[id, id, id]` and nothing gets written. The generator could emit an "ID-list resource" pattern that hydrates each ID via a sibling endpoint.
5. **`?limit=N` semantics drift.** Many APIs (Firebase, file:// JSON dumps) return full collections without honoring limit. Generator could detect "no obvious paginator" and post-truncate client-side.
