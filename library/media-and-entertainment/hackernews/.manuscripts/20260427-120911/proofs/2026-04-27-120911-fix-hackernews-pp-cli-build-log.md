# hackernews-pp-cli Build Log (Regen on v2.3.9)

## Strategy

- Spec covers Firebase only (single base_url constraint unchanged from v1.3.3 → v2.3.9).
- Algolia helper hand-built in `internal/algolia/` — used by 7 commands.
- Novel commands hand-built in `internal/cli/` — wired into root.go.
- `cliutil.FanoutRun` used by `controversial` for parallel item fetches (8 workers).

## Generator output (Phase 2)

The generator emitted, on first try and with all 7 quality gates passing:
- root, doctor, agent-context, profile, feedback, which, export, import, search (local FTS), sync, workflow, api commands
- Per-resource generated commands: stories (top/new/best/get), ask, show, jobs, users, updates, maxitem
- internal/cliutil (fanout, freshness, probe, text, verifyenv) emitted
- internal/store with FTS5, snapshots, sync_state
- internal/client for Firebase
- README.md + SKILL.md

## Phase 3: Built features

### Foundation (Priority 0)
- `internal/algolia/algolia.go` — typed Algolia client (Search, Item) with tests.

### Absorbed (Priority 1)
- `comments` — Algolia /items tree fetch; --depth, --flat, --author, --match, --since filters.
- `live-search` — Algolia search with --tag/--since/--until/--min-points/--by-date/--page/--hits-per-page.
- `hiring` — finds latest "Ask HN: Who is hiring" thread, regex filter on top-level posts.
- `freelance` — same shape, finds Freelancer thread.
- `open` — print/launch URL. Side-effect convention honored: `--launch` to act, `cliutil.IsVerifyEnv` short-circuit, prints by default.
- `bookmark add/list/rm` — local SQLite bookmarks table.

### Transcendence (Priority 2)
- `since` — front-page diff with snapshot table; auto-snapshots on every run, diffs against previous.
- `pulse` — Algolia date-bucketed topic aggregation; top stories + per-day mentions/points.
- `repost` — Algolia URL search with substring filter; pre-submit dupe check.
- `my` — user submission stats: score buckets, median, best hour weighted by points.
- `hiring-stats` — cross-month aggregate of Who's Hiring threads; languages, remote %, top companies, visa counts.
- `controversial` — fetches top 100 IDs, fans out via cliutil.FanoutRun, ranks by descendants/score ratio.
- `velocity` — reads frontpage_snapshots for an ID, returns trajectory.
- `tldr` — deterministic thread digest: top authors by reply count, root vs reply, depth histogram, heat metric.
- `local-search` — FTS5 search across local store.

### Skipped/intentional
- `karma` (dropped from prior manifest in reprint reconciliation — re-implementation risk; karma is a single number, not a series).
- `timing` (dropped — low actionability; absorbed into `my`'s best-hour calculation).
- `local-search` mostly redundant with the generator's auto-emitted `search`; kept as a separate command per the manifest, may consolidate post-shipcheck.

## Generator-vs-API mismatch fixes (printed CLI specific)

- **`?limit=N` not honored by Firebase.** Generated `stories_top.go`, `stories_new.go`, `stories_best.go`, `promoted_ask.go`, `promoted_show.go`, `promoted_jobs.go` patched to truncate client-side via new `truncateJSONArray` helper in `internal/cli/limit_helper.go`. Same root cause as movie-goat F8 — single-base-URL constraint is broader than just dual-API; "limit query param honored" is also an API-specific assumption. Filed in retro candidates.

- **Sync of "stories" resource writes zero rows.** `/topstories.json` returns bare integers, not full Item objects; the generator's generic UpsertBatch silently skips bare numbers because `extractObjectID` expects an object with `id`. Worked around by making `controversial` self-fetch (top IDs + per-item live fetch). `velocity` and `since` only need IDs so still work via the snapshot table (populated independently by `since`). `local-search` will return empty until the user does additional work; documented in README. Filed in retro candidates as a generator capability gap.

- **`bookmarks` table created via in-command DDL** rather than a generator-supplied schema. `internal/cli/bookmark.go::ensureBookmarksTable` is idempotent (CREATE TABLE IF NOT EXISTS).

## Build status

```
go build ./...   PASS
go vet ./...     PASS  (clean)
go test ./...    pending (Phase 4)
```

7 quality gates passed at generation time, all still pass after Phase 3 patches.

## Lock heartbeat

Updated at: generate, build-p0, build-p1, build-p2.
