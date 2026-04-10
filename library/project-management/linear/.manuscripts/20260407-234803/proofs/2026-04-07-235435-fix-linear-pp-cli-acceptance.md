Acceptance Report: linear-pp-cli
  Level: Full Dogfood
  Tests: 11/13 passed
  Failures:
    - [similar ""]: FTS5 syntax error on empty query (fixed inline)
    - [stale --json empty]: returned null instead of [] (fixed inline)
  Fixes applied: 4
    - Fixed GraphQL TeamsQuery complexity (too many nested fields)
    - Fixed CyclesQuery referencing non-existent completedScopeCount field
    - Fixed FTS5 content-linked trigger approach (was using wrong DELETE syntax)
    - Fixed stale JSON output for empty results / similar empty query handling
  Printing Press issues: 2
    - GraphQL parser creates FTS triggers for all entities even when columns don't exist
    - Promoted commands generated from GraphQL lack examples
  Gate: PASS

  Test Results:
  [1/13] doctor: PASS - auth configured, API reachable
  [2/13] me: PASS - returns viewer info (mvanhorn, Esper Labs)
  [3/13] sync --full: PASS - 1234 items in 12s (1104 issues, 47 projects, 12 users, etc.)
  [4/13] sql count: PASS - 1104 issues counted
  [5/13] similar "onboarding": PASS - 16 results via FTS5
  [6/13] today: PASS - 1 active issue (ESP-1155)
  [7/13] stale --days 60: PASS - shows stale backlog items
  [8/13] bottleneck: PASS - shows 5 overloaded members
  [9/13] workload: PASS - shows 7 members with issue counts
  [10/13] velocity: PASS (no scope data available in this workspace)
  [11/13] --json output: PASS - valid JSON arrays
  [12/13] error path: PASS after fix - "search query cannot be empty"
  [13/13] SQL GROUP BY: PASS - correct state distribution (588 Done, 244 Backlog, etc.)
