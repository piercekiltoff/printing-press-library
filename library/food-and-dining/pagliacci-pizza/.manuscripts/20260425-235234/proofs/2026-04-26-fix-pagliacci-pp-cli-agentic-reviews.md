# Pagliacci Agentic Reviews — Phase 4.8 / 4.9 / 4.85

## Phase 4.8 (SKILL Semantic): 4 PASS, 2 FINDING
1. Trigger phrases: PASS
2. Novel-feature descs: FINDING (recipe used wrong field names — see F1)
3. Stub disclosure: PASS — 6/6 features in `novel_features_built` match `novel_features`
4. Auth narrative: PASS — `auth login --chrome`, `auth status`, `doctor` all wired correctly
5. Recipe claims: FINDING (slices today recipe broken — F1)
6. Marketing-copy smell: PASS

## Phase 4.9 (README/SKILL Correctness): 4 PASS, 2 FINDING
1. Commands resolve: FINDING — `stores list` and `scheduling time-window-days` referenced in troubleshooting don't exist (F2)
2. No placeholders: PASS
3. Boilerplate match: FINDING — Retryable claim, empty config path, ghost search paths (F4, F5, F7 — warnings)
4. Auth + public split: PASS
5. Brand name canonical: PASS — "Pagliacci Pizza" used consistently
6. Novel features doc: PASS — all 6 transcendence features documented

## Phase 4.85 (Output Plausibility): 3 PASS, 1 FINDING
1. Semantic match: FINDING — Recipe broken; live data correct (F1)
2. Format bugs: PASS — no entities, mojibake, or malformed URLs
3. Aggregation drops: N/A (no fan-out commands)
4. Result ordering: PASS — store-grouped, sortable
5. Live-check classification:
   | Feature | Result |
   |---------|--------|
   | slices today | HEURISTIC_FP (correct output, "today" token absent because rendered fields are store/slice/price) |
   | stores tonight | REAL_BUG (corrected before review) |
   | rewards stack | EXPECTED — no auth in scorecard env |
   | orders reorder | EXPECTED — no synced orders |
   | address best-time | EXPECTED — no auth |
   | orders summary | EXPECTED — no synced orders |

## Findings

### F1 [error → fixed] — Recipe used wrong field names + --agent override
**Was:** `slices today --agent --select store,slice,price` produced `[{},{},{}]`. Two bugs: (a) field names should be `store_name`,`slice_name`,`price`; (b) `--agent` includes `--compact` which overrode `--select`.
**Fix applied:** SKILL.md recipe changed to `slices today --json --select store_name,slice_name,price`. Verified live: returns 92 rows of real data.

### F2 [error → fixed] — Commands don't exist in README troubleshooting
**Was:** `stores list` (no such command) and `scheduling time-window-days` (no such subcommand).
**Fix applied:** README.md line 357 corrected to `store list` and `scheduling window_days <storeId> DEL`.

### F3 [warning → deferred to polish] — `--compact` override of `--select` in `--agent` recipes
The same field-vs-compact interaction will likely affect other agent-mode recipes once data is synced. Polish-worker can audit all `--agent --select` recipes.

### F4 [warning → deferred to polish] — "Retryable" boilerplate claim in Agent Usage
README claims idempotent create/delete semantics. The CLI doesn't implement these for Pagliacci endpoints. Boilerplate from generator template; polish-worker should drop or qualify.

### F5 [info → deferred to polish] — Empty config-path backticks
README line 343 shows `Config file: \`\`` (empty). Generator template substitution gap. Polish-worker can fix.

### F6 [info → no action] — Auth narrative accuracy verified
Acknowledgement; no fix needed.

### F7 [info → deferred to polish] — Ghost `*search` paths in freshness list
Lines 282-302 list `address search`, `credit search`, etc. — none exist as subcommands. Generator-side issue; polish-worker can prune.

## Generator-bug retro entry
**`--agent` flag bundle includes `--compact` which silently overrides explicit `--select`.** Either `--agent` should not include `--compact`, or `--select` should take precedence over `--compact` field filtering. This is a systemic UX bug across every Printing Press CLI, not specific to Pagliacci. Reviewer agent flagged this explicitly.

## Verdict
**SHIP** — both errors resolved. Warnings are polish-worker scope.
