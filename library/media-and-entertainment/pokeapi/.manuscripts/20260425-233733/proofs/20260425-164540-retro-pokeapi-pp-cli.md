# Printing Press Retro: PokeAPI

## Session Stats
- API: PokeAPI
- Spec source: official OpenAPI spec (`https://raw.githubusercontent.com/PokeAPI/pokeapi/master/openapi.yml`)
- Printing Press version: 2.3.5 (`19e996a` / `v2.3.5`)
- Scorecard: not re-run in this retro; shipcheck evidence came from dogfood + publish validation
- Verify/pass gate: `printing-press publish validate` passed
- Dogfood: WARN; novel feature check passed 5/5 after manual graph-command implementation
- Fix loops: 1 major loop (initial generate → dogfood missing features → implement graph commands → dogfood/validate pass)
- Manual code edits: 2 generated CLI areas (`internal/cli/pokemon_graph.go`, root command wiring)
- Features built from scratch: 5 graph workflows (`pokemon profile`, `pokemon evolution`, `pokemon matchups`, `pokemon moves`, `team coverage`)

## Findings

### 1. Novel features were documented but not emitted as commands (missing scaffolding)
- **What happened:** The generated PokeAPI CLI advertised graph workflows in root help, README, SKILL.md, `which`, and manifest-facing research, but the actual command tree did not include `pokemon` or `team` commands after generation.
- **Scorer correct?** Yes. Dogfood's `novel_features_check` correctly reported 2/5 found initially and missing `pokemon matchups`, `pokemon moves`, and `team coverage`; live CLI execution also failed with `unknown command "pokemon"` before manual implementation.
- **Root cause:** Generator/templates consume `NovelFeatures` for documentation and discoverability surfaces but do not have a corresponding command-generation path for composed workflow commands.
- **Cross-API check:** This will recur for any CLI whose absorb/transcend manifest includes workflow-style novel features rather than one-to-one OpenAPI endpoints.
- **Frequency:** subclass: APIs where valuable agent features require composing multiple endpoints/resources (graph APIs, catalog/reference APIs, analytics APIs, search/detail APIs).
- **Fallback if the Printing Press doesn't fix it:** Claude/human must hand-write workflow commands every time. The fallback is unreliable because docs can look shippable while the commands do not exist.
- **Worth a Printing Press fix?** Yes. The machine already treats novel features as publish-blocking, so it should either emit executable command stubs/workflow scaffolds or fail generation before publish when planned features have no implementation path.
- **Inherent or fixable:** Fixable. The Printing Press can add a workflow scaffolding contract for novel features, or tighten absorb/dogfood so non-endpoint novel features require implementation artifacts before docs are generated.
- **Durable fix:** Add a `NovelFeature` implementation contract: each approved novel feature must map to either an existing generated endpoint command, a generated workflow command, or an explicit `implementation_status: planned-only` that is excluded from README/SKILL/manifest. For workflow features, emit a command skeleton under `internal/cli/` with argument parsing, JSON output, and TODO-safe failure until implemented, then make dogfood/publish reject TODO skeletons.
- **Test:** Positive: a research.json with `novel_features: [{command:"pokemon matchups"}]` produces a registered `pokemon matchups` command or publish validation fails before docs claim it. Negative: endpoint-only CLIs without novel features still generate no extra workflow commands.
- **Evidence:** Initial dogfood result: `novel_features_check` planned 5, found 2, missing `pokemon matchups`, `pokemon moves`, `team coverage`; CLI smoke before manual implementation returned `unknown command "pokemon"`.

### 2. Dogfood caught the missing features, but publish safety depends on dogfood writing back state (scorer/default gap)
- **What happened:** The intended safety loop worked only after dogfood wrote `novel_features_built` and the manifest was repackaged from verified features. Without that writeback, generated docs could still overclaim planned features.
- **Scorer correct?** Yes. The scorer behavior is correct and valuable: publish validation should fail unless verified novel features are recorded.
- **Root cause:** The generation flow separates planned novel features (`novel_features`) from built novel features (`novel_features_built`), but that distinction is not yet first-class in the end-to-end CLI generation UX.
- **Cross-API check:** Recurs whenever absorb/transcend proposes features that are later pruned, renamed, or implemented under aliases.
- **Frequency:** most APIs that use transcendence/novel feature generation.
- **Fallback if the Printing Press doesn't fix it:** Operators must remember to run dogfood with `--research-dir`, then regenerate/package from the updated research file. Forgetting any step risks stale claims.
- **Worth a Printing Press fix?** Yes.
- **Inherent or fixable:** Fixable by making the pipeline state transition explicit and hard-gated.
- **Durable fix:** Add an end-to-end publish preflight that requires `novel_features_built` to exist when `novel_features` exists, and make package/validate surface the exact remediation command (`printing-press dogfood --dir ... --research-dir ...`). Consider a single `publish package --research-dir` path that reads verified features directly.
- **Test:** Positive: package/validate fails on planned-only novel features with actionable instructions. Negative: zero-novel-feature CLIs and already-dogfooded CLIs pass unchanged.
- **Evidence:** Maintainer guidance explicitly required dogfood writeback; final manifest passed after all five built features were recorded.

### 3. Public no-auth APIs previously emitted auth residue and non-ASCII env names (bug, fixed in 2.3.5)
- **What happened:** Earlier PokeAPI output included `POKÉAPI_BASIC_AUTH`; v2.3.5 regenerated output has `auth_type: none`, no auth env vars, and no accented `POKÉAPI_*` environment names.
- **Scorer correct?** Yes. The earlier failure was a generator/parser/auth-classification bug, not a PokeAPI-specific requirement.
- **Root cause:** API-name normalization and auth inference were not strict enough for public APIs with non-ASCII display names.
- **Cross-API check:** Recurs for APIs with accents/punctuation in names and specs that expose public/no-auth endpoints.
- **Frequency:** subclass: non-ASCII API names; public APIs with misleading security/default auth fragments.
- **Fallback if the Printing Press doesn't fix it:** Operators must grep for invalid env vars and manually patch config/auth surfaces. This is easy to miss and embarrassing in generated output.
- **Worth a Printing Press fix?** Already fixed in v2.3.5; keep regression coverage.
- **Inherent or fixable:** Fixable and fixed.
- **Durable fix:** Keep regression tests that assert env/config identifiers are ASCII-safe and public APIs do not emit phantom auth fields.
- **Test:** Positive: PokeAPI emits `POKEAPI_*` only for generic config/feedback/base URL and no `POKÉAPI_*`. Negative: APIs that genuinely require auth still emit normalized ASCII auth env vars.
- **Evidence:** Final grep found no `POKÉAPI`; final manifest has `auth_type: none` and `auth_env_vars: null`.

### 4. Dead helper functions remain after generation (template cleanup)
- **What happened:** Final dogfood still warned about two dead helper functions: `extractResponseData` and `wrapResultsWithFreshness`.
- **Scorer correct?** Likely yes. Dogfood identified unused helpers after compile/test success; no live functionality appears to depend on them.
- **Root cause:** Generator emits helper utilities broadly even when a specific CLI shape does not need them.
- **Cross-API check:** Likely recurs across many generated CLIs as templates accrete optional helpers.
- **Frequency:** most APIs, depending on which optional features are enabled.
- **Fallback if the Printing Press doesn't fix it:** Harmless but creates noisy dogfood WARNs and makes operators normalize warnings that may hide real issues.
- **Worth a Printing Press fix?** Yes, but lower priority than command/documentation correctness.
- **Inherent or fixable:** Fixable by conditional helper emission or a generated-code dead helper pruning pass.
- **Durable fix:** Emit helpers only when referenced by selected templates, or add an AST-based post-generation cleanup for unused private functions.
- **Test:** Positive: PokeAPI generation has no dead helper warning. Negative: CLIs that use freshness wrappers still retain those helpers.
- **Evidence:** Final dogfood issue list contained only `2 dead helper functions found`.

## Prioritized Improvements

### P1 — High priority
| Finding | Title | Component | Frequency | Fallback Reliability | Complexity | Guards |
|---|---|---|---|---|---|---|
| F1 | Novel features documented but not emitted as commands | Generator templates + dogfood/publish gates | API subclass: composed workflow features | Low — docs can overclaim while commands are absent | Medium/Large | Only activate for `novel_features`; skip endpoint-only features already mapped to generated commands |
| F2 | Built-feature writeback should be a first-class publish gate | Binary/scorer (`dogfood`, `publish package`, `publish validate`) | Most transcendence runs | Medium — easy to forget `--research-dir` or package stale research | Medium | Only require when planned `novel_features` exists |

### P2 — Medium priority
| Finding | Title | Component | Frequency | Fallback Reliability | Complexity | Guards |
|---|---|---|---|---|---|---|
| F4 | Dead helper functions remain after generation | Generator templates/post-processing | Most APIs | Medium — warning noise is survivable but erodes signal | Small/Medium | Preserve helpers referenced by enabled templates |

### P3 — Low priority
| Finding | Title | Component | Frequency | Fallback Reliability | Complexity | Guards |
|---|---|---|---|---|---|---|
| F3 | ASCII-safe env names and no-auth public APIs | Parser/generator auth normalization | API subclass | High now that v2.3.5 fixed it | Small regression coverage | Keep non-ASCII/public API regression tests |

## Work Units

### WU-1: Make novel feature implementation explicit (from F1)
- **Goal:** Prevent README/SKILL/root help/manifest from claiming novel commands that are not registered in the CLI.
- **Target:** Generator novel-feature handling in `internal/generator/`, plus dogfood/publish checks in `internal/pipeline/dogfood.go` and `internal/cli/publish.go`.
- **Acceptance criteria:**
  - positive test: a research fixture with `pokemon matchups` either emits a registered command or publish validation fails before packaging.
  - negative test: endpoint-only CLIs without novel features do not get placeholder workflow commands.
- **Scope boundary:** Does not require the generator to synthesize domain-specific graph logic perfectly; it only requires truthful implementation state and command registration.
- **Dependencies:** none.
- **Complexity:** large.

### WU-2: Add a verified-novel-feature publish path (from F2)
- **Goal:** Make `novel_features_built` writeback and consumption impossible to skip in publish flows.
- **Target:** `publish package`, `publish validate`, manifest generation, and help text around `dogfood --research-dir`.
- **Acceptance criteria:**
  - positive test: planned novel features without `novel_features_built` fail with an actionable remediation command.
  - negative test: CLIs with no planned novel features and CLIs with verified built features continue to pass.
- **Scope boundary:** Does not change how dogfood matches aliases beyond existing behavior.
- **Dependencies:** none.
- **Complexity:** medium.

### WU-3: Conditional helper emission/pruning (from F4)
- **Goal:** Remove dead helper warnings from clean generated CLIs.
- **Target:** Helper template emission in `internal/generator/` or a post-generation cleanup pass.
- **Acceptance criteria:**
  - positive test: PokeAPI-like generation has no `extractResponseData` / `wrapResultsWithFreshness` dead helper warning.
  - negative test: freshness-enabled CLIs still compile and retain required helpers.
- **Scope boundary:** Private helper cleanup only; does not refactor public generated command behavior.
- **Dependencies:** none.
- **Complexity:** medium.

### WU-4: Preserve PokeAPI auth/name regression tests (from F3)
- **Goal:** Ensure v2.3.5's ASCII env/name and public no-auth behavior does not regress.
- **Target:** OpenAPI parser/auth inference tests and generator/config tests.
- **Acceptance criteria:**
  - positive test: PokeAPI emits no `POKÉAPI_*` identifiers and has no auth env vars.
  - negative test: an API requiring auth emits normalized ASCII env vars.
- **Scope boundary:** Regression coverage only; no new auth schemes.
- **Dependencies:** none.
- **Complexity:** small.

## Anti-patterns
- Treating `novel_features` as documentation-only while publish validation treats them as shippable product claims.
- Allowing a generated CLI to look impressive in README/SKILL/root help before the command tree proves those commands exist.
- Normalizing scorer WARNs as acceptable when they are actually template cleanup opportunities.

## What the Printing Press Got Right
- v2.3.5 fixed the PokeAPI name/auth issue: no accented env vars and no phantom auth requirement.
- Dogfood's novel feature check caught the core overclaim before publish.
- Publish validation now has a meaningful transcendence gate and passed once verified features were recorded.
- The generated endpoint surface, MCP tool count, docs, and basic CLI quality gates worked after the graph workflow gap was closed.

## Issue Gate
There are actionable Printing Press findings (F1, F2, F4, F3 regression coverage), so a GitHub issue is warranted if the operator chooses to submit. This local run intentionally did not upload artifacts or create an issue.
