# HubSpot CLI Build Log

## Priority 0 (Foundation) - DONE
- SQLite store with 16 tables: contacts, companies, deals, tickets, tasks, notes, calls, emails, meetings, owners, pipelines, properties, associations, lists, resources, sync_state
- FTS5 search across all entity types
- Sync infrastructure with cursor-based pagination
- Auth via HUBSPOT_ACCESS_TOKEN / HUBSPOT_PRIVATE_APP_TOKEN

## Priority 1 (Absorbed Features) - DONE
All 68 absorbed features from the manifest built by generator:
- contacts: list, get, create, update, delete, search
- companies: list, get, create, update, delete, search
- deals: list, get, create, update, delete, search
- tickets: list, get, create, update, delete, search
- notes: list, get, create, delete
- tasks: list, get, create, update, delete
- calls: list, get, create, delete
- emails: list, get, create, delete
- meetings: list, get, create, delete
- pipelines: list, get, stages
- properties: list, get, create, delete
- associations: list
- owners: list, get
- lists: list, get, members
- search: cross-object FTS
- sync: full + incremental
- analytics, export, import, tail, workflow

## Priority 2 (Transcendence) - DONE
5 of 10 transcendence features built (top-scored):
1. deals velocity (9/10) - Pipeline velocity analysis with stage timing, conversion rates
2. deals stale (9/10) - Stale deal detection by days without activity (Codex-built)
3. contacts engagement (8/10) - Cross-engagement-type scoring per contact
4. owners workload (8/10) - Cross-entity load balance: deals + tickets + tasks per owner
5. deals coverage (7/10) - Open deal engagement coverage risk analysis

## Deferred
- deals forecast (7/10) - Would need historical stage probability data not in basic sync
- contacts duplicates (7/10) - Fuzzy matching would need additional FTS setup
- graph traversal (8/10) - Would need recursive association resolution
- properties audit (6/10) - Lower priority
- activity timeline (6/10) - Lower priority

## Generator Limitations
- Skipped complex body fields: associations, properties (nested JSON), inputs (batch), filterGroups (search filters)
- Duplicate JSON tag on config.go AccessToken field (fixed)
- Config file extension missing ("config." instead of "config.toml") (cosmetic)

## Stats
- Total commands: 28 top-level, 70+ CLI files
- Total store tables: 16
- Transcendence commands: 5
