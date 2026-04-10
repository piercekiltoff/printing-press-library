# linear-pp-cli Shipcheck Report

## Dogfood
- Path Validity: SKIP (GraphQL schema, no REST paths to validate)
- Auth Protocol: SKIP (spec not provided as OpenAPI)
- Dead Flags: 0 (PASS)
- Dead Functions: 0 (PASS)
- Data Pipeline: PARTIAL - 1 domain table detected (store has 30+ but dogfood detected 1)
- Examples: 2/10 promoted commands have examples (FAIL) - generated promoted commands lack examples
- Novel Features: 6/8 survived - projects burndown and cycles compare deferred

## Verify
- Pass Rate: 97% (59/61 passed, 0 critical)
- Verdict: PASS
- Minor failures: `me` and `similar` (need args to run standalone)

## Workflow Verify
- Verdict: workflow-pass (no workflow manifest)

## Scorecard
- Total: 90/100, Grade A
- All dimensions >= 7/10
- Highlights: Output Modes 10, Auth 10, Agent Native 10, Local Cache 10, Insight 10

## Fixes Applied
1. Fixed duplicate struct fields in types.go (GraphQL parser issue)
2. Added missing usageErr function to helpers.go
3. Replaced generated stale command with custom transcendence version
4. Added 8 new commands: today, stale, bottleneck, similar, workload, velocity, me, sql

## Final Ship Recommendation: ship-with-gaps
- Gaps: 2 deferred transcendence features, promoted command examples sparse
- Core functionality is solid: sync, search, all transcendence features work
- Score of 90/100 exceeds 65 threshold significantly
