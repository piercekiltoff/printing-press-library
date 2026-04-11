# Hacker News CLI Absorb Manifest

## Sources Analyzed
1. **circumflex** (Go, TUI) — story browsing, comment reading, reader mode, favorites, vim nav
2. **haxor-news** (Python) — story lists, search, who's hiring regex, comment filtering, user profiles
3. **hnterminal** (Python) — browsing, login, upvote, commenting
4. **mcp-hacker-news** (TypeScript MCP) — 11 tools: top/best/new/ask/show/job stories, item, user, comments, maxitem, updates
5. **hackernews-mcp** (Python MCP) — article content extraction + discussions, LLM-optimized
6. **hn-mcp** (MCP) — Firebase + Algolia search integration
7. **hackernews-api** (npm) — Firebase API wrapper
8. **hacker-news-api** (npm) — Algolia API wrapper
9. **hnclient** (PyPI) — cached API calls

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Top stories | All CLIs/MCPs | `hn top` | --limit, --json, --since, shows points+comments inline |
| 2 | New stories | All CLIs/MCPs | `hn new` | Same flags |
| 3 | Best stories | circumflex, MCPs | `hn best` | Same flags |
| 4 | Ask HN | haxor-news, MCPs | `hn ask` | Same flags |
| 5 | Show HN | haxor-news, MCPs | `hn show` | Same flags |
| 6 | Job stories | haxor-news, MCPs | `hn jobs` | Same flags |
| 7 | View item details | All | `hn item <id>` | Full details with comment count, score, time ago |
| 8 | Read comments | circumflex, haxor-news | `hn comments <id>` | Threaded display, --flat, --depth, --json |
| 9 | Search stories | haxor-news (Algolia) | `hn search <query>` | Full Algolia params: --tags, --date-range, --sort, --json |
| 10 | User profile | haxor-news, MCPs | `hn user <username>` | Karma, about, submission history, --json |
| 11 | Who's Hiring | haxor-news | `hn hiring <regex>` | Filter monthly threads by tech/location/remote |
| 12 | Freelance thread | haxor-news | `hn freelance <regex>` | Same pattern |
| 13 | Open in browser | haxor-news | `hn open <id>` | Opens story URL or HN thread |
| 14 | Recently updated | MCP servers | `hn updates` | Changed items and profiles |
| 15 | Favorites/bookmarks | circumflex | `hn bookmark <id>` / `hn bookmarks` | Local SQLite, not tied to HN account |
| 16 | Comment filtering | haxor-news | --author, --since, --match flags on comments | Regex + time + author filters |

## Transcendence (only possible with our approach)

| # | Feature | Command | Score | Why Only We Can Do This |
|---|---------|---------|-------|------------------------|
| 1 | Front page diff | `hn diff` | 9/10 | Show what changed on the front page since you last checked. Stories that appeared, disappeared, or moved significantly. Requires local snapshot of previous front page state in SQLite. No existing tool tracks front page changes over time. |
| 2 | Thread summarizer | `hn tldr <id>` | 9/10 | Condense a 500-comment thread into the key arguments, consensus, and dissenting views. Fetches full comment tree, groups by sentiment/topic. No existing CLI does comment analysis. |
| 3 | Submission tracker | `hn my <username>` | 8/10 | Track YOUR submissions: which got traction (>10 points), which died, average score, best time to post. Requires syncing submission history + computing stats. |
| 4 | Topic pulse | `hn pulse <topic>` | 8/10 | "What's HN saying about Rust this week?" Search by date range, aggregate points/comments/frequency. Shows trending topics with velocity. Requires Algolia date search + aggregation. |
| 5 | Who's Hiring analytics | `hn hiring-stats` | 8/10 | Aggregate Who's Hiring data across months: most requested languages, remote %, salary mentions, company frequency. Requires syncing multiple months of hiring threads. |
| 6 | Story velocity | `hn velocity <id>` | 7/10 | Track how fast a story is climbing: points/hour, comments/hour, rank trajectory. Requires polling and local time-series storage. |
| 7 | User karma graph | `hn karma <username>` | 7/10 | Show karma earned over time by analyzing submission scores. Local computation across synced submissions. |
| 8 | Dead thread detector | `hn controversial` | 7/10 | Find stories with high comment-to-point ratios — the controversial discussions. Requires sorting synced stories by comments/points ratio. |
| 9 | Repost finder | `hn repost <url>` | 6/10 | "Has this URL been posted before?" Search Algolia for the URL, show all previous submissions with scores. Useful before posting. |
| 10 | Weekend vs weekday | `hn timing` | 6/10 | Best time to post on HN based on historical data: day of week × time of day × average score. Requires bulk date analysis via Algolia. |
