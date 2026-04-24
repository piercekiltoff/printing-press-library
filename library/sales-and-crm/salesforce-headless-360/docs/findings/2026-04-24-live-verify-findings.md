# SF360 Live-Verify Findings — trent@nfchq.com

**Tester:** Trent Matthias
**Org:** NFC TimDev3 sandbox (Developer sandbox, refreshed 4/2/2026)
**Date:** 2026-04-22
**CLI version:** v1.1 @ feat/salesforce-headless-360
**PR:** https://github.com/mvanhorn/printing-press-library/pull/111

## Philosophy
Per Matt: "Brute force it / And take notes / To FIX it / For everyone else."
Every papercut is a finding. Every workaround becomes a line item in the fix plan.

---

## Finding F-001: Sandbox login requires different args than plan specifies
**Severity:** doc gap / UX
**Where:** `docs/plans/2026-04-22-004-feat-salesforce-360-writes-plan.md` Step 1
**Plan says:** `sf org login web --alias sf360-test`
**Reality for sandbox orgs:** that command hits `login.salesforce.com` (prod only). Sandboxes need `--instance-url https://test.salesforce.com` OR the org's My Domain sandbox URL `https://<mydomain>--<sandbox>.sandbox.my.salesforce.com`.
**Impact:** any tester running this against a sandbox (which is what Matt's "Don't run against critical account" guidance implicitly recommends) silently hits wrong endpoint.
**Fix suggestion:** add a one-liner table to plan Step 1 covering prod vs sandbox vs DE org login commands.

## Finding F-002: Sandbox password != prod password (refresh-time snapshot)
**Severity:** doc gap
**Reality:** Sandbox snapshots prod password at refresh time. If prod password changed since refresh, sandbox password is stale.
**Easiest unblock:** from prod Salesforce UI → Setup → Sandboxes → click "Log In" next to the sandbox. Uses prod SSO session. No password needed. Then run `sf org login web` with the sandbox's My Domain URL — OAuth rides on existing browser session.
**Fix suggestion:** "Troubleshooting" block in plan covering this exact path.

## Finding F-003: Metadata deploy step missing from plan (confirmed by audit)
**Severity:** blocker (audit writes fail silently without this)
**Where:** audit report of `metadata/README.md` — confirmed `SF360_Bundle_Audit__c`, `SF360_Write_Audit__c`, `SF360_Bundle_Key__mdt`, and `SF360_Idempotency_Key__c` fields must be deployed first.
**Plan gap:** Step 0 / Step 1 do not mention `sf project deploy start --source-dir metadata`.
**Fix suggestion:** add as explicit Step 0.5 before any CLI command runs.

## Finding F-004: FLS.AllowFieldWrite(user) ignores user parameter
**Severity:** spec-vs-code inconsistency
**Where:** `internal/security/fls.go:136` — `_ = user` discards the parameter
**Impact:** `--run-as-user` flag is only enforced at Apex companion layer, not at Go FLS filter. If Apex not deployed, writes flow through UI API as authenticated token holder's FLS, not run-as-user's.
**Fix suggestion:** either enforce user properly in Go layer, or remove the parameter from the signature to avoid implying enforcement that isn't there.

## Finding F-005: "Log In" shortcut from Setup > Sandboxes doesn't carry prod SSO
**Severity:** UX surprise (not a blocker, but wastes 5 min of confusion)
**Reality:** on at least some orgs (NFC confirmed), clicking "Log In" from the Sandboxes setup page redirects to the sandbox login form, does NOT automatically authenticate via prod session. Tester thinks they're about to be SSO'd, instead hits password wall again.
**Impact:** anyone following the "use the Log In shortcut to avoid password friction" advice hits a dead end.
**Unblock:** click "Forgot Your Password?" on the sandbox login page. Reset email routes to real prod inbox (SF strips the `.sandbox-name` suffix). Set new password, log in.
**Fix suggestion:** plan's troubleshooting section should recommend the Forgot Password path as the canonical unblock, not the "Log In" shortcut.

## Finding F-006: NFC org requires Sandbox Access public-group selection
**Severity:** UX / doc gap (blocks provisioning)
**Where:** Setup > Sandboxes > New Sandbox > Sandbox Options
**Reality:** NFC org config makes "Sandbox Access" a required field. Blocks Create until a public group is selected.
**Impact:** First-time sandbox creators on orgs with this config setting will hit a surprise required field. Plan assumes sandbox creation is a no-op.
**Unblock:** pre-create a "Sandbox Access" public group containing only the tester, before attempting sandbox creation. 30 sec in Setup > Public Groups > New.
**Fix suggestion:** plan should note that some orgs require this and provide the 1-minute pre-flight.

## Finding F-007: Metadata `fields/` flat layout is not SFDX-deployable
**Severity:** blocker (breaks plan Step 0/0.5 "deploy metadata")
**Where:** `metadata/fields/<SObject>.SF360_Idempotency_Key__c.field-meta.xml`
**Reality:** modern `sf project deploy start --source-dir metadata` fails with `TypeInferenceError: Could not infer a metadata type`. SFDX source format requires fields at `objects/<SObject>/fields/<Field>.field-meta.xml` with bare `<fullName>` (no object prefix), not the flat `fields/` layout.
**Reproduction:** from a fresh clone, run `sf project deploy start --source-dir metadata --target-org <alias>`. Errors immediately.
**Impact:** the exact command Matt's README documents fails for all testers. `README.md` line ~33 says "Admins that prefer SFDX object folders can move each file under metadata/objects/<SObject>/fields/" — but this isn't a preference, it's a requirement for any modern sf CLI.
**Fix suggestion:** restructure fields into per-object folders, rewrite `<fullName>` bare, update README. OR provide a package.xml + switch to MDAPI deploy.
**Workaround applied:** restructured in-place via shell script.

## Finding F-008: Task/Event idempotency fields fail to deploy
**Severity:** blocker for Task/Event write verbs
**Where:** `metadata/objects/Task/fields/` and `metadata/objects/Event/fields/`
**Reality:** deploy error "Entity Enumeration Or ID: bad value for restricted picklist field: Event/Task". Task and Event are Activity sub-types in modern Salesforce; custom External ID fields on them have specific limitations.
**Workaround applied:** removed Task/Event idempotency fields from deploy. `agent log-activity` (W3 in verify) creates Tasks without the idempotency field — verify if that breaks idempotency guarantees for W3.
**Fix suggestion:** investigate Activity parent object approach OR document that Task/Event idempotency is a v1.2 feature pending Activity-object research.

## Finding F-009: CMDT declares invalid `<deploymentStatus>` element
**Severity:** blocker
**Where:** `metadata/SF360_Bundle_Key__mdt/SF360_Bundle_Key__mdt.object-meta.xml` line 3
**Reality:** `<deploymentStatus>Deployed</deploymentStatus>` is valid for CustomObject but not for Custom Metadata Type. Error: "Cannot specify: deploymentStatus for Custom Metadata Type".
**Workaround applied:** removed the element.
**Fix suggestion:** remove from source.

## Finding F-010: CMDT at wrong root-level location for SFDX source format
**Severity:** blocker
**Where:** `metadata/SF360_Bundle_Key__mdt/` (at metadata root, not under `metadata/objects/`)
**Reality:** SFDX source format requires all CustomObject definitions (including CMDT) under `objects/`. Located at metadata root, the CMDT is either not picked up or causes deploy ambiguity.
**Workaround applied:** moved to `metadata/objects/SF360_Bundle_Key__mdt/`.
**Fix suggestion:** reorganize source tree.

## Finding F-011: `ActingUser__c` lookup to User with `<deleteConstraint>Restrict</deleteConstraint>` invalid
**Severity:** blocker
**Where:** `metadata/objects/SF360_Write_Audit__c/fields/ActingUser__c.field-meta.xml`
**Reality:** User object does not support cascade/restrict delete constraints for lookups. Error: "Cannot add a lookup relationship child with cascade or restrict options to User".
**Workaround applied:** removed `<deleteConstraint>` AND `<required>true</required>` (required lookups must specify a constraint, User lookups can't have one — contradiction, so must be optional).
**Fix suggestion:** redesign ActingUser__c as optional lookup, or use text field for User ID with explicit validation in Apex.

## Finding F-012: `ExecutionStatus__c` has invalid `<defaultValue>pending</defaultValue>`
**Severity:** blocker
**Where:** `metadata/objects/SF360_Write_Audit__c/fields/ExecutionStatus__c.field-meta.xml`
**Reality:** restricted picklist with values inside `<valueSet><valueSetDefinition>` already declares the default via `<default>true</default>` on the "pending" value. Top-level `<defaultValue>pending</defaultValue>` is treated as formula expression and rejected.
**Workaround applied:** removed top-level defaultValue element.
**Fix suggestion:** remove from source — `<default>true</default>` inside valueSetDefinition is sufficient.

## Finding F-013: LongTextArea fields declared `<required>true</required>` (not supported)
**Severity:** blocker
**Where:** `metadata/objects/SF360_Bundle_Key__mdt/fields/PublicKeyPem__c.field-meta.xml`, `Receipt__c.field-meta.xml`
**Reality:** Salesforce does not support required constraint on LongTextArea fields. Error: "Can not specify 'required' for a CustomField of type LongTextArea".
**Workaround applied:** removed `<required>true</required>` from both.
**Fix suggestion:** use Text field if uniqueness/required needed, or enforce at application layer.

## Finding F-014: Required-field FLS declarations rejected in PermissionSet
**Severity:** blocker
**Where:** `metadata/permissionsets/SF360_Key_Registrar.permissionset-meta.xml`
**Reality:** 8 `<fieldPermissions>` blocks declare FLS for required fields on SF360_Write_Audit__c + SF360_Bundle_Audit__c. Required fields are always accessible — FLS declarations are redundant and rejected at deploy. Error: "You cannot deploy to a required field".
**Impact:** PermissionSet fails to deploy even when all custom fields succeed.
**Workaround applied:** stripped all `<fieldPermissions>` for the 8 required fields from the permissionset.
**Fix suggestion:** remove from source — required fields don't need explicit FLS.

---

# Summary so far

**Deploy succeeded after 8 metadata fixes.** Matt's `sf project deploy start --source-dir metadata` one-liner fails catastrophically from a clean clone. 4 rounds of deploy-debug-edit needed to get all components to deploy on a vanilla Developer sandbox.

## Finding F-015: Plan assumes vanilla org; real prod orgs break seeding
**Severity:** blocker for testing against any realistic org
**Reality:** the plan assumes you can "pick an Account with rich data" pre-existing in the org. Developer sandboxes copy metadata only (no records). Seeding a test Account in a real NFC-style customized org hit:
- Custom validation rule: Industry required + must be from 12-value restricted list (not 'Technology')
- Custom required field: `Title_Tier__c` on Contact
- Custom validation rule: Opportunities must start in `Briefing` stage
- Custom required field: `Number_of_Courts__c` on Opportunity
- Custom required User lookup: `PD_Collaborator__c` on Opportunity
- Restricted picklists: some picklist values visible in describe aren't valid for all record types (Title_Tier__c "TBD -To Be Determined" rejected)
- Standard `AnnualRevenue` field removed from Account entirely

Total: 5 rounds of seed-debug-edit.
**Impact:** plan Step 1's "pick test fixtures" guidance silently assumes a vanilla org. Any heavily customized prod-derived sandbox (which is most real Salesforce orgs) breaks.
**Fix suggestion:**
1. Provide `scripts/seed-minimal.apex` that creates generic test data but allows field overrides via env vars
2. Document: "for heavily customized orgs, disable validation rules in sandbox OR use existing data"
3. Add a preflight script that describes required fields per sobject and generates a minimal seed accordingly

## Finding F-016: Plan's `agent log-activity` creates Task — broken by F-008
**Severity:** high (W3 verify check will fail)
**Reality:** W3 in live-verify creates a Task via `agent log-activity`. Task.SF360_Idempotency_Key__c was removed in F-008 fix due to Activity-object metadata bug. Idempotency on Task creation is therefore unenforced — W3 expects idempotency behavior it can't have.
**Impact:** either W3 will silently succeed without real idempotency, or the CLI will fail hard looking for a field that doesn't exist.
**Fix suggestion:** fix Activity object metadata support OR disable W3 entirely OR document limitation.

## Finding F-017: CLI does not accept `--org` flag; verify script is broken
**Severity:** CRITICAL BLOCKER (script cannot run)
**Where:** `scripts/live-verify.sh` — every `"$CLI" --org "$ORG"` invocation; also `docs/plans/*.md` plan document
**Reality:** `salesforce-headless-360-pp-cli --help` shows no `--org` flag globally and no per-command `--org`. Running any `$CLI --org <alias> <verb>` returns `Error: unknown flag: --org`. Org selection is implicit via `sf config set target-org=<alias>`.
**Impact:** nobody following the plan runbook verbatim can get past Check 1. Script crashes instantly.
**Fix suggestion:**
1. Either add `--org` (aliasing to `sf config get target-org` override), OR
2. Rewrite script to drop `--org` and require `sf config set target-org` preflight, OR
3. Use env var `SF_TARGET_ORG` and document it.

## Finding F-018: cleanup trap crashes on unbound arrays
**Severity:** medium
**Where:** `scripts/live-verify.sh:29`
**Reality:** when script fails early (before any Task is created), `cleanup_task_ids` is empty. `set -u` + `${cleanup_task_ids[@]}` → unbound variable crash DURING cleanup trap, masking the real failure and potentially leaving state unreverted.
**Fix suggestion:** use `${cleanup_task_ids[@]:-}` or wrap loop with `[[ ${#cleanup_task_ids[@]} -gt 0 ]] &&`.
