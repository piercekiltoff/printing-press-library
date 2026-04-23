# Live-org verification report

> **Status: NOT YET RUN**
>
> This report will be filled in when Matt pairs with his contact on the runbook (`docs/live-verification-runbook.md`). It is the delivery gate to Benioff outreach.

---

## Session metadata

| Field | Value |
|-------|-------|
| Date | _to be filled_ |
| Tester | _to be filled (org owner)_ |
| Witness | Matt Van Horn |
| Org type | _Developer Edition / Enterprise sandbox / Unlimited sandbox_ |
| Org ID | _to be filled_ |
| CLI version | _from `salesforce-headless-360-pp-cli version`_ |
| Salesforce API version targeted | v63.0 |
| `sf` CLI version | _to be filled_ |

---

## Required checks

| # | Check | Status | Observed | Notes |
|---|-------|--------|----------|-------|
| 1 | sf CLI fall-through | _PASS / FAIL_ | | |
| 2 | doctor full pass | _PASS / FAIL_ | | |
| 3 | Composite Graph in sync | _PASS / FAIL_ | | |
| 4 | UI API sharing cross-check | _PASS / FAIL_ | | |
| 5 | FLS intersection actually hides a field | _PASS / FAIL_ | | |
| 6 | Tooling compliance map loads | _PASS / FAIL_ | | |
| 7 | trust register writes Certificate or CMDT | _PASS / FAIL_ | | |
| 8 | agent context produces a bundle | _PASS / FAIL_ | | |
| 9 | agent verify --strict --deep PASS on valid bundle | _PASS / FAIL_ | | |
| 10 | agent verify --strict --deep FAIL on tampered bundle | _PASS / FAIL_ | | |
| 11 | SF360_Bundle_Audit__c row appears | _PASS / FAIL_ | | |
| W1 | agent update writes one safe Account field | _PASS / FAIL_ | | |
| W2 | agent upsert twice with same key shows no-op | _PASS / FAIL_ | | |
| W3 | agent log-activity creates Task | _PASS / FAIL_ | | |
| W4 | agent advance moves Opportunity stage | _PASS / FAIL_ | | |
| W5 | stale write conflict rejected | _PASS / FAIL_ | | |
| W6 | FLS write denial enforced | _PASS / FAIL_ | | |

## Optional checks

| # | Check | Status | Observed / Skip reason |
|---|-------|--------|------------------------|
| O1 | Apex companion deploy, including SafeWrite and SafeUpsert | _PASS / FAIL / SKIP_ | |
| O2 | Bulk fallback path | _PASS / FAIL / SKIP_ | |
| O3 | Data Cloud profile | _PASS / FAIL / SKIP_ | |
| O4 | Slack linkage | _PASS / FAIL / SKIP_ | |
| O5 | Slack inject end-to-end | _PASS / FAIL / SKIP_ | |

---

## Outstanding issues found during the session

_If any required check FAILed, list the bug + tracking issue + fix commit here. No FAILs are allowed at v1.1.0._

| Check # | Issue | Tracking | Fix commit |
|---------|-------|----------|-----------|
| _none yet_ | | | |

---

## Sign-off

```
Tester signature: _____________________________
Date:             _____________________________
JWS over this report: _________________________
   (use: salesforce-headless-360-pp-cli agent verify --deep over a bundle wrapping this report file)
```

Once every required row is PASS and this report is signed, tag the repo `v1.1.0` and proceed with Benioff outreach per the plan's "Delivery to Benioff" section.
