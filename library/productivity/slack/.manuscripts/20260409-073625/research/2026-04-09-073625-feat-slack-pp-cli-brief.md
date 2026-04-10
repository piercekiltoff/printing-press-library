# Slack CLI Brief

## API Identity
- Domain: Team communication & collaboration
- Users: Developers, ops teams, workspace admins, power users who live in terminal
- Data profile: Messages (high volume, threaded), channels, users, files, reactions, reminders, usergroups. Rich metadata (timestamps, thread_ts, permalinks). 174 endpoints across 25 categories in the official OpenAPI 2.0 spec.

## Reachability Risk
- None. Slack Web API is well-maintained, officially documented, has an OpenAPI 2.0 spec at github.com/slackapi/slack-api-specs. OAuth 2.0 with bot tokens (xoxb-) and user tokens (xoxp-). No reports of blocked access.

## Auth Model
- **Bot Token (xoxb-)**: For monitoring, channel reading, message sending. Must be invited to channels. Created via Slack App at api.slack.com/apps.
- **User Token (xoxp-)**: For personal use, search, admin operations. Sees everything the user can see.
- **Browser Token (xoxc-/xoxd-)**: Session tokens from browser, used by korotovsky MCP for stealth mode.
- Auth header: `Authorization: Bearer <token>`
- Env vars: `SLACK_BOT_TOKEN`, `SLACK_USER_TOKEN`

## Top Workflows
1. **Channel monitoring** - tail a channel in real-time, get unread counts, digest mode (like Discrawl)
2. **Message search** - find messages across workspace by text, user, channel, date range
3. **Send messages** - quick sends to channels/DMs/threads without opening Slack
4. **Thread management** - read and reply to threads from terminal
5. **Team ops** - manage channels, usergroups, user lookup, presence/DND management

## Table Stakes (from competitor analysis)
- Send/read/search messages (rockymadden, shaharia-lab, lox, all MCPs)
- Thread support (shaharia-lab, lox, korotovsky MCP)
- Channel list/info/create/archive (piekstra MCP, rockymadden)
- User list/info/lookup (all tools)
- File upload (rockymadden, piekstra MCP)
- Reactions add/remove (rockymadden, korotovsky, piekstra)
- Status/presence management (rockymadden)
- Reminders (rockymadden)
- DND/snooze (rockymadden)
- Multi-workspace support (shaharia-lab, lox)
- JSON/table/text output (shaharia-lab)
- Pipe-friendly (rockymadden)
- Browser token auth (korotovsky, shaharia-lab)
- Unread counts (korotovsky)
- Usergroup management (korotovsky)
- Block Kit support (piekstra)
- Canvas read (shaharia-lab)

## Data Layer
- Primary entities: messages, channels, users, threads, reactions, files, usergroups, reminders
- Sync cursor: conversations.history cursor-based pagination, channel-level latest_ts tracking
- FTS/search: FTS5 on message text, channel names, user display names
- Historical: message archive with thread backfill enables offline search and analytics

## User Vision
- Two auth modes: bot token for Discrawl-style monitoring, user token for personal/admin use
- All three feature categories: power user workflows, automation/CI, team ops
- User owns their company's Slack workspace (admin access)
- Wants to understand and use both token types

## Product Thesis
- Name: slack-pp-cli
- Why it should exist: No CLI covers the full Slack surface. The official `slack` CLI is for app development. slacrawl has SQLite but limited commands. rockymadden is bash-only with no data layer. lox/slack-cli is read-only. shaharia-lab is TypeScript. The MCP servers cover messaging but not the full API. Nobody has offline search + channel monitoring + admin ops + agent-native output in one tool.

## Build Priorities
1. Data layer for messages, channels, users, threads, reactions, files, usergroups
2. Full sync with cursor-based pagination and incremental updates
3. All absorbed features from every competitor
4. Transcendence: cross-entity analytics only possible with local SQLite
5. Dual auth (bot + user token) with doctor diagnostics
