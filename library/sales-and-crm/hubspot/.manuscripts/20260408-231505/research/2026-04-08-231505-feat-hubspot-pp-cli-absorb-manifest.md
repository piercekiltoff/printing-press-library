# HubSpot CLI Absorb Manifest

## Sources Cataloged
1. **bcharleson/hubspot-cli** - TypeScript CLI, 55 CRM commands across 11 groups, MCP server mode
2. **Official HubSpot MCP Server** (beta) - Read-only access to 11 CRM object types
3. **peakmojo/mcp-hubspot** - Community MCP with vector search, caching for contacts/companies/conversations
4. **@hubspot/api-client** (npm) - Official Node.js SDK v13.5.0 covering full API surface
5. **hubspot-api-client** (PyPI) - Official Python SDK v12.0.0
6. **clarkmcc/go-hubspot** (Go) - OpenAPI-generated Go client covering all specs
7. **HubSpot/hubspot-cli** (official) - CMS-only CLI (design manager, serverless, HubDB)

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | List contacts | bcharleson contacts list | contacts list with --json, --select, --csv, --limit | Offline after sync, FTS search, SQL composable |
| 2 | Get contact by ID | bcharleson contacts get | contacts get <id> --json | Cached locally, works offline |
| 3 | Create contact | bcharleson contacts create | contacts create --email --firstname --lastname --dry-run | Agent-native, --dry-run, --stdin batch |
| 4 | Update contact | bcharleson contacts update | contacts update <id> --property key=value --dry-run | Idempotent, batch-capable |
| 5 | Delete contact | bcharleson contacts delete | contacts delete <id> --dry-run --confirm | Safe defaults, requires confirmation |
| 6 | Search contacts | bcharleson contacts search | contacts search <query> --property --operator | Offline FTS + API search, regex capable |
| 7 | Merge contacts | bcharleson contacts merge | contacts merge <primary> <secondary> --dry-run | Preview merge diff before execution |
| 8 | List companies | bcharleson companies list | companies list --json --select --csv --limit | Offline, FTS, SQL composable |
| 9 | Get company | bcharleson companies get | companies get <id> --json | Cached, offline |
| 10 | Create company | bcharleson companies create | companies create --name --domain --dry-run | Batch, agent-native |
| 11 | Update company | bcharleson companies update | companies update <id> --property key=value | Idempotent |
| 12 | Delete company | bcharleson companies delete | companies delete <id> --dry-run --confirm | Safe defaults |
| 13 | Search companies | bcharleson companies search | companies search <query> | Offline FTS |
| 14 | List deals | bcharleson deals list | deals list --pipeline --stage --owner --json | Offline, filterable by pipeline/stage/owner |
| 15 | Get deal | bcharleson deals get | deals get <id> --json | Cached, associations included |
| 16 | Create deal | bcharleson deals create | deals create --name --pipeline --stage --amount --dry-run | Agent-native, scriptable |
| 17 | Update deal | bcharleson deals update | deals update <id> --stage --amount --dry-run | Stage transitions tracked |
| 18 | Delete deal | bcharleson deals delete | deals delete <id> --dry-run --confirm | Safe |
| 19 | Search deals | bcharleson deals search | deals search <query> | Offline FTS |
| 20 | List tickets | bcharleson tickets list | tickets list --pipeline --status --json | Offline, filterable |
| 21 | Get ticket | bcharleson tickets get | tickets get <id> --json | Cached |
| 22 | Create ticket | bcharleson tickets create | tickets create --subject --pipeline --status --dry-run | Agent-native |
| 23 | Update ticket | bcharleson tickets update | tickets update <id> --status --dry-run | Status transitions tracked |
| 24 | Delete ticket | bcharleson tickets delete | tickets delete <id> --dry-run --confirm | Safe |
| 25 | Search tickets | bcharleson tickets search | tickets search <query> | Offline FTS |
| 26 | List owners | bcharleson owners list | owners list --json --select | Offline, cached |
| 27 | Get owner | bcharleson owners get | owners get <id> --json | Cached |
| 28 | List pipelines | bcharleson pipelines list | pipelines list --object-type --json | Offline |
| 29 | Get pipeline | bcharleson pipelines get | pipelines get <id> --json | Includes stages |
| 30 | Pipeline stages | bcharleson pipelines stages | pipelines stages <id> --json | Offline, with deal counts per stage |
| 31 | Create engagement note | bcharleson engagements create-note | notes create --contact --body --dry-run | Agent-native, attach to any object |
| 32 | Create engagement email | bcharleson engagements create-email | emails create --contact --subject --body --dry-run | Scriptable |
| 33 | Create engagement call | bcharleson engagements create-call | calls create --contact --duration --body --dry-run | Log calls from scripts |
| 34 | Create engagement task | bcharleson engagements create-task | tasks create --contact --subject --due --dry-run | Batch task creation |
| 35 | Create engagement meeting | bcharleson engagements create-meeting | meetings create --contact --title --start --end --dry-run | Calendar integration |
| 36 | List engagements | bcharleson engagements list | engagements list --type --contact --json | Offline, filterable |
| 37 | Get engagement | bcharleson engagements get | engagements get <id> --json | Cached |
| 38 | Delete engagement | bcharleson engagements delete | engagements delete <id> --confirm | Safe |
| 39 | List associations | bcharleson associations list | associations list --from-type --from-id --to-type --json | Offline graph traversal |
| 40 | Create association | bcharleson associations create | associations create --from --to --type --dry-run | Batch, idempotent |
| 41 | Delete association | bcharleson associations delete | associations delete --from --to --type --confirm | Safe |
| 42 | List contact lists | bcharleson lists list | lists list --json --select | Offline |
| 43 | Get list | bcharleson lists get | lists get <id> --json | With member count |
| 44 | Create list | bcharleson lists create | lists create --name --filters --dry-run | Agent-native |
| 45 | Update list | bcharleson lists update | lists update <id> --name | Idempotent |
| 46 | Delete list | bcharleson lists delete | lists delete <id> --confirm | Safe |
| 47 | Add list members | bcharleson lists add-members | lists add-members <id> --contacts --dry-run | Batch capable |
| 48 | Remove list members | bcharleson lists remove-members | lists remove-members <id> --contacts --dry-run | Batch capable |
| 49 | Get list members | bcharleson lists get-members | lists members <id> --json --limit | Offline after sync |
| 50 | List properties | bcharleson properties list | properties list --object-type --json | Offline, filterable |
| 51 | Get property | bcharleson properties get | properties get <object-type> <name> --json | Cached |
| 52 | Create property | bcharleson properties create | properties create --object-type --name --type --dry-run | Agent-native |
| 53 | Update property | bcharleson properties update | properties update <object-type> <name> --dry-run | Idempotent |
| 54 | Delete property | bcharleson properties delete | properties delete <object-type> <name> --confirm | Safe |
| 55 | Cross-object search | bcharleson search run | search <query> --types contacts,deals,tickets | Offline FTS across all synced types |
| 56 | Auth/config management | bcharleson login/logout/status | auth login, auth status, doctor | Better diagnostics, env var validation |
| 57 | MCP read contacts | Official HubSpot MCP | Already covered by contacts list/get/search | Faster, offline capable |
| 58 | MCP read companies | Official HubSpot MCP | Already covered by companies list/get/search | Faster, offline capable |
| 59 | MCP read deals | Official HubSpot MCP | Already covered by deals list/get/search | Faster, offline capable |
| 60 | MCP read tickets | Official HubSpot MCP | Already covered by tickets list/get/search | Faster, offline capable |
| 61 | MCP read products | Official HubSpot MCP | products list/get/search | Offline capable |
| 62 | MCP read quotes | Official HubSpot MCP | quotes list/get | Offline capable |
| 63 | MCP read invoices | Official HubSpot MCP | invoices list/get | Offline capable |
| 64 | MCP read line items | Official HubSpot MCP | line-items list/get | Offline capable |
| 65 | Vector search | peakmojo/mcp-hubspot | Already beaten by FTS5 offline search | No external vector DB needed |
| 66 | Full sync | @hubspot/api-client patterns | sync --full, sync --incremental | SQLite-backed, cursor-tracked, offline |
| 67 | Batch operations | @hubspot/api-client batch methods | batch create/update/delete on all object types | --stdin batch, --dry-run preview |
| 68 | Imports | HubSpot Imports API | imports create --file --object-type, imports status | CSV/JSON import directly from CLI |

### Transcendence (only possible with our local data layer)

| # | Feature | Command | Why Only We Can Do This | Score | Evidence |
|---|---------|---------|------------------------|-------|----------|
| 1 | Pipeline velocity analysis | deals velocity --pipeline <name> --weeks 8 | Requires historical deal stage snapshots in SQLite. Shows avg time per stage, conversion rates, bottleneck stages. No HubSpot report does this without Sales Hub Enterprise. | 9/10 | Domain-specific CRM opportunity. bcharleson/hubspot-cli has no analytics. HubSpot requires Enterprise tier for velocity reports. |
| 2 | Stale deal detection | deals stale --days 14 --pipeline <name> | Requires local join of deal stage history + last activity date + owner. Finds deals stuck in a stage with no recent engagement. | 9/10 | #1 sales manager pain point. No existing CLI or MCP offers this. HubSpot web UI requires manual deal board inspection. |
| 3 | Contact engagement scoring | contacts engagement --days 30 | Requires joining contacts + all engagement types (calls, emails, meetings, tasks, notes) in SQLite. Shows engagement frequency, last touch, engagement gap. | 8/10 | RevOps pain point from research. peakmojo MCP has basic activity but no scoring. HubSpot's native engagement scoring is paid add-on. |
| 4 | Owner workload balance | owners workload | Requires cross-entity join: owner -> deals (by stage) + tickets (by status) + tasks (overdue). Shows who's overloaded and who has capacity. | 8/10 | Research shows "automatically assigning leads to the relevant person" is top workflow. No tool shows current load to inform assignment. |
| 5 | Association graph traversal | graph <object-type> <id> --depth 2 | Requires recursive association lookups stored in SQLite. Shows the full relationship web: contact -> company -> deals -> tickets -> engagements. One command gives complete context. | 8/10 | bcharleson has flat association list. No tool does recursive graph traversal. Agent-native: "tell me everything about this contact." |
| 6 | Pipeline forecast | deals forecast --pipeline <name> | Requires deals + pipeline stages + historical close rates in SQLite. Weighted pipeline: (deal amount * stage probability * historical conversion). | 7/10 | HubSpot Forecasts API exists but requires paid tier. Our version uses actual historical data from the user's own deals. |
| 7 | Duplicate detection | contacts duplicates --threshold 0.8 | Requires FTS5 across contact names, emails, company associations. Fuzzy matching on name + email + company to find likely duplicates. | 7/10 | "Merge contacts" exists in bcharleson but no detection. HubSpot native dedup is Operations Hub paid feature. |
| 8 | Deal-to-engagement coverage | deals coverage --pipeline <name> | Requires join of deals + associated contacts + engagements. Shows which open deals have contacts with no recent engagement. Finds deals going cold. | 7/10 | Compound of workflows #1 (deal mgmt) and #3 (engagement logging). No tool correlates deal health with engagement activity. |
| 9 | Property audit | properties audit --object-type contacts | Requires synced contacts + property definitions. Shows which custom properties are unused (null for >90% of records), which have inconsistent values. | 6/10 | CRM hygiene is #2 workflow. No tool audits property usage. RevOps manually checks this. |
| 10 | Activity timeline | timeline <object-type> <id> --days 90 | Requires all engagement types + associations in SQLite. Unified chronological view of every interaction with an entity. | 6/10 | HubSpot Timeline API exists but is event-definition focused. This merges all engagement types into one stream. |
