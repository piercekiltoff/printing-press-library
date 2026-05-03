# Paperclip CLI — Absorb Manifest

## Absorbed (match or beat everything that exists)

### From MCP Server (paperclipai/paperclip — packages/mcp-server)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 1 | Get current agent identity | MCP paperclipMe | `me` command | + --json, agent-native |
| 2 | Get inbox-lite | MCP paperclipInboxLite | `issues inbox` | + --json, filter by status |
| 3 | List agents in company | MCP paperclipListAgents | `agents list` | + --json, --status filter, table view |
| 4 | Get a single agent | MCP paperclipGetAgent | `agents get <id>` | + --json, config view |
| 5 | List issues with filters | MCP paperclipListIssues | `issues list` | + --json, --status, --project, --agent, --label, --q |
| 6 | Get issue | MCP paperclipGetIssue | `issues get <id>` | + by identifier (e.g. TRA-42) |
| 7 | Get heartbeat context | MCP paperclipGetHeartbeatContext | `issues heartbeat-context <id>` | + --json |
| 8 | List issue comments | MCP paperclipListComments | `issues comments list <id>` | + --after, --limit, --order |
| 9 | Get single comment | MCP paperclipGetComment | `issues comments get <id> <commentId>` | + --json |
| 10 | List issue approvals | MCP paperclipListIssueApprovals | `issues approvals list <id>` | + --json |
| 11 | List issue documents | MCP paperclipListDocuments | `issues documents list <id>` | + --json |
| 12 | Get issue document | MCP paperclipGetDocument | `issues documents get <id> <key>` | + render markdown |
| 13 | List document revisions | MCP paperclipListDocumentRevisions | `issues documents revisions <id> <key>` | + --json |
| 14 | List projects | MCP paperclipListProjects | `projects list` | + --json |
| 15 | Get project | MCP paperclipGetProject | `projects get <id>` | + --json |
| 16 | Get issue workspace runtime | MCP paperclipGetIssueWorkspaceRuntime | `issues workspace <id>` | + --json |
| 17 | Control workspace services | MCP paperclipControlIssueWorkspaceServices | `issues workspace-service <id> start/stop/restart` | + --service-name |
| 18 | Wait for workspace service | MCP paperclipWaitForIssueWorkspaceService | `issues workspace-wait <id>` | + --timeout, --service-name |
| 19 | List goals | MCP paperclipListGoals | `goals list` | + --json |
| 20 | Get goal | MCP paperclipGetGoal | `goals get <id>` | + --json |
| 21 | List company approvals | MCP paperclipListApprovals | `approvals list` | + --status filter, --json |
| 22 | Create approval | MCP paperclipCreateApproval | `approvals create` | + --issue-id linking, --dry-run |
| 23 | Get approval | MCP paperclipGetApproval | `approvals get <id>` | + --json |
| 24 | Get approval linked issues | MCP paperclipGetApprovalIssues | `approvals issues <id>` | + --json |
| 25 | List approval comments | MCP paperclipListApprovalComments | `approvals comments <id>` | + --json |
| 26 | Create issue | MCP paperclipCreateIssue | `issues create` | + --stdin JSON, --dry-run |
| 27 | Update issue | MCP paperclipUpdateIssue | `issues update <id>` | + --dry-run |
| 28 | Checkout issue | MCP paperclipCheckoutIssue | `issues checkout <id>` | + auto-agent from context |
| 29 | Release issue | MCP paperclipReleaseIssue | `issues release <id>` | + --dry-run |
| 30 | Add comment | MCP paperclipAddComment | `issues comment <id> <body>` | + --stdin, markdown |
| 31 | Suggest tasks interaction | MCP paperclipSuggestTasks | `issues interact suggest-tasks <id>` | + --json |
| 32 | Ask user questions interaction | MCP paperclipAskUserQuestions | `issues interact ask-questions <id>` | + --json |
| 33 | Request confirmation interaction | MCP paperclipRequestConfirmation | `issues interact request-confirm <id>` | + --json |
| 34 | Upsert issue document | MCP paperclipUpsertIssueDocument | `issues documents write <id> <key>` | + --stdin, --format |
| 35 | Restore document revision | MCP paperclipRestoreIssueDocumentRevision | `issues documents restore <id> <key> <revId>` | |
| 36 | Link approval to issue | MCP paperclipLinkIssueApproval | `issues approvals link <id> <approvalId>` | |
| 37 | Unlink approval from issue | MCP paperclipUnlinkIssueApproval | `issues approvals unlink <id> <approvalId>` | |
| 38 | Approval decision (approve/reject/revise/resubmit) | MCP paperclipApprovalDecision | `approvals decide <id> <action>` | + --note |
| 39 | Add approval comment | MCP paperclipAddApprovalComment | `approvals comment <id> <body>` | |
| 40 | Generic API request escape hatch | MCP paperclipApiRequest | `api <method> <path> [body]` | + --json-body-file |

### From existing TypeScript CLI (paperclipai/paperclip — cli/)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 41 | Auth login (CLI challenge flow) | TypeScript CLI auth login | `auth login` | Same challenge flow, browser-launch |
| 42 | Auth logout | TypeScript CLI auth logout | `auth logout` | Clears stored credential |
| 43 | Auth whoami | TypeScript CLI auth whoami | `auth whoami` | + --json |
| 44 | Context profiles (named base URLs) | TypeScript CLI context | `context show/list/use/set` | Multi-server support |
| 45 | Company list/get | TypeScript CLI company | `companies list/get` | + --json, table view |
| 46 | Dashboard get | TypeScript CLI dashboard | `dashboard` | + --json |
| 47 | Activity log list | TypeScript CLI activity | `activity list` | + --json, --limit |
| 48 | Plugin list/install/uninstall/enable/disable | TypeScript CLI plugin | `plugins list/install/uninstall/enable/disable` | + --json |
| 49 | Feedback list/export | TypeScript CLI feedback | `feedback list/export` | + --json |
| 50 | Doctor | TypeScript CLI doctor | `doctor` | + --json |

### Additional from REST API (not in MCP or TypeScript CLI)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 51 | Agent pause/resume/terminate/wakeup | REST API | `agents pause/resume/terminate/wakeup <id>` | + --dry-run |
| 52 | Agent runtime state | REST API | `agents runtime-state <id>` | + --json |
| 53 | Agent config revisions list/get/rollback | REST API | `agents revisions list/get/rollback <id>` | + --json |
| 54 | Agent budget update | REST API | `agents budget set <id>` | |
| 55 | Agent API keys list/create/delete | REST API | `agents keys list/create/delete <id>` | |
| 56 | Agent skills list/sync | REST API | `agents skills list/sync <id>` | |
| 57 | Agent task sessions | REST API | `agents task-sessions <id>` | + --json |
| 58 | Cost summary | REST API | `costs summary` | + --json |
| 59 | Cost by-agent | REST API | `costs by-agent` | + --json, table |
| 60 | Cost by-project | REST API | `costs by-project` | + --json |
| 61 | Cost by-provider | REST API | `costs by-provider` | + --json |
| 62 | Cost by-biller | REST API | `costs by-biller` | + --json |
| 63 | Budget overview | REST API | `costs budget` | + --json |
| 64 | Budget policies create | REST API | `costs policy create` | + --dry-run |
| 65 | Quota windows | REST API | `costs quota-windows` | |
| 66 | Window spend | REST API | `costs window-spend` | |
| 67 | Routines list | REST API | `routines list` | + --json |
| 68 | Routines get | REST API | `routines get <id>` | + --json |
| 69 | Routines run | REST API | `routines run <id>` | + --json |
| 70 | Routine runs list | REST API | `routines runs <id>` | + --json |
| 71 | Routine triggers create/list | REST API | `routines triggers <id>` | |
| 72 | Routine trigger update/delete/rotate-secret | REST API | `routines trigger update/delete/rotate <id>` | |
| 73 | Secrets list/create/update/delete/rotate | REST API | `secrets list/create/update/delete/rotate` | |
| 74 | Company members list | REST API | `members list` | + --json |
| 75 | Company invites list/create/revoke | REST API | `invites list/create/revoke` | |
| 76 | Adapters list/enable/disable/reload | REST API | `adapters list/enable/reload` | |
| 77 | Skills list/get/create/delete | REST API | `skills list/get/create/delete` | |
| 78 | Heartbeat runs list/get/cancel/log | REST API | `heartbeat-runs list/get/cancel/log` | + --json |
| 79 | Issue runs list | REST API | `issues runs list <id>` | + --json |
| 80 | Issue work products list/create | REST API | `issues work-products list/create <id>` | |
| 81 | Issue tree-holds list/create/release | REST API | `issues tree-holds <id>` | |
| 82 | Execution workspaces list | REST API | `workspaces list` | + --json |
| 83 | Environments list/get/create | REST API | `environments list/get/create` | |
| 84 | Instance settings get/update | REST API | `instance settings get/set` | + --json |
| 85 | Health check | REST API | `doctor` (extends existing) | live endpoint probe |

## Transcendence (only possible with our approach)

| # | Feature | Command | Why Only We Can Do This |
|---|---------|---------|------------------------|
| 1 | Agent fleet dashboard | `fleet` | Aggregates live status, costs, active issues, and idle time across all agents in a company — no single API call provides this |
| 2 | Cost anomaly detection | `costs anomalies` | Requires comparing current spending vs rolling 7/30-day average per agent — only possible with locally-cached cost history |
| 3 | Pending approval queue | `approvals queue` | Cross-joins pending approvals with their linked issues, assignee agents, and wait time — not exposed in any single endpoint |
| 4 | Agent activity timeline | `agents timeline <id>` | Reconstructs run history from heartbeat-runs + task-sessions + comments into a human-readable chronological view |
| 5 | Stale issue finder | `issues stale` | Finds issues in in-progress state with no activity in N days — requires cross-filtering runs + comments by timestamp |
| 6 | Cost drilldown by issue | `costs drilldown <issueId>` | Correlates cost events with heartbeat runs linked to a specific issue — multi-step join no single endpoint does |
| 7 | Routine health monitor | `routines health` | Cross-checks all routine runs for consecutive failures, overdue schedules, and error rates |
