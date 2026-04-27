# Food52 Shipcheck Report

**Run:** 20260426-230853
**Verdict:** `ship`

## Gate Outputs

### dogfood — WARN

```
Examples:          10/10 commands have examples (PASS)
Novel Features:    7/7 survived (PASS)
Data Pipeline:     PARTIAL (sync calls domain Upsert; search uses direct SQL)
Verdict: WARN — 30 dead helper functions found
```

The dead helpers are leftover scaffolding from the generated `extractHTMLResponse` path that my food52-specific replacements bypass. Cosmetic, not functional. Documented as a Printing Press gap in the acceptance report.

### verify — PASS

```
Pass Rate: 83% (15/18 passed, 0 critical)
Verdict: PASS
```

The 3 mock-mode failures (`print`, `scale`, `which`) are commands requiring positional args that mock verify can't supply — not real failures. Real-API exercise of all three passed in Phase 5 dogfood (44/44).

### workflow-verify — PASS

```
Overall Verdict: workflow-pass
  - no workflow manifest found, skipping
```

### verify-skill — PASS

```
✓ All checks passed (flag-names, flag-commands, positional-args)
```

After fixing two SKILL.md examples that called `articles get` without a slug.

### scorecard — 79/100 Grade B

```
Output Modes   10/10
Auth           10/10
Error Handling 10/10
Terminal UX    9/10
README         8/10
Doctor         10/10
Agent Native   10/10
Local Cache    10/10
Breadth        7/10
Vision         8/10
Workflows      8/10
Insight        4/10

Domain Correctness
Data Pipeline Integrity 10/10
Sync Correctness        10/10
Type Fidelity           4/5
Dead Code               0/5

Total: 79/100 - Grade B
```

Path Validity, Auth Protocol, and live API verification omitted from the denominator (synthetic CLI, no auth, no path-spec to validate against).

## Top blockers found

1. **Examples missed by dogfood** — `strings.TrimSpace` stripped the indentation needed for dogfood's parser. Fixed across 17 files.
2. **"undefined" tokens in ingredient strings** — Food52's JSON-LD pre-rendered them when the source CMS field was unset. Fixed with `cleanIngredientStrings` post-processor.
3. **SKILL examples missing positional args** — verify-skill caught `articles get --agent --select ...` without a slug. Fixed both occurrences.

## Fixes applied

| Category | Fix |
|---------|-----|
| 1. Build break | n/a — generation succeeded first try |
| 2. Invalid path / auth mismatch | n/a — no auth, all paths valid |
| 3. Dead flags / functions / ghost tables | Left dead helpers in helpers.go (cosmetic; documented for retro) |
| 4. Broken dry-run / runtime failures | scale errors gracefully on yield-less recipes (correct behavior, not a fix) |
| 5. Missing novel features | None — all 7 transcendence features built and verified |
| 6. Scorecard polish | Examples + verify-skill + ingredient-undefined fixes |

## Before / after

- Before fixes: dogfood Examples 0/10 (FAIL), verify-skill 1 error
- After fixes: dogfood Examples 10/10 (PASS), verify-skill 0 errors

- Before scorecard: not run before fixes
- After scorecard: 79/100 Grade B

- Live dogfood: 44/44 PASS

## Final ship recommendation

**`ship`**

Every ship-threshold condition is met:
- `verify` PASS, 0 critical failures
- `dogfood` WARN-only (30 dead helpers, cosmetic)
- `workflow-verify` workflow-pass
- `verify-skill` exits 0
- `scorecard` 79 (>= 65)
- Behavioral correctness: 44/44 live tests pass; flagship features (recipes search, recipes top, recipes get, pantry match, scale, print) verified semantically correct on real Food52 data
- No known functional bugs in shipping-scope features.

The user constraint (unauthenticated only, no sign-in) is honored — the CLI ships with `auth.type: none`, no Vercel clearance cookie required, and the Typesense search-only key is auto-discovered from Food52's public JS bundle at runtime (never persisted into source).
