warning: resource "export" from path "/tenant/{tenant}/export/invoice-templates" would shadow framework cobra command "export"; renamed to "memberships-export"
WARNING: spec.OwnerName is empty; falling back to slug-shaped Owner ("user") for `author:` field. Set `git config user.name` (display name, e.g. "Trevin Chow") to populate this correctly.
WARNING: spec.Printer is empty; README printer attribution will be omitted. Set `git config github.user` (your GitHub @handle) to populate this correctly before publishing.
PASS go mod tidy
PASS govulncheck ./...
PASS go vet ./...
PASS go build ./...
PASS build runnable binary
PASS servicetitan-memberships-pp-cli --help
PASS servicetitan-memberships-pp-cli version
PASS servicetitan-memberships-pp-cli doctor
Generated servicetitan-memberships at C:\Users\pierc\printing-press\.runstate\temp-855456de\runs\20260515-072215\working\servicetitan-memberships-pp-cli
Bundled C:\Users\pierc\printing-press\.runstate\temp-855456de\runs\20260515-072215\working\servicetitan-memberships-pp-cli\build\servicetitan-memberships-pp-mcp-windows-amd64.mcpb

## Phase 2.5 — Generator regression patches

Same 3-bug carry-forward as the inventory retro (2026-05-13); v4.6.1 still regresses these against ST module specs.

| Patch | File | Why |
|---|---|---|
| composed-auth-apikey-config | `internal/config/config.go` | Added `StAppKey` (`toml:"app_key"`) and `TenantID` (`toml:"tenant_id"`) fields; `Load()` now reads `ST_APP_KEY` and `ST_TENANT_ID` with defensive `TrimSpace`; auth-source label updated. (#1303 apiKey half, #1332) |
| composed-auth-apikey-wire | `internal/client/client.go` | Inject `ST-App-Key` header on every request alongside `Authorization`; mirror in `--dry-run` output. (#1303 apiKey half) |
| composed-auth-doctor | `internal/cli/doctor.go` | Recognize composed-auth-ready state from credentials present (not bearer); require all four of ST_APP_KEY / ST_CLIENT_ID / ST_CLIENT_SECRET / ST_TENANT_ID in env-var check. (#1303, #1332) |
| sync-registry | `internal/cli/sync.go` | `defaultSyncResources()` populated with the 6 cacheable Memberships v2 resources; `syncResourcePath()` populated with tenant-substituted paths. (#1305) |

Verified: `go build ./...` exit 0, `go vet ./...` exit 0, `doctor --json` returns `auth=configured (composed)`, `auth_source=env:ST_APP_KEY`, `env_vars=OK 4/4 available`.

Mirrors patch comments and structure of sibling `servicetitan-pricebook` (shipped 2026-05-14 against the same v4.6.1 binary). Same retro signal — file as Phase 6 retro-candidate.
