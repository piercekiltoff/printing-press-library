# HubSpot CLI Shipcheck

## Verify
- Pass Rate: 100% (24/24 commands)
- All commands PASS on help, dry-run, and exec
- Verdict: PASS

## Scorecard
- Total: 93/100 - Grade A
- Output Modes: 10/10
- Auth: 10/10
- Error Handling: 10/10
- Terminal UX: 9/10
- README: 10/10
- Doctor: 10/10
- Agent Native: 10/10
- Local Cache: 10/10
- Breadth: 8/10
- Vision: 9/10
- Workflows: 10/10
- Insight: 8/10
- Data Pipeline Integrity: 10/10
- Sync Correctness: 10/10

## Workflow Verify
- Verdict: workflow-pass (no manifest, skipping)

## Top Blockers Found
- None. Clean pass across all checks.

## Fixes Applied
- Fixed duplicate JSON tag on config.go AccessToken field
- Fixed missing "strings" import in deals_velocity.go
- Fixed type mismatch between pipelineData and loadVelocityPipelines signature

## Final Verdict: ship
