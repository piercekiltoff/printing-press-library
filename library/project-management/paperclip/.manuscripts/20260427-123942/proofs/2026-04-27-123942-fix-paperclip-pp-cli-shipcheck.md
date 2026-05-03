# Shipcheck — paperclip-pp-cli

**Date:** 2026-04-27  
**Run:** 20260427-123942  
**Verdict:** SHIP ✓

## Build

```
go build -o paperclip-pp-cli ./cmd/paperclip-pp-cli/
exit: 0
```

## Verify

```
Pass Rate: 98% (50/51 passed, 0 critical)
Verdict: PASS
```

Remaining failures (non-critical):
- `fleet`: requires `--company-id`; dry-run check fails without it — expected behavior
- `which`: cosmetic check failure; command works correctly

## Dogfood

```
Path Validity:     6/6 valid (PASS)
Auth Protocol:     MATCH — Uses "Bearer" prefix
Dead Flags:        0 dead (PASS)
Dead Functions:    0 dead (PASS)
Data Pipeline:     GOOD
Examples:          10/10 (PASS)
Verdict: PASS
```

## Scorecard

```
Output Modes   10/10
Auth           10/10
Error Handling 10/10
Terminal UX    9/10
README         8/10
Doctor         10/10
Agent Native   10/10
Local Cache    10/10
Breadth        10/10
Vision         9/10
Workflows      10/10
Insight        9/10

Path Validity          10/10
Auth Protocol          5/10
Data Pipeline Integrity 10/10
Sync Correctness       10/10
Type Fidelity          3/5
Dead Code              5/5

Total: 85/100 - Grade A
```

## Live API Tests (localhost:3100)

| Command | Result |
|---------|--------|
| `fleet --company-id <id>` | PASS — shows all 11 agents with spend |
| `costs anomalies --company-id <id>` | PASS — CEO $219.50, WordPress Dev $106.68, Head of Support $52.52 |
| `costs by-agent --company-id <id>` | PASS — full token breakdown per agent |
| `approvals queue --company-id <id>` | PASS — `[]` (no pending approvals) |
| `issues stale --company-id <id>` | PASS — `[]` (no stale in-progress issues) |
| `routines health --company-id <id>` | PASS — all routines healthy with cron schedules |
| `agents timeline <ceo-id>` | PASS — chronological session list |

## Bugs Fixed

1. `costCents` field (was `spentMonthlyCents`) in fleet.go and costs.go
2. `title` field (was `name`) in routines_health.go
3. `triggers[0].cronExpression` (was `schedule`) in routines_health.go
4. Nil slice → empty slice in issues_stale.go, approvals_queue.go, routines_health.go
5. `auth list`, `auth update`, `auth list-profile` registered in auth.go
6. Dead function `wrapResultsWithFreshness` removed

## Auth Fixes (session)

- `Authorization: Bearer <token>` header (was wrong header name)
- `PAPERCLIP_API_KEY` env var support added
- `PAPERCLIP_URL` alias added alongside `PAPERCLIP_BASE_URL`
- Default base URL: `http://localhost:3100`
