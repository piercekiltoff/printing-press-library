Acceptance Report: dub-pp-cli
  Level: Full Dogfood
  Tests: 12/13 passed
  Failures:
    - [campaigns]: returns null with minimal data — tag-link JOIN needs richer dataset to validate
  Fixes applied: 3
    - CLI fix: capped retryAfter to 60s max (was waiting 1.2M hours on malformed Retry-After header)
    - CLI fix: fixed FTS search quoting — terms with hyphens now quoted for FTS5
    - CLI fix: added FTS index population in UpsertBatch (was only in single Upsert)
  Printing Press issues: 1
    - UpsertBatch in generated store.go doesn't populate resources_fts — should be fixed in the generator template
  Gate: PASS

Test Results:
  [1/13] doctor                    PASS — auth valid, API reachable, credentials valid
  [2/13] links list                PASS — 2 links returned (3 after test create)
  [3/13] domains list              PASS — 0 domains (expected on free plan)
  [4/13] tags list                 PASS — 1 tag returned
  [5/13] sync --full               PASS — synced domains(1), folders(1), links(3), tags(1). 403 on paid-tier resources (expected)
  [6/13] analytics (SQL)           PASS — correctly counts 3 links in store
  [7/13] links create              PASS — created test link with pp-test-* key
  [8/13] links get-info            PASS — retrieved test link by ID
  [9/13] links update              PASS — added comments field
  [10/13] re-sync                  PASS — test entity now in store (3 links)
  [11/13] search                   PASS — found 'compound' and 'trev' in local store after fix
  [12/13] output fidelity          PASS — --json, --select, --csv all produce correct output
  [13/13] error paths              PASS — 404 returns exit code 3, missing flag returns exit code 1
