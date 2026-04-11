# Dub CLI Absorb Manifest

## Sources Analyzed
1. **dubco CLI** (sujjeee/dubco, 24 stars) — Node.js CLI, 3 commands: login, config, link
2. **dubco-mcp-server** (Gitmaxd/dubco-mcp-server-npm, 7 stars) — MCP server, 3 tools: create_link, update_link, delete_link
3. **Official TypeScript SDK** (dub on npm) — Speakeasy-generated, 47+ methods across 14 resources
4. **Official Python SDK** (dub on PyPI) — Speakeasy-generated, mirrors TypeScript SDK
5. **Make.com integration** — Triggers on click/lead/sale, action: create/update link
6. **Zapier integration** — Similar triggers and actions to Make

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Create short link | dubco CLI `link` | `links create --url <url> --key <slug>` | --dry-run, --json, --stdin batch, geo/device targeting flags |
| 2 | Login/auth config | dubco CLI `login` | `auth login` + `config set api-key` | `doctor` validates auth, env var auto-detect, multiple workspaces |
| 3 | Show config | dubco CLI `config` | `config show` | --json output, `config edit`, workspace switching |
| 4 | Create link (MCP) | dubco-mcp-server create_link | `links create` | Same command, but with all 20+ link params, not just 4 |
| 5 | Update link (MCP) | dubco-mcp-server update_link | `links update <id>` | All updatable fields, --dry-run, bulk via `links bulk-update` |
| 6 | Delete link (MCP) | dubco-mcp-server delete_link | `links delete <id>` | --dry-run, bulk via `links bulk-delete`, confirmation prompt |
| 7 | List links | TypeScript SDK links.list | `links list` | --json, --csv, --select, --compact, offline via SQLite after sync |
| 8 | Get link info | TypeScript SDK links.get | `links get <id>` | --json, --select fields, resolves by ID or short key |
| 9 | Count links | TypeScript SDK links.count | `links count` | Group by domain/tag/folder, --json output |
| 10 | Bulk create links | TypeScript SDK links.createMany | `links bulk-create --file <csv/json>` | CSV/JSON file input, --dry-run, progress bar, error report |
| 11 | Bulk update links | TypeScript SDK links.updateMany | `links bulk-update --file <csv/json>` | Same batch input, --dry-run, delta preview |
| 12 | Bulk delete links | TypeScript SDK links.deleteMany | `links bulk-delete --ids <id1,id2,...>` | --dry-run, confirmation, batch from file |
| 13 | Upsert link | TypeScript SDK links.upsert | `links upsert --url <url>` | Idempotent, --dry-run, CI/CD friendly |
| 14 | Retrieve analytics | TypeScript SDK analytics.retrieve | `analytics` | Group by 10+ dimensions, --json/--csv, date ranges, local cache |
| 15 | List events | TypeScript SDK events.list | `events list` | Filter by type/link/customer, --json, --compact, pagination |
| 16 | List tags | TypeScript SDK tags.list | `tags list` | --json, --select, offline search after sync |
| 17 | Create tag | TypeScript SDK tags.create | `tags create <name>` | --color flag, --json output |
| 18 | Update tag | TypeScript SDK tags.update | `tags update <id>` | --json, --dry-run |
| 19 | Delete tag | TypeScript SDK tags.delete | `tags delete <id>` | --dry-run, confirmation |
| 20 | List folders | TypeScript SDK folders.list | `folders list` | --json, tree view, offline search |
| 21 | Create folder | TypeScript SDK folders.create | `folders create <name>` | --json output |
| 22 | Update folder | TypeScript SDK folders.update | `folders update <id>` | --json, --dry-run |
| 23 | Delete folder | TypeScript SDK folders.delete | `folders delete <id>` | --dry-run, warns about orphaned links |
| 24 | List domains | TypeScript SDK domains.list | `domains list` | --json, DNS status column, offline after sync |
| 25 | Create domain | TypeScript SDK domains.create | `domains create <domain>` | DNS setup instructions in output |
| 26 | Update domain | TypeScript SDK domains.update | `domains update <id>` | --json, --dry-run |
| 27 | Delete domain | TypeScript SDK domains.delete | `domains delete <id>` | --dry-run, warns about affected links |
| 28 | Register domain | TypeScript SDK domains.register | `domains register <domain>` | Availability check + register in one flow |
| 29 | Check domain status | TypeScript SDK domains.checkStatus | `domains status <domain>` | DNS verification status, actionable next steps |
| 30 | Track lead | TypeScript SDK track.lead | `track lead --customer <id> --link <id>` | --json, --dry-run, batch from file |
| 31 | Track sale | TypeScript SDK track.sale | `track sale --customer <id> --amount <cents>` | --json, --dry-run, batch from file |
| 32 | List customers | TypeScript SDK customers.list | `customers list` | --json, --select, offline search, FTS |
| 33 | Get customer | TypeScript SDK customers.get | `customers get <id>` | --json, includes linked events summary |
| 34 | Update customer | TypeScript SDK customers.update | `customers update <id>` | --json, --dry-run |
| 35 | Delete customer | TypeScript SDK customers.delete | `customers delete <id>` | --dry-run, confirmation |
| 36 | List partners | TypeScript SDK partners.list | `partners list` | --json, --select, commission summary |
| 37 | Create partner | TypeScript SDK partners.create | `partners create` | --json, --dry-run |
| 38 | Get partner links | TypeScript SDK partners.retrieveLinks | `partners links <id>` | --json, performance summary per link |
| 39 | Create partner link | TypeScript SDK partners.createLink | `partners create-link <partnerId>` | --json, --dry-run |
| 40 | Upsert partner link | TypeScript SDK partners.upsertLink | `partners upsert-link <partnerId>` | Idempotent, --dry-run |
| 41 | Partner analytics | TypeScript SDK partners.analytics | `partners analytics <id>` | --json, date range, group by dimension |
| 42 | Ban partner | TypeScript SDK partners.ban | `partners ban <id>` | --dry-run, reason flag |
| 43 | Deactivate partner | TypeScript SDK partners.deactivate | `partners deactivate <id>` | --dry-run |
| 44 | List commissions | TypeScript SDK commissions.list | `commissions list` | --json, filter by partner/status, totals |
| 45 | Update commission | TypeScript SDK commissions.update | `commissions update <id>` | --json, --dry-run |
| 46 | List payouts | TypeScript SDK payouts.list | `payouts list` | --json, filter by partner/status, totals |
| 47 | List bounty submissions | TypeScript SDK bounties.listSubmissions | `bounties list` | --json, filter by status |
| 48 | Approve submission | TypeScript SDK bounties.approveSubmission | `bounties approve <id>` | --dry-run, batch approve |
| 49 | Reject submission | TypeScript SDK bounties.rejectSubmission | `bounties reject <id>` | --dry-run, reason flag |
| 50 | Create embed token | TypeScript SDK embedTokens.referrals | `embed-tokens create` | --json |
| 51 | Get QR code | TypeScript SDK qrCodes.get | `qr <url>` | SVG/PNG output, --size, --output file, clipboard |
| 52 | Sync all data | (no competitor) | `sync --full` | SQLite persistence for offline queries |
| 53 | Offline search | (no competitor) | `search "<term>"` | FTS5 across links, tags, customers, folders |
| 54 | SQL queries | (no competitor) | `sql "<query>"` | Direct SQLite access for power users |

## Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 1 | Campaign performance dashboard | `campaigns` | Requires local join across links + analytics + tags — Dub analytics API groups by link/country/device but NOT by tag. Only a local store can aggregate analytics per tag/campaign. | 10/10 | Dub analytics groupBy lacks tag dimension; G2 comparison notes per-campaign reporting gap |
| 2 | Attribution funnel | `funnel --tag <campaign>` | Requires joining events by customer across links to compute click→lead→sale conversion rates per campaign. No single API call provides this. | 10/10 | Dub positions as "link attribution platform"; PIMMS competitor comparison focuses on conversion tracking |
| 3 | Tag analytics rollup | `tags analytics` | Analytics per tag across all tagged links — which tags drive the most clicks/leads/sales? Requires local join of links + tags + analytics time series. | 10/10 | SDK confirms tags are core resource; analytics API groupBy doesn't include tags as dimension |
| 4 | Customer journey | `customers journey <id>` | Full timeline of a customer's interactions: links clicked, lead conversion, purchases. Requires joining customers + events + links in SQLite. | 9/10 | Dub tracks customers + events + lead/sale tracking; no existing tool shows the full path |
| 5 | Link health audit | `links stale --days 30` | Find links with declining/zero click velocity over time. Requires historical click time series stored locally — API only returns current totals. | 8/10 | No competitor surfaces stale link detection; analogous to stale issue detection in PM tools |
| 6 | Partner leaderboard | `partners leaderboard` | Rank partners by commission earned, conversion rate, clicks generated. Requires joining partners + commissions + analytics across all partner links. | 8/10 | Dub markets partner programs heavily; no tool provides cross-partner comparison |
| 7 | Domain utilization | `domains report` | Links per domain, click distribution, conversion rates by domain. Requires joining domains + links + analytics in SQLite. | 8/10 | Multiple custom domains is a Dub differentiator; no existing tool shows domain-level performance comparison |
| 8 | Duplicate link detector | `links duplicates` | FTS5 across all link URLs and titles to find near-duplicates pointing to the same destination or conflicting redirects. | 5/10 | Large workspaces accumulate duplicate links; FTS5 makes this trivial |

**Total: 54 absorbed + 8 transcendence = 62 features**
