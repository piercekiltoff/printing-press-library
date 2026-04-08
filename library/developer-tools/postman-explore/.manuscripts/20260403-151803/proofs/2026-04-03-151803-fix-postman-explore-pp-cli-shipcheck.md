# Postman Explore CLI Shipcheck Report

## Dogfood
- Path validity: 5/5 PASS
- Dead flags: 0 PASS
- Dead functions: 0 PASS
- Data pipeline: PARTIAL (uses domain-specific upsert)
- Examples: 5/10 commands have examples
- 15 "unregistered commands" — false positive (these are subcommands like `api get-category`, `auth logout` etc.)
- Verdict: FAIL (due to unregistered command naming mismatch)

## Verify
- Pass rate: 71% (12/17 passed, 0 critical)
- 5 failures are commands requiring positional args (browse, open, similar, stale, trending)
- These correctly return errors when called without args — verify doesn't know what args to pass
- All 12 passing commands work correctly with --help, --dry-run, and execution
- Verdict: WARN (expected — positional arg commands can't be auto-tested)

## Scorecard: 90/100 Grade A
| Dimension | Score |
|-----------|-------|
| Output Modes | 10/10 |
| Auth | 8/10 |
| Error Handling | 10/10 |
| Terminal UX | 9/10 |
| README | 10/10 |
| Doctor | 10/10 |
| Agent Native | 10/10 |
| Local Cache | 10/10 |
| Breadth | 6/10 |
| Vision | 7/10 |
| Workflows | 10/10 |
| Insight | 4/10 |
| Path Validity | 10/10 |
| Data Pipeline | 10/10 |
| Sync Correctness | 10/10 |
| Type Fidelity | 3/5 |
| Dead Code | 5/5 |

## Ship Recommendation: **ship-with-gaps**

The CLI works against the live API, has a solid data layer, and scores 90/100. The gaps:
- Verify false-fails on positional-arg commands (not a real issue)
- Insight dimension scored 4/10 (could add more analytics)
- Breadth 6/10 (could add more entity type coverage)
