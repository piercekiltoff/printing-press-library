# x-twitter-pp-cli Shipcheck Report

## Verdict: SHIP (91/100, Grade A)

## Shipcheck legs (all PASS)
| Leg | Result | Notes |
|---|---|---|
| dogfood | PASS | 14/14 novel features survived; 0 dead code; 4/4 paths valid; data pipeline domain helpers wired |
| verify | PASS | 100% (22/22 commands), 0 critical |
| workflow-verify | PASS | No manifest required |
| verify-skill | PASS | All flag-name/command/positional checks pass |
| scorecard | PASS | 91/100 Grade A; 14/14 live-check |

## Score progression
- Baseline: 86/100
- Bug fix pass + README polish: 87/100
- Codex delegation (cache freshness scaffolding): 89/100
- Typed Upsert/Search methods (data pipeline): **91/100**

## Gaps remaining (acceptable)
- MCP Token Efficiency: 7/10 — already strong, structural cap
- MCP Remote Transport: 5/10 — structural (CLI is local-first)
- MCP Tool Design: 5/10 — generated descriptions are functional, agent-grade
- Auth Protocol: 8/10 — cookie-auth scoring caps via spec inference
- Type Fidelity: 3/5 — would require adding 2+ MarkFlagRequired (breaks verify-friendly RunE)

## Codex delegation summary
- 1 task delegated, 1 successful (cache_freshness scaffolding: 4 files)
- 0 consecutive failures, no fallback to direct mode
- Files written by Codex:
  - `internal/cliutil/freshness.go` — EnsureFresh, FormatAge
  - `internal/cli/auto_refresh.go` — autoRefreshIfStale hook
  - `internal/share/share.go` — Bundle export/import
  - `internal/cli/share_commands.go` — share export/import cobra commands
  - `internal/cli/root.go` — registration of newShareCmd

## Auth context
- Auth method: cookie-based (auth_token + ct0 + guest_id), captured via `auth login --chrome` from a logged-in Chrome session
- No paid X API tier required
- Bearer token: hardcoded default (X public web-client token), overridable via X_TWITTER_BEARER_TOKEN

## Novel features built (14/14)
All approved features from the absorb manifest are implemented:
1. relationships not-following-back
2. relationships mutuals
3. relationships unfollowed-me
4. relationships ghost-followers
5. audit inactive
6. relationships fans
7. relationships overlap
8. audit suspicious-followers
9. relationships new-followers
10. tweets engagement
11. whois
12. search saved
13. archive import
14. export jsonl

## Browser-Sniff gate
- Decision: skip-silent (spec from fa0311/twitter-openapi was complete: 39 endpoints, GraphQL surface)
- Marker: `browser-browser-sniff-gate.json` (1 entry for x-twitter source)

## Final ship recommendation: ship
