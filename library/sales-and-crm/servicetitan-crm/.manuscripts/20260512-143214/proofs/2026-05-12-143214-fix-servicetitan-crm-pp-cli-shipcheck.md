# Shipcheck Report: ServiceTitan CRM pp-cli

Run: `20260512-143214`
Spec: `tenant-crm-v2.enriched.json`
Binary: printing-press v4.5.1

## Umbrella verdict: **PASS (6/6 legs)**

| Leg | Result | Exit | Elapsed |
|---|---|---|---|
| dogfood | PASS | 0 | 3.83s |
| verify | PASS | 0 | 7.78s |
| workflow-verify | PASS | 0 | 505ms |
| verify-skill | PASS | 0 | 1.12s |
| validate-narrative | PASS | 0 | 1.03s |
| scorecard | PASS | 0 | 305ms |

## Verify (100% pass rate)
23/23 commands across 23 distinct top-level command groups. 0 critical failures.

Notable: **all 9 Phase 3 hand-built transcendence commands** pass help + dry-run + exec checks (`segments`, `sync-status`, plus subcommands of `customers`/`leads`/`bookings`).

## Validate-narrative (10/10 commands resolved)
Every README quickstart + SKILL recipe command resolves against the built CLI. The earlier `--since auto` issue was fixed pre-shipcheck.

## Scorecard: 86/100 (Grade A)

| Dimension | Score |
|---|---|
| Output Modes | 10/10 |
| Auth | 10/10 |
| Error Handling | 10/10 |
| Terminal UX | 9/10 |
| README | 8/10 |
| Doctor | 10/10 |
| Agent Native | 10/10 |
| MCP Quality | 8/10 |
| MCP Remote Transport | 10/10 |
| MCP Tool Design | 10/10 |
| MCP Surface Strategy | 10/10 |
| Local Cache | 10/10 |
| Cache Freshness | 5/10 |
| Breadth | 10/10 |
| Vision | 9/10 |
| Workflows | 8/10 |
| Insight | 8/10 |
| Agent Workflow | 9/10 |
| Path Validity | 10/10 |
| Auth Protocol | 9/10 |
| Data Pipeline Integrity | 7/10 |
| Sync Correctness | 10/10 |
| Type Fidelity | 1/5 |
| Dead Code | 4/5 |

**MCP architectural dimensions all 10/10** — confirming the Cloudflare-pattern collapse from 86 endpoints to 2 intent tools is correctly emitted.

## Gaps surfaced
- `type_fidelity scored 1/5` — needs improvement (will be addressed in polish).
- Sample-output-probe failed because scorecard looked for binary at `<workdir>/servicetitan-crm-pp-cli.exe` but it's at `<workdir>/build/stage/bin/servicetitan-crm-pp-cli.exe`. Path-resolution issue, not a behavioral bug.
- `Cache Freshness 5/10` — sync cursor / freshness reporting could be richer.

## Behavioral correctness check
- **Live composed-auth roundtrip verified** earlier (`customers get-list` returned real K&C Lending customer record from JKA tenant).
- **Sync to local store verified** — 100 customers + 100 leads pulled into SQLite, `sync-status` reports correct row counts.
- All 9 transcendence commands' help text + `--dry-run` paths exercised by the verify leg's mechanical matrix.

## Verdict: **ship**

All 6 legs PASS. No critical failures. Scorecard 86/100 ≥ 65 threshold. Composed auth verified live against JKA tenant. All 9 approved transcendence commands resolve to their planned leaf paths and pass mechanical verify.

Polish skill (Phase 5.5) will address the gaps: type_fidelity, sample-output-probe binary path, cache freshness reporting.
