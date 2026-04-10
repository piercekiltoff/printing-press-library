# Linear CLI Brief

## API Identity
- Domain: Project management / issue tracking for engineering teams
- Users: Software engineers, engineering managers, product managers, DevOps
- Data profile: GraphQL API at https://api.linear.app/graphql, 98+ entity types, 43k-line schema, cursor-based pagination via Connection pattern. Auth via API key (Authorization header) or OAuth2.

## Reachability Risk
- None. Linear's API is well-maintained, official, and actively developed. No blocking issues found.

## Top Workflows
1. Issue triage and sprint planning - review inbox, assign priorities, move issues to cycles
2. Sprint execution - view my assigned issues, update status, track cycle progress and burndown
3. Project health monitoring - project status, milestone tracking, team velocity over time
4. PR/branch workflow - create branch from issue, link PRs, track development progress
5. Backlog grooming - find stale issues, detect duplicates, prioritize by impact

## Table Stakes
- Full issue CRUD with filtering, sorting, assignment
- Project and cycle management
- Team and user listing
- Label and workflow state management
- Comment creation and listing
- Document management
- Notification handling
- Git branch integration (create branch from issue ID)
- Search across issues
- --json output for all commands
- Watch mode for real-time changes

## Data Layer
- Primary entities: Issues, Projects, Cycles, Teams, Users, Labels, WorkflowStates, Comments, Documents, Milestones, Initiatives
- Sync cursor: updatedAt-based incremental sync with cursor pagination
- FTS/search: Issue titles, descriptions, comments via SQLite FTS5
- High-gravity fields: issue identifier (ABC-123), title, state, priority, assignee, cycle, project, labels, due date, estimate

## Product Thesis
- Name: linear-pp-cli
- Why it should exist: No existing Linear CLI combines comprehensive API coverage with an offline SQLite data layer. Finesssee/linear-cli (Rust, 60+ commands) is the most complete but lacks offline search, historical velocity data, and compound analytics. linearis is agent-optimized but covers limited surface. Our CLI absorbs every feature from every competitor, adds SQLite persistence, and enables transcendence features (bottleneck detection, velocity trends, duplicate detection, stale issue hunting) that only work when all data lives locally.

## Build Priorities
1. Foundation: SQLite store for issues, projects, cycles, teams, users, labels, states + sync + FTS5
2. Absorb: Every command from Finesssee/linear-cli + schpet/linear-cli + linearis + official MCP
3. Transcend: Compound analytics that require local data joins (velocity, bottlenecks, duplicates, staleness)
