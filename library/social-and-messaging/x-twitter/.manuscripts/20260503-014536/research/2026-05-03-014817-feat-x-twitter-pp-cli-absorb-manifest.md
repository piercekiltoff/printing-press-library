# X (Twitter) CLI Absorb Manifest

## Source Inventory
| Source | Type | License | Stars | Notes |
|---|---|---|---|---|
| fa0311/twitter-openapi | OpenAPI spec | AGPL/custom | 181 | Source of truth for GraphQL operations |
| steipete/birdclaw | TS CLI + web UI | MIT | new | Local-first Twitter workspace; same SQLite-FTS5 pattern |
| d60/twikit | Python library | MIT | 4.4k | Cookie auth, no API key, full GraphQL |
| trevorhobenshield/twitter-api-client | Python library | MIT | 2.5k+ | V1+V2+GraphQL, comprehensive |
| sferik/x-cli ("t") | Ruby CLI | MIT | 7k+ | Original Twitter CLI, V1 (deprecated) |
| Infatoshi/x-cli | Go CLI | MIT | <500 | V2 paid tier only |
| public-clis/twitter-cli | CLI | MIT | <500 | Cookie auth, read-only |
| Rishikant181/Rettiwt-API | TS CLI/lib | MIT | 800+ | Cookie auth, has CLI |
| EnesCinr/twitter-mcp | MCP | MIT | 200+ | V2 paid key |
| Infatoshi/x-mcp | MCP | MIT | 100+ | V2 paid key, 16 tools |
| Circleboom | SaaS | proprietary | n/a | Not-following-back, mutuals, audit |
| Followerwonk | SaaS | proprietary | n/a | Bot detection, audience overlap |
| FollowerAudit | SaaS | proprietary | n/a | Fake/bot/inactive detection |
| Twiangulate | SaaS | proprietary | n/a | Connection mapping |
| jfullstackdev/twitter-x-unfollow-tool | Web | MIT | <500 | Offline ZIP-based not-following-back |

## Absorbed (match or beat everything that exists)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---|---|---|---|
| 1 | Post tweet (text) | sferik/x-cli, twikit, all MCPs | `tweets create --text` (auto-generated from spec) | `--stdin` for piping, `--reply-to`, `--thread`, `--dry-run`, JSON output |
| 2 | Post tweet (media) | twikit | `tweets create --media <path>` | Multi-file upload, progress, dry-run |
| 3 | Post thread | sferik/x-cli | `tweets thread --from-file <md>` | Markdown-driven threads, dry-run |
| 4 | Delete tweet | all | `tweets delete <id>` (generated) | `--dry-run`, idempotent |
| 5 | Search tweets | all | `tweets search <query>` (generated) | `--latest/--top`, `--lang`, `--save` (persists to local), `--limit`, JSON |
| 6 | Get user profile | all | `users get <handle>` (generated) | `--json --select`, cached locally |
| 7 | Get user tweets | all | `users tweets <handle>` (generated) | `--limit`, `--since`, paginates, persists |
| 8 | Get user followers | twikit, MCPs | `users followers <handle>` (generated) | Paginates, persists to local store |
| 9 | Get user following | twikit | `users following <handle>` (generated) | Paginates, persists to local store |
| 10 | Follow user | sferik/x-cli, twikit, MCPs | `users follow <handle>` (generated) | `--dry-run`, idempotent |
| 11 | Unfollow user | all | `users unfollow <handle>` (generated) | `--dry-run`, batch via stdin |
| 12 | Like tweet | all | `tweets like <id>` (generated) | `--dry-run` |
| 13 | Unlike tweet | all | `tweets unlike <id>` (generated) | `--dry-run` |
| 14 | Retweet | all | `tweets retweet <id>` (generated) | `--dry-run` |
| 15 | Unretweet | all | `tweets unretweet <id>` (generated) | `--dry-run` |
| 16 | Bookmark add | all | `bookmarks add <id>` (generated) | `--dry-run` |
| 17 | Bookmark remove | all | `bookmarks remove <id>` (generated) | `--dry-run` |
| 18 | List bookmarks | all | `bookmarks list` (generated) | Paginates, persists locally, `--json` |
| 19 | Send DM | sferik, twikit | `dms send <user> --text` (generated) | `--dry-run`, JSON |
| 20 | List DMs | sferik | `dms list` (generated) | Persists locally, `--json` |
| 21 | List management (create/delete) | sferik | `lists create/delete` (generated) | `--dry-run` |
| 22 | List add/remove member | sferik | `lists members add/remove` (generated) | `--dry-run`, batch |
| 23 | List timeline | sferik | `lists timeline <id>` (generated) | Paginates, persists |
| 24 | Get trends | twikit, sferik | `trends list --woeid` (generated) | Snapshots locally for time-series |
| 25 | Block/unblock | sferik | `users block/unblock` (generated) | `--dry-run` |
| 26 | Mute/unmute | sferik | `users mute/unmute` (generated) | `--dry-run` |
| 27 | Get notifications | spec | `notifications list` (generated) | Persists, `--json` |
| 28 | Mentions timeline | sferik | `users mentions` (generated) | Persists, paginates |
| 29 | Home timeline | sferik | `tweets home` (generated) | Persists, paginates |
| 30 | Get tweet detail | spec | `tweets get <id>` (generated) | `--json --select`, cached |
| 31 | Reply to tweet | spec | `tweets reply <id> --text` (generated) | `--dry-run`, threading |
| 32 | Quote tweet | spec | `tweets quote <id> --text` (generated) | `--dry-run` |
| 33 | Search users | spec | `users search <query>` (generated) | `--limit`, persists |
| 34 | Spaces (read-only) | spec | `spaces get <id>` (generated) | `--json` |
| 35 | Communities (read) | spec | `communities get <id>` (generated) | `--json` |
| 36 | Health check | universal | `doctor` | Tests cookie validity, rate limit headroom, DB integrity |
| 37 | Multi-account auth | sferik | `auth login --chrome --account <name>` | Browser cookie capture, multiple accounts |
| 38 | Agent context | (PP universal) | `agent-context` | Auto-emitted PP feature |
| 39 | Sync (full state refresh) | (none — invented) | `sync followers/following/tweets/bookmarks/lists` | Cursor-aware, resumable, persists to local |
| 40 | Stream search (Streaming API approximation) | sferik (older) | `tweets watch <query>` | Polls + diffs, emits new matches as JSON lines |
| 41 | Multi-transport (cookie + V2 API + auto) | birdclaw | `--transport cookie/api/auto` (config + flag) | Cookie default, V2 fallback when key set, auto picks best per call |
| 42 | Multi-account shared DB | birdclaw, sferik | `auth login --account <name>`, `--account` global flag | One SQLite DB, multiple accounts, account-scoped queries |

### Stubs (explicit)
| # | Feature | Status | Reason |
|---|---|---|---|
| (none) | | | All approved features will be fully implemented |

## Transcendence (only possible with our approach)
| # | Feature | Command | Why Only We Can Do This | Score |
|---|---|---|---|---|
| 1 | **Following but not following back** | `relationships not-following-back` | Requires local JOIN of `following` and `followers` tables. SaaS tools require paid OAuth. Existing offline tools require manual ZIP upload. Only way to get this CLI-native, agent-callable, free. | 10 |
| 2 | **Mutuals (two-way follows)** | `relationships mutuals` | Local INTERSECT on `following` and `followers`. SaaS only. | 9 |
| 3 | **Who unfollowed me (since)** | `relationships unfollowed-me --since 7d` | Requires snapshot history — only possible because we persist `follow_snapshots` over time. SaaS does this only for paying users. | 9 |
| 4 | **Ghost followers (inactive)** | `relationships ghost-followers --days 90` | Requires `last_tweet_at` per follower, populated by syncing each user's recent tweet. Local SQL filter. SaaS does this for $$$. | 8 |
| 5 | **Audit inactive accounts I follow** | `audit inactive --days 90` | Same data layer as #4 but inverse direction. | 8 |
| 6 | **Fans (they follow me, I don't follow back)** | `relationships fans` | Inverse of #1. Local SQL. | 8 |
| 7 | **Mutual followers overlap (between two users)** | `relationships overlap <u1> <u2>` | Twiangulate's flagship feature, but as a CLI/agent tool. Requires syncing followers of both users into local store, then INTERSECT. | 7 |
| 8 | **Suspicious follower audit (bots)** | `audit suspicious-followers --threshold` | Heuristic: no profile pic + default-name pattern + recent account + high follow ratio + low engagement. All computable from local store. FollowerAudit-class for free. | 7 |
| 9 | **New followers since** | `relationships new-followers --since 7d` | Requires snapshot diffing. | 7 |
| 10 | **Tweet engagement leaderboard** | `tweets engagement --top 10 --since 30d` | Local SQL over synced tweets: ORDER BY (likes + 2*retweets + 3*replies). Composable with `--user`, `--lang`, `--has-media`. | 7 |
| 11 | **Whois (aggregated profile)** | `whois <handle>` | One command shows: profile, follower count, post velocity, engagement rate, mutuals with you, last active, recent top tweets. Stitches local store + live API into a single agent-friendly view. | 7 |
| 12 | **Saved-search (search local store)** | `search saved --query <q> --since <t>` | Local FTS5 over synced tweets. No rate limits, regex support, composable with `--user`, `--has-media`, JSON output. | 7 |
| 13 | **Twitter archive ZIP import** | `archive import <zip>` | Bootstrap entire local store from your free Twitter data export ZIP — no rate-limit-bound sync needed for historical data. Birdclaw-class. | 8 |
| 14 | **Git-friendly export (yearly JSONL shards)** | `export jsonl --yearly` | Export tweets as version-controllable JSONL with yearly partitioning. Pairs with `--format markdown` for human-readable backups. | 6 |

### Themes (for README grouping)
- **"Asymmetric relationship analytics"** (features 1, 2, 3, 6, 7, 9) — the killer cluster
- **"Audience hygiene"** (features 4, 5, 8) — find dead/spam/inactive accounts
- **"Local intelligence"** (features 10, 11, 12) — analytics powered by the local store
- **"Local-first onboarding & export"** (features 13, 14) — bootstrap from archive, export anywhere

## Self-Brainstorm (extra novel ideas considered, NOT in shipping scope)
- `tweets timeline-replay <user> --since` — re-render someone's timeline as it was at a point in time. **Cut** — requires too much historical data we don't have.
- `dm digest --since` — condensed DM summary. **Cut** — DMs are sensitive, narrow audience.
- `bookmarks tag <id> <tag>` — local tagging. **Cut** — nice-to-have, low score.
- `lists snapshot --diff` — track list membership changes. **Cut** — narrow audience.
- `network triangulation` — find accounts connected to A, B, AND C. **Cut** — would be cool but explodes API call count.

## Total feature count
- **Absorbed**: 42 features (matches every competitor incl. birdclaw; most are auto-generated from spec)
- **Transcendence**: 14 features (all score >= 6, ten score >= 7)
- **Total**: 56 features

## Per-source attribution sanity check
Every absorbed feature is tied to a real competing tool. Every transcendence feature is justified with "only possible because of our local store + cookie auth combo." No SaaS-only features that we can't replicate (e.g., paid analytics dashboards) made the list.
