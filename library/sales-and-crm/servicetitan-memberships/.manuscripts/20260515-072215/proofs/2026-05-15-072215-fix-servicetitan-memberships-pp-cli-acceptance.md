# Phase 5 Live Dogfood Acceptance — servicetitan-memberships

## Level
Quick check (user-selected; production tenant, no approved mutation fixture).

## Binary-owned matrix
- matrix_size: 5
- tests_passed: 5
- tests_failed: 0
- tests_skipped: 3 (all `command path [api] has fewer segments than placeholders (1)` or `no positional argument` — known shape constraints, not failures)
- verdict: PASS
- gate marker: `phase5-acceptance.json` written, status=pass

## Manual smoke supplement (read-only against live JKA tenant)
| Probe | Result |
|---|---|
| `doctor --json` | `auth=configured`, `auth_source=oauth2`, `base_url=https://api.servicetitan.io/memberships/v2`, `api=reachable` |
| `membership-types get-list <tenant> --page-size 3 --json` | exit 0; returned real membership-type ("Water Treatment Maintenance", id 67026827, created 2024-03-06). Composed auth + ST-App-Key header + OAuth2 bearer + tenant substitution all confirmed working end-to-end. |
| `memberships customer-get-list <tenant> --page-size 2 --json` | exit 0; returned empty `data: []` envelope. Valid empty result — the test workspace has no customer memberships yet. Not an error. |

## Fixes applied during Phase 5
None — quick matrix and manual smoke both passed first try.

## Printing Press issues observed
The same v4.6.1 generator regressions surfaced as in `servicetitan-inventory` and `servicetitan-pricebook` runs:
- Composed-auth apiKey half not wired (#1303 apiKey half) — patched in Phase 2.5 by mirroring the sibling pricebook template (StAppKey field + Load() env-read + client.go ST-App-Key header injection + doctor checks).
- `defaultSyncResources()` / `syncResourcePath()` emitted as empty stubs (#1305) — patched.
- `{tenant}` placeholder not auto-substituted (#1332) — partial mitigation via the sync-resource-path tenant injection; generated list commands still take tenant as positional `args[0]`, matching sibling pricebook's shipped behavior on the same v4.6.1 binary.
- README env-vars table generated with only `ST_CLIENT_ID`/`SECRET`; missing `ST_APP_KEY` + `ST_TENANT_ID` — patched in Phase 4.9.
- MCP install snippets generated with only `ST_CLIENT_ID` env var — patched in Phase 4.9.
- Config-file path slug in README used `memberships-pp-cli` instead of `servicetitan-memberships-pp-cli` — patched in Phase 4.9.

All retro-candidates for the next Printing Press improvement pass; identical pattern to the inventory retro carry-forward.

## Gate
**PASS.** Proceed to Phase 5.5 (polish).
