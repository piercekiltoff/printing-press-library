# Paperclip CLI Brief

## API Identity
- Domain: AI agent management control plane (self-hosted or cloud)
- Users: Platform operators, engineering leads, product managers managing teams of AI agents
- Data profile: Agents + Issues + Companies + Costs + Routines + Approvals + Plugins + Skills + Secrets

## Reachability Risk
- None — API confirmed reachable at localhost:3100, JSON responses validated, API key verified

## Auth
- Type: Bearer token (`BoardApiKeyAuth`)
- Header: `Authorization: Bearer <token>`
- Env var: `PAPERCLIP_API_KEY`
- Base URL env var: `PAPERCLIP_URL` (default: `http://localhost:3100`)
- Company context: stored persistently (keyed by base URL) so `--company-id` is rarely needed

## Top Workflows
1. **Agent fleet ops** — list running agents, pause/resume/terminate an agent, wakeup an idle agent, check runtime state
2. **Issue management** — list by status/project/agent, create, checkout/release, comment, approve, view documents/work products
3. **Cost monitoring** — spending by agent/project/provider, budget utilization, quota windows
4. **Routine management** — list, run manually, manage triggers (webhook/cron), rotate secrets
5. **Approval workflows** — list pending approvals, approve/reject/request-revision, add comments

## Table Stakes (from MCP server — 40 tools)
- Get current agent identity + inbox
- List/get agents per company
- List/get/create/update issues with rich filters
- Checkout and release issues
- Add comments to issues
- Create approvals, approve/reject/revise/resubmit
- List/get/create/update documents + revisions + restore
- List projects/goals
- Get issue workspace runtime, control runtime services
- Generic API escape hatch

## Table Stakes (from existing TypeScript CLI)
- Auth login/logout/whoami (CLI auth challenge flow)
- Context profiles (multiple server/company contexts)
- Agent list/get + local-cli
- Company list/get
- Issue list/get/create with text match
- Approval list/get/create/approve
- Plugin list/install/uninstall/enable/disable
- Dashboard get
- Activity log list
- Feedback report/export
- Doctor command (server-side health checks)
- Routines management
- Environment operations

## Data Layer
- Primary entities: agents, issues, companies, projects, routines, approvals, goals, plugins, secrets
- Sync cursor: issues by updatedAt, heartbeat-runs by createdAt
- FTS/search: issue text search (title, description, identifier), agent name/status filter
- Key compound queries: costs cross-agent, stale issues by agent+status+date, approval backlog by company

## Codebase Intelligence
- Source: Local repo (private)
- MCP server: `/packages/mcp-server/src/tools.ts` — 40 tools, Agent Bearer auth, JSON responses
- Existing CLI: `/cli/src/` — TypeScript commander.js, already has auth challenge flow (`/api/cli-auth/challenges`)
- The CLI auth challenge flow (`POST /api/cli-auth/challenges` → browser approve → poll) is the correct login mechanism
- Auth key format: `pcp_board_<hex>` for board keys, `pcp_agent_<hex>` for agent keys

## Product Thesis
- Name: `paperclip-pp-cli`
- Why it should exist: The MCP server covers agent-native operations but not the control-plane power-user surface. The TypeScript CLI is for server operators (env setup, doctor, run), not for teams using the platform daily. This CLI covers the daily-driver operator surface: fleet status, cost dashboards, approval queue, issue queue — all in one fast Go binary with `--json`, offline-composable output, and agent-native flags.

## Build Priorities
1. Auth (login/logout/whoami via CLI challenge flow + API key direct)
2. Context management (named profiles, active company)
3. Agents: list, get, pause/resume/terminate/wakeup, runtime-state, config, keys, budget
4. Issues: list, get, create, update, checkout/release, comment, approve, documents, work-products
5. Costs: summary, by-agent, by-project, by-provider, budget overview
6. Approvals: list, get, approve/reject/revise, comments
7. Routines: list, get, run, triggers
8. Companies: list, get, members, dashboard
9. Projects: list, get
10. Skills/Plugins/Secrets: list, manage
