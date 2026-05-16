
=== dogfood ===
warning: resource "export" from path "/tenant/{tenant}/export/invoice-templates" would shadow framework cobra command "export"; renamed to "memberships-export"
Dogfood Report: servicetitan-memberships-pp-cli
================================

Path Validity:     0/0 valid (FAIL)

Auth Protocol:     MATCH
  Spec: bearer token format (expects "Bearer " prefix)
  Generated: Uses "Bearer" prefix
  Detail: spec and generated client both use "Bearer"

Dead Flags:        0 dead (PASS)

Dead Functions:    0 dead (PASS)

Data Pipeline:     PARTIAL
  Sync: calls domain-specific Upsert methods (GOOD)
  Search: uses generic Search only or direct SQL
  Domain tables: 1

Examples:          10/10 commands have examples (PASS)

Novel Features:    12/12 survived (PASS)

MCP Surface:       PASS (MCP surface mirrors the Cobra tree at runtime)

Verdict: PASS

=== verify ===
warning: resource "export" from path "/tenant/{tenant}/export/invoice-templates" would shadow framework cobra command "export"; renamed to "memberships-export"
Runtime Verification: C:\Users\pierc\printing-press\.runstate\temp-855456de\runs\20260515-072215\working\servicetitan-memberships-pp-cli\servicetitan-memberships-pp-cli.exe
Mode: mock

COMMAND                        KIND         HELP   DRY-RUN  EXEC     SCORE
agent-context                  read         PASS   PASS     PASS     3/3
analytics                      data-layer   PASS   PASS     PASS     3/3
api                            local        PASS   PASS     PASS     3/3
auth                           local        PASS   PASS     PASS     3/3
bill-preview                   read         PASS   PASS     FAIL     2/3
complete                       read         PASS   FAIL     FAIL     1/3
doctor                         local        PASS   PASS     PASS     3/3
drift                          read         PASS   PASS     FAIL     2/3
expiring                       read         PASS   PASS     FAIL     2/3
feedback                       read         PASS   PASS     PASS     3/3
find                           read         PASS   PASS     PASS     3/3
health                         data-layer   PASS   PASS     PASS     3/3
import                         data-layer   PASS   PASS     PASS     3/3
overdue-events                 read         PASS   PASS     FAIL     2/3
profile                        read         PASS   PASS     PASS     3/3
recurring-service-events       read         PASS   PASS     PASS     3/3
renewals                       read         PASS   PASS     FAIL     2/3
revenue                        read         PASS   PASS     FAIL     2/3
risk                           read         PASS   PASS     FAIL     2/3
schedule                       read         PASS   PASS     FAIL     2/3
search                         data-layer   PASS   PASS     PASS     3/3
stale-services                 read         PASS   PASS     FAIL     2/3
sync                           data-layer   PASS   PASS     PASS     3/3
tail                           data-layer   PASS   PASS     PASS     3/3
which                          read         PASS   PASS     PASS     3/3
workflow                       read         PASS   PASS     PASS     3/3

Data Pipeline: PASS: sync completed (table validation skipped — sql command unavailable)
Pass Rate: 96% (25/26 passed, 0 critical)
Verdict: PASS

=== workflow-verify ===
Workflow Verification: servicetitan-memberships-pp-cli
================================

Overall Verdict: workflow-pass
  - no workflow manifest found, skipping

=== verify-skill ===
=== servicetitan-memberships-pp-cli ===
  ✓ All checks passed (flag-names, flag-commands, positional-args, unknown-command)
  ✓ canonical-sections passed

=== validate-narrative ===
OK: 10 narrative commands resolved and full examples passed

=== scorecard ===
Quality Scorecard: servicetitan-memberships

  Output Modes         10/10
  Auth                 10/10
  Error Handling       10/10
  Terminal UX          9/10
  README               8/10
  Doctor               10/10
  Agent Native         10/10
  MCP Quality          8/10
  MCP Desc Quality     N/A
  MCP Token Efficiency 0/10
  MCP Remote Transport 10/10
  MCP Tool Design      10/10
  MCP Surface Strategy 10/10
  Local Cache          10/10
  Cache Freshness      5/10
  Breadth              10/10
  Vision               8/10
  Workflows            10/10
  Insight              10/10
  Agent Workflow       9/10

  Domain Correctness
  Path Validity           10/10
  Auth Protocol           9/10
  Data Pipeline Integrity 7/10
  Sync Correctness        10/10
  Live API Verification   N/A
  Type Fidelity           3/5
  Dead Code               5/5

  Total: 87/100 - Grade A
  Note: omitted from denominator: mcp_description_quality, live_api_verification

Sample Output Probe (live command sample)
  Unable to run: binary "C:\\Users\\pierc\\printing-press\\.runstate\\temp-855456de\\runs\\20260515-072215\\working\\servicetitan-memberships-pp-cli\\servicetitan-memberships-pp-cli.exe" is not executable

Gaps:
  - mcp_token_efficiency scored 0/10 - needs improvement
  - MCP: 30 tools (0 public, 30 auth-required) — readiness: full

Shipcheck Summary
=================
  LEG               RESULT  EXIT      ELAPSED
  dogfood           PASS    0         2.807s
  verify            PASS    0         7.638s
  workflow-verify   PASS    0         51ms
  verify-skill      PASS    0         625ms
  validate-narrative  PASS    0         857ms
  scorecard         PASS    0         214ms

Verdict: PASS (6/6 legs passed)
