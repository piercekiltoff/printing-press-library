# Slack CLI Absorb Manifest

## Sources Analyzed
1. **korotovsky/slack-mcp-server** (TypeScript) - 15 tools, browser token auth, unreads, usergroups
2. **piekstra/slack-mcp-server** (TypeScript) - 22 tools, Block Kit, channel management, files
3. **rockymadden/slack-cli** (Bash) - Chat, files, status, reminders, DND, pipe-friendly
4. **shaharia-lab/slackcli** (TypeScript/Bun) - AI-friendly, JSON/table/text, canvas, browser auth
5. **lox/slack-cli** (Go) - Search, channel read, thread read, multi-workspace, Amp skill
6. **vincentkoc/slacrawl** (Go) - SQLite + FTS5, sync, tail, watch, mentions, desktop ingest
7. **slack-digest skill** (Claude) - Channel digest, action item extraction, routing

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Send message to channel | rockymadden, piekstra MCP | `messages send --channel #general "hello"` | --dry-run, --json, --thread, --blocks, stdin pipe |
| 2 | Send DM | korotovsky MCP | `messages send --user @alice "hey"` | Auto-resolve user by name/email, --dry-run |
| 3 | Read channel history | korotovsky MCP, lox, slacrawl | `channels read #general --limit 50` | Offline from SQLite after sync, --since, --until |
| 4 | Read thread replies | korotovsky MCP, lox, shaharia-lab | `threads read <permalink>` | Accept permalink or channel+ts, offline after sync |
| 5 | Search messages | korotovsky MCP, lox | `search "deploy failed" --from @alice --in #ops` | FTS5 offline search + API search, regex support |
| 6 | List channels | all tools | `channels list --type public,private` | Offline, --json, --select, --csv, FTS on names |
| 7 | Channel info | piekstra MCP, lox | `channels info #general` | Member count, topic, purpose, created, archived status |
| 8 | Create channel | piekstra MCP | `channels create --name ops-alerts --private` | --dry-run, --topic, --purpose |
| 9 | Archive/unarchive channel | piekstra MCP | `channels archive #old-project` | --dry-run, bulk via --stdin |
| 10 | Set channel topic/purpose | piekstra MCP | `channels topic #general "New topic"` | --dry-run |
| 11 | Invite user to channel | piekstra MCP | `channels invite #general @alice` | Bulk invite via --users |
| 12 | List users | all tools | `users list --active` | Offline, --json, filterable by status/role |
| 13 | User info/lookup | lox, piekstra MCP | `users info @alice` or `users info alice@company.com` | Email lookup, presence, profile fields |
| 14 | File upload | rockymadden, piekstra MCP | `files upload ./report.pdf --channel #reports` | Stdin pipe, --title, --comment |
| 15 | File list/info | rockymadden | `files list --user @me --type pdf` | Offline after sync, filterable |
| 16 | Add/remove reaction | rockymadden, korotovsky, piekstra | `reactions add :thumbsup: <permalink>` | Accept permalink, --dry-run |
| 17 | Set status/presence | rockymadden | `status set "In meeting" :calendar:` | --expiration, presets |
| 18 | DND/snooze | rockymadden | `dnd start --minutes 60` | Info, end commands |
| 19 | Reminders | rockymadden | `reminders add "standup" --at "9am tomorrow"` | Natural language time, list/complete/delete |
| 20 | Usergroup management | korotovsky MCP | `usergroups list`, `usergroups create --name team-leads` | Members list/update, --dry-run |
| 21 | Unread counts | korotovsky MCP | `unreads` | Priority-sorted, per-channel counts |
| 22 | Mark channel read | korotovsky MCP | `channels mark #general` | Mark all with --all |
| 23 | Multi-workspace | shaharia-lab, lox | `--workspace company` flag | Workspace list, set-default |
| 24 | JSON/table/text output | shaharia-lab | `--json`, `--csv`, `--select fields` | All commands, agent-native |
| 25 | Browser token auth | korotovsky, shaharia-lab | `auth login --browser` | xoxc-/xoxd- session tokens |
| 26 | Emoji list | Slack API | `emoji list` | Custom emoji with aliases |
| 27 | Pins | Slack API | `pins list #general`, `pins add <permalink>` | --dry-run |
| 28 | Stars/bookmarks | Slack API | `stars list`, `stars add <permalink>` | Offline after sync |
| 29 | Team info | piekstra MCP, Slack API | `team info` | Access logs, billable info |
| 30 | Canvas read | shaharia-lab | `canvas read <url>` | Multiple formats |
| 31 | Scheduled messages | Slack API | `messages schedule --channel #general --at "9am" "reminder"` | List, delete scheduled |
| 32 | Update/delete message | rockymadden, piekstra MCP | `messages update <permalink> "fixed text"` | --dry-run |
| 33 | Sync to SQLite | slacrawl | `sync --full` | Incremental by default, channel-level cursors |
| 34 | FTS5 offline search | slacrawl | `search --offline "deploy"` | Regex, SQL-composable |
| 35 | SQL queries | slacrawl | `sql "SELECT * FROM messages WHERE ..."` | Read-only, any query |
| 36 | Live tail (Socket Mode) | slacrawl | `tail --channels #ops,#alerts` | Real-time stream, --json |
| 37 | Doctor diagnostics | slacrawl | `doctor` | Token validity, scopes, connectivity |
| 38 | Mention extraction | slacrawl | `mentions --user @me` | Per-channel mention tracking |

### Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 1 | Channel health report | `health` | Requires local join across messages + users + channels: messages/day, response time, active posters, quiet channels | 9/10 | Domain archetype: communication analytics. korotovsky MCP has unreads but no cross-entity health metrics |
| 2 | Response time analytics | `response-times --channel #support` | Requires correlating message timestamps with reply timestamps across threads in SQLite | 8/10 | Communication domain core metric. No competitor tracks this. Pain point: "how fast does my team respond?" |
| 3 | Daily/weekly digest | `digest --period today` | Requires local message archive + mention extraction + thread summarization across all channels | 8/10 | slack-digest Claude skill does this but only for current session. Our version works offline from synced data |
| 4 | Thread staleness radar | `threads stale --days 3` | Requires scanning all thread_ts in messages table for threads with no recent reply | 8/10 | User pain: threads go unanswered. No tool detects this. Requires full message history in SQLite |
| 5 | Who-talks-to-whom graph | `network --channel #engineering` | Requires join across messages + threads + reactions to build interaction adjacency | 7/10 | Team ops use case. Only possible with full message corpus in SQLite |
| 6 | Channel activity trends | `trends --channel #general --weeks 4` | Requires historical message counts grouped by day/week from SQLite | 7/10 | slacrawl has data but no analytics commands. Domain-specific time series |
| 7 | Quiet channel detector | `channels quiet --days 14` | Requires comparing last message timestamp across all synced channels | 7/10 | Admin ops: find channels to archive. Requires full channel + message sync |
| 8 | User activity summary | `users activity @alice --days 30` | Requires join across messages + reactions + channels per user from local store | 7/10 | Team ops: understand who's active where. No competitor offers per-user cross-channel analytics |


## User-Requested Features (from brainstorm)

### Additional Absorbed
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 39 | Channel-based alert filtering | User request | `alerts --channel #deploys --since today` | Filter CI/CD/monitoring alerts by type, status, severity from synced messages |
| 40 | Cross-workspace search | User request | `search "query" --workspace all` | Unified FTS5 across multiple synced workspaces |
| 41 | Message export/backup | User request | `export --channel #general --format json,csv,markdown` | Full channel export with threads, files, reactions |
| 42 | Daemon/tail mode (Discrawl-style) | User request + slacrawl tail | `daemon --channels #ops,#alerts --store` | Background process that continuously syncs to SQLite, like Discrawl for Slack |

### Additional Transcendence
| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 9 | Funny digest | `digest --mode funny --period week` | Requires reaction-weighted message scoring (most :joy: :rofl: reacted messages) across all public channels in SQLite | 9/10 | User's killer feature idea. Unique - no tool does humor curation. Reactions + messages join |
| 10 | Discrawl-style daemon | `daemon` | Continuous background sync to SQLite with Socket Mode, channel-level cursors, automatic backfill | 8/10 | User request inspired by Peter Steinberger's Discrawl. slacrawl has `tail` but not persistent daemon mode |
| 11 | Smart notification filter | `notifications --priority high` | Requires classifying messages by urgency using mention patterns, reaction velocity, and thread activity from SQLite | 7/10 | User pain: too many notifications. Requires cross-entity scoring only possible with local data |
| 12 | Archive search (free tier saver) | `search --archive "old query"` | Full-text search across locally archived messages that Slack's free tier no longer shows | 9/10 | User pain: can't find old messages. Free tier limits search. Our SQLite archive is permanent |

