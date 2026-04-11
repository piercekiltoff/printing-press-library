---
title: "test: verify archive-is-pp-cli capture quality across non-WSJ/non-NYT sites"
type: test
status: completed
date: 2026-04-11
---

# Test Plan — Archive Capture Matrix

**Target repo:** archive-is-pp-cli at `~/printing-press/library/archive-is/`. This plan runs and verifies the installed binary; it does not modify CLI source code. Findings feed into a follow-up implementation plan if gaps are found.

## Overview

We built the archive-is CLI believing archive.today was the best path for paywall bypass on hard-paywall sites. Today's WSJ debug session surfaced a DataDome wall that blocks archive.is's scraper at the source. Matt's correction: the CLI DOES sometimes capture NYT and WSJ successfully — the Iran peace talks NYT URL worked great this morning because a pre-existing snapshot was already in archive.is's database.

The open question: **does the CLI work reliably for the vast majority of sites that aren't NYT/WSJ/DataDome-protected?** Or are there subtler failure modes we haven't caught?

This plan runs the CLI against a deliberately diverse URL matrix (15-20 URLs across 6-7 categories), records results, and delivers a report with manual-verification links so Matt can spot-check each claimed success. The goal is to surface failures in the cheap path (timegate lookup) and in the Wayback fallback before we assume the feature "works" for everything-but-WSJ.

## Problem Frame

We have two data points and no clear picture between them:

- **Datum 1:** WSJ submit failed hard today. 6 mirrors × 4 attempts = 24 429s, cooldown triggered. Blank right panel in browser. DataDome at the source blocking both my CLI and archive.is's own scraper.
- **Datum 2:** NYT Iran peace talks URL worked great via timegate (pre-existing snapshot from 08:06:59 this morning). Real article content, opened cleanly in the browser.

From these two data points, we've drawn inconsistent conclusions over the last few hours:
- "Archive.is is broken for DataDome sites" (wrong — NYT works sometimes)
- "Archive.is works great" (incomplete — we only tested one URL)
- "Split the paywall list into Tier A/B" (withdrawn — Matt's principle says don't route around, try and fail loudly)

**The actionable question:** what's the success rate of the CLI across typical web URLs? Where are the silent-failure modes? Is the Wayback fallback doing meaningful work or is it mostly teaser-only? Does our Unit 8 silent-redirect detection actually catch the DataDome-stored-a-homepage case, or does it miss some?

The only way to answer any of this is to run the CLI against a diverse, verifiable set of URLs and have a human check each result. That's what this plan does.

## Requirements Trace

- **R1.** Cover at least 15-20 URLs across 6-7 categories: open web, mainstream news (no paywall), tech news, soft-paywall, JS-overlay paywall, DataDome-protected (control), non-English/non-US.
- **R2.** For each URL, record which backend served the result (archive-is or wayback), whether submit was needed, and the final memento URL.
- **R3.** For each success, verify the captured content is the real article — not a homepage, not a paywall teaser, not a bot-wall page — via either `tldr` (LLM summarization) or `get` (text extraction).
- **R4.** For each failure, classify the failure mode: rate-limited, silent-redirect, CAPTCHA, no snapshot found, other.
- **R5.** Produce a report Matt can spot-check: clickable archive URLs plus a one-line "expected content" hint per URL, so he can open 3-5 at random and verify they show the real article.
- **R6.** Respect the submit rate-limit cooldown. Tests that require fresh submits run ONLY after the cooldown window clears (earliest around 10:09 local this morning; check the state file before running).
- **R7.** The test itself must run entirely via the public CLI interface. No internal-API probing, no test-only shortcuts.

## Scope Boundaries

- **In scope:** Running the installed `~/go/bin/archive-is-pp-cli` against a URL matrix, capturing outputs, writing a test report.
- **Out of scope:** Modifying the CLI. Any bugs found here become a follow-up implementation plan.
- **Out of scope:** Testing WSJ or NYT specifically. The whole point is to understand what works OUTSIDE those two sites. One WSJ URL and one NYT URL are included as controls, nothing more.
- **Out of scope:** Testing edge cases of individual flags (`--copy-text`, `--dry-run`, menu interaction). Those are unit-test territory. This is an integration/smoke test of the hero workflow.
- **Out of scope:** Testing the MCP server. Different surface, separate concern.
- **Out of scope:** Load-testing or concurrent-submit tests. One URL at a time, sequential, with respect for rate limits.

## Context & Research

### Relevant Code and Patterns

- `internal/cli/read.go` — `newReadCmd()` timegate-first flow, Wayback fallback on no-snapshot, silent-redirect detection (Unit 8).
- `internal/cli/paywall_domains.go` — hard-paywall domain list; influences the warning shown when Wayback is used.
- `internal/cli/rate_limit.go` — persistent cooldown state at `~/Library/Application Support/archive-is-pp-cli/rate-limit.json`. Check this before running submit-dependent tests.
- `internal/cli/silent_redirect.go` — `detectSilentRedirect()` heuristic for bot-wall captures. This is what caught the NYT Iran oil issue yesterday.
- `internal/cli/tldr.go` — `runTldr()` pipeline: find archive → fetch body → LLM summarize. Used for content verification.

### Institutional Learnings

- **Submit rate limit is per-IP, 1 hour cooldown.** Documented in `.notes/improvements.md` and in the CLI's state file. Triggered this session around 08:09 local; earliest retry ~10:09.
- **DataDome blocks archive.is's scraper for some sites.** NYT and WSJ both return `x-datadome: protected`. Once a good snapshot exists, timegate lookup still works — the block affects only fresh submits.
- **Wayback API expects URLs unencoded.** Past bug. Fixed in `waybackLookup()`. Relevant if the test matrix includes URLs with unusual characters.
- **Archive.is serves CAPTCHAs for direct body fetches.** The `get` command falls back to Wayback on CAPTCHA detection. Silently, by design. Makes it hard to tell whether a `get` result came from archive.is or Wayback without inspecting stderr.

### External References

- Memento Protocol RFC 7089 — timegate/timemap semantics.
- ArchiveTeam wiki on archive.today — documents the per-IP quota and common failure modes.

## Key Technical Decisions

- **Test via the installed binary at `~/go/bin/archive-is-pp-cli`, not a fresh build.** The goal is to verify what real users experience. If Matt runs `archive-is-pp-cli <url>` himself, he's running the installed binary. Test the same thing.
- **Two phases: Phase A (timegate-safe) runs now; Phase B (fresh submits) runs after cooldown.** Phase A exercises `read` / `get` / `tldr` / `history` on URLs that likely have existing snapshots or fall through to Wayback. Phase B exercises `save` on URLs likely NOT in archive.is yet, so we see the full submit path. Splitting lets Phase A produce actionable results within minutes instead of being gated on rate-limit recovery.
- **Verify content via `tldr`, not `get` alone.** `get` returns whatever text was extracted — could be a CAPTCHA page, a homepage, or real content, and the CLI doesn't always know which. Passing the text through Claude gets us a summary that a human can sanity-check: if the summary is about "browser security check, please wait" then the capture failed silently. If the summary is about the article's actual topic, it worked.
- **Record both archive.is and Wayback results when both have snapshots.** Don't just show the first one the CLI returns. For a subset of URLs, explicitly run with `--backend archive-is` and `--backend wayback` to compare what each backend has. This exposes the "archive.is has a silent-redirect, Wayback has the real thing" case.
- **Matt picks the URLs for the "categories he cares about."** I can propose a default matrix but Matt should have a chance to swap in URLs from his own reading list. His manual verification is more meaningful when he actually wants to read the article.
- **Single-session, sequential runs with gaps between requests.** No parallelism. Each CLI invocation respects archive.today's soft rate limits naturally. Enforced by running commands one at a time, not because of artificial delays.
- **Fixed content hints per URL** — the report includes a one-line expected-content summary per URL so Matt can spot-check without reading every article. Example: "Wikipedia — 'Go (programming language)', should mention Google origin, 2009 release, goroutines."

## Open Questions

### Resolved During Planning

- **Which URLs go in the matrix?** A first pass is proposed in Unit 1 below. Matt has the option to swap in his own URLs before Phase A runs. Default matrix is built from stable URLs unlikely to 404 (Wikipedia, AP, BBC, etc.).
- **How many URLs total?** 18 in the default matrix. 3 per category × 6 categories. Small enough to run in one session, large enough to surface patterns.
- **How to verify "real content"?** Run `tldr` on each successful result and include the Claude summary in the report. The summary is the signal — if it's coherent and topical, the capture worked; if it's "please complete the security check" or "welcome to nytimes.com", it failed silently.
- **What about the rate-limit gate?** Read the state file before Phase B runs. If cooldown is still active, skip Phase B and flag it in the report: "Phase B deferred — IP in cooldown until HH:MM."
- **Should Phase A include submit attempts?** No. Phase A is strictly timegate-plus-Wayback. If a URL has no existing archive.is snapshot AND no Wayback snapshot, it shows up as "no snapshot found" — not a fresh submit. That's Phase B's job.

### Deferred to Implementation

- **Exact URLs to use from each category.** The plan proposes categories and approximate URL shapes. Picking specific live URLs happens at execution time so we don't link-rot the plan document.
- **Whether to include Chinese, Japanese, or Arabic URLs.** The matrix has one non-English slot; language choice deferred. Probably a German or French news site for the first pass.
- **Claude CLI vs direct API for tldr verification.** The installed CLI already prefers `claude` on PATH and falls back to API keys. No decision needed here; it'll use whatever's available.

## Implementation Units

- [ ] **Unit 1: Assemble the URL matrix and present it to Matt for approval or edit**

**Goal:** Produce a concrete list of 18 URLs across 6-7 categories, with one-line expected-content hints for each. Give Matt the chance to swap in his own URLs before running anything.

**Requirements:** R1, R5

**Dependencies:** None.

**Files:**
- Create: `docs/tests/2026-04-11-archive-capture-matrix.md` (the test-run artifact — category table with URLs and content hints, before running any CLI)

**Approach:**

**The concrete default matrix (20 URLs).** URLs are biased toward homepages and stable reference pages — dated article URLs risk link rot and the test becomes unreliable when a URL 404s mid-run. Homepages always exist, always have content, and archive.is has captured them for years so timegate will find something. Matt can swap in article URLs from his reading list before Phase A runs.

| # | Category | URL | Expected content hint |
|---|----------|-----|------------------------|
| **Category 1: Open web / no paywall — baseline** |
| 1 | Wikipedia | `https://en.wikipedia.org/wiki/Go_(programming_language)` | Article about Google's Go language, 2009 release, goroutines, static typing |
| 2 | Personal blog | `https://daringfireball.net/` | John Gruber's Daring Fireball front page, Apple commentary, linked list |
| 3 | Personal tech blog | `https://simonwillison.net/` | Simon Willison's blog, LLM tooling, datasette, recent posts |
| **Category 2: Mainstream news, no paywall** |
| 4 | BBC | `https://www.bbc.com/news` | BBC News front page, world headlines |
| 5 | NPR | `https://www.npr.org/` | NPR front page, US+world news mix |
| 6 | Reuters | `https://www.reuters.com/` | Reuters front page, wire service headlines |
| **Category 3: Tech news / blogs** |
| 7 | TechCrunch | `https://techcrunch.com/` | TechCrunch front page, startup + venture news |
| 8 | Ars Technica | `https://arstechnica.com/` | Ars front page, deep tech coverage |
| 9 | GitHub blog | `https://github.blog/` | GitHub product + engineering blog |
| **Category 4: Soft paywall / newsletter / metered** |
| 10 | Guardian | `https://www.theguardian.com/us` | Guardian US section, soft paywall after N articles |
| 11 | Platformer | `https://www.platformer.news/` | Casey Newton's Substack, free teaser + paid subscribers |
| 12 | Stratechery | `https://stratechery.com/` | Ben Thompson, Monday free + paid weekdaily |
| **Category 5: JS-overlay paywall — archive.is's historical sweet spot** |
| 13 | The Atlantic | `https://www.theatlantic.com/` | The Atlantic front page, JS overlay paywall |
| 14 | The New Yorker | `https://www.newyorker.com/` | New Yorker front page, metered + JS overlay |
| 15 | The Economist | `https://www.economist.com/` | Economist front page, hard paywall with JS overlay |
| **Category 6: International / non-English** |
| 16 | Le Monde (French) | `https://www.lemonde.fr/` | Le Monde front page, French, soft paywall |
| 17 | Der Spiegel (German) | `https://www.spiegel.de/` | Der Spiegel front page, German, mixed paywall |
| 18 | NHK (Japanese) | `https://www3.nhk.or.jp/news/` | NHK News Japanese front page, free |
| **Category 7: DataDome controls — expected to show failure modes** |
| 19 | NYT article (real) | `https://www.nytimes.com/2026/04/11/world/middleeast/iran-peace-talks-demands.html` | Iran peace talks demands article (this one worked for Matt today via timegate — verify it still does) |
| 20 | WSJ article | `https://www.wsj.com/world/middle-east/free-seas-iran-strait-hormuz-toll-3404b2e1?mod=WSJ_home_mediumtopper_pos_1` | Iran Hormuz toll article (failed submit today, Wayback has real content — verify the fallback story) |

**Why this matrix and not a different one:**

- **Homepages over articles (mostly):** Homepages never 404, always have content, are commonly archived. This is a smoke test of the CLI's flow, not a stress test of link-rot handling. Articles are higher fidelity but brittle. Matt can swap in articles he's actually reading before Phase A runs.
- **No Bloomberg or WSJ-style-Financial-Times in Category 5:** Bloomberg uses DataDome, same as NYT/WSJ. Putting them in Cat 5 would muddle the "archive.is JS-overlay sweet spot" signal. If we want to test Bloomberg specifically, it belongs in Category 7 with the other DataDome controls.
- **No Medium/Substack article-specific URLs:** Medium's article URLs are long-lived but individual posts vary wildly in whether archive.is has them. Platformer/Stratechery as homepages are a more stable soft-paywall signal.
- **The Economist is a gamble:** It may or may not use DataDome. Worth testing — if it works, it validates the JS-overlay story. If it fails, we learn something.
- **Le Monde, Der Spiegel, NHK for internationalization:** Covers French (Latin-1-ish), German (umlauts in UTF-8), and Japanese (multi-byte). If our `extractReadableText` mangles any of these, we want to know.
- **The two controls are the exact URLs from today's debug session.** Re-running them gives us a "has anything changed in the last hour?" signal and confirms the silent-redirect / DataDome story isn't drifting.

- For each URL: capture the **one-line content hint** (already in the table above). Matt can eyeball the claude summary or the browser preview and say "yes, that matches" or "no, that's wrong."

- Before running any CLI commands, present the matrix to Matt via AskUserQuestion with options:
  - **Run with this matrix** — proceed to Unit 2
  - **I'll swap in my own URLs** — Matt provides 5-10 URLs, plan author fills remaining slots from the default matrix
  - **Add more categories** — e.g., "also include forums/social like Reddit or Lobsters"

- The matrix file is what Unit 2 iterates over. It's also the artifact Matt can take and spot-check independently of Unit 3's report.

**Patterns to follow:**
- `.notes/improvements.md` — similar structured-document format with categorized sections.

**Test scenarios:**
<!-- This unit produces an artifact (the matrix), not code. No test scenarios in the usual sense. -->
- Test expectation: none — non-code artifact. The matrix's quality is judged by Matt's "yes/swap/add" response.

**Verification:**
- The file `docs/tests/2026-04-11-archive-capture-matrix.md` exists and contains 18 URLs across 6-7 categories.
- Each URL has a one-line expected-content hint.
- Matt has either approved the matrix or provided replacements.

---

- [ ] **Unit 2: Run the CLI against each URL and capture structured results**

**Goal:** For every URL in the approved matrix, run the CLI (`read` first, then `tldr` for content verification, optionally `history` for context) and record: which backend served the result, the memento URL, the LLM summary, the raw stderr (for detecting warnings like paywall/silent-redirect), and any errors.

**Requirements:** R2, R3, R4, R6, R7

**Dependencies:** Unit 1.

**Files:**
- Create: `docs/tests/2026-04-11-archive-capture-results.md` (the filled-in results table)
- Reference: `~/Library/Application Support/archive-is-pp-cli/rate-limit.json` (read-only, for Phase B gate check)

**Approach:**

**Phase A — Timegate-safe (runs immediately, no submit dependency):**

For each URL in the matrix, sequentially:

1. Run `archive-is-pp-cli read <url> --no-clipboard --no-prompt --json` and parse the JSON output.
2. Record: `backend`, `memento_url`, whether the CLI emitted a paywall warning or silent-redirect warning (check stderr for `NEXT:` lines vs warning lines).
3. If `read` succeeded (non-empty memento_url), run `archive-is-pp-cli tldr <url> --json` and capture the `headline` and `bullets`. This is the content verification — if the summary is coherent and topical, the capture is real.
4. If `read` failed (no archive.is snapshot AND no Wayback snapshot), mark the URL as "no snapshot" and skip `tldr`.
5. If the URL is on the hard-paywall domain list and `read` fell back to Wayback, also try `archive-is-pp-cli read <url> --backend archive-is --no-clipboard --no-prompt --json` to see if archive.is has anything at all (vs only Wayback knowing the URL).
6. Record everything in the results table.

**Phase B — Fresh submit tests (runs only after cooldown clears):**

1. Before running Phase B, read `~/Library/Application Support/archive-is-pp-cli/rate-limit.json`. If `cooldown_until` is in the future, skip Phase B and mark it "deferred — IP in cooldown until HH:MM local."
2. If cooldown has cleared, pick 2-3 URLs from Phase A that had "no snapshot" results and run `archive-is-pp-cli save <url> --no-prompt --json` on each. These are the fresh-submit test cases.
3. Record submit outcomes: success (with memento URL), rate-limited (write cooldown to state file — the CLI already does this), source-blocked (suspected DataDome), timeout.
4. For any successful submit, re-run `tldr` to verify content.

**For each URL captured, the results row includes:**
- `Category`
- `URL`
- `Backend used` (archive-is / wayback / none)
- `Memento URL` (clickable)
- `tldr summary` (headline + first bullet)
- `Warnings` (paywall warning? silent-redirect warning? submit failure?)
- `Manual verify` (flag for Matt — "expected: X, got: Y")

**Patterns to follow:**
- Existing CLI invocation patterns from the first session's dogfood run.
- Phase-based execution like the Printing Press `/printing-press-polish` skill — Phase A is cheap and always runs, Phase B is gated.

**Test scenarios:**
<!-- This unit runs tests. The "test scenarios" here describe what the test RUN must demonstrate. -->
- Happy path: all 18 URLs return a backend (archive-is or wayback), tldr produces a coherent summary that matches the URL's expected content hint from Unit 1.
- Happy path: Phase A completes in under 10 minutes for 18 URLs, with sequential execution.
- Edge case: a URL returns a backend but tldr produces a nonsense summary ("browser security check, please wait") — flag as "silent failure."
- Edge case: a URL has an archive.is snapshot that silently redirects to the homepage — the CLI's Unit 8 silent-redirect warning fires and is captured in the results.
- Edge case: the CLI's paywall warning fires for a Tier A (JS-overlay) domain — is Wayback actually teaser-only, or coherent enough for Claude to summarize?
- Error path: cooldown state file shows IP in cooldown during Phase B — Phase B is skipped entirely, noted in the results.
- Error path: Claude CLI is not on PATH — `tldr` falls back to Anthropic API; if that's also absent, the row shows "tldr unavailable" rather than crashing.
- Error path: a URL in the matrix is a 404 (link-rot) — the CLI returns an error; results table shows it and recommends swapping the URL.
- Integration: the stderr warning system works end-to-end — we should see at least one paywall warning (for Tier A sites) and possibly one silent-redirect warning (for the NYT/WSJ controls).

**Verification:**
- `docs/tests/2026-04-11-archive-capture-results.md` exists with one row per matrix URL.
- Every row has a result (even if it's "failed" or "deferred").
- Phase A completed; Phase B either completed or is explicitly marked as deferred with a reason.
- At least one row shows a fresh tldr summary produced by running the CLI — not cached from a prior session.

---

- [ ] **Unit 3: Write the spot-check report and surface any patterns**

**Goal:** Produce a short, scannable report based on Unit 2's results that Matt can spot-check in 5-10 minutes. Categorize any failures by type, flag URLs where the result looked suspicious, and summarize what we learned about the CLI's true success rate outside of WSJ/NYT.

**Requirements:** R4, R5

**Dependencies:** Unit 2.

**Files:**
- Create: `docs/tests/2026-04-11-archive-capture-report.md` (the human-readable report)

**Approach:**

The report has three sections:

1. **Scoreboard** — a single-line summary per category: "News (5/5 worked), Tech blogs (3/3 worked), Soft paywall (2/3 worked, 1 silent-redirect), JS-overlay paywall (3/3 worked), International (3/3 worked), DataDome controls (0/2 submitted fresh — cooldown), Wikipedia baseline (3/3 worked)." One line each. Easy to glance.

2. **Spot-check list** — 5-7 URLs hand-picked for Matt to actually open in his browser. Picked from results where:
   - The result looked suspicious (warnings present, or the tldr summary was thin)
   - The category was previously unverified (first run for international, first run for soft paywall)
   - The CLI fell back to Wayback for a non-hard-paywall domain (unusual, worth investigating)

   For each spot-check URL: clickable archive URL, expected content hint from Unit 1, and the claude summary Unit 2 produced.

3. **Failure patterns** — if any URLs failed, classify them:
   - `no-snapshot` — neither backend had anything; suggests the URL is too new and nobody has archived it
   - `silent-redirect` — archive.is has a bot-wall capture keyed to the URL, resolver redirects to homepage (Unit 8 warning fires)
   - `teaser-only` — Wayback has a partial page behind a paywall overlay
   - `rate-limited` — self-explanatory
   - `source-blocked` — the site returns `x-datadome: protected` to archive.is's scraper (suspected on NYT/WSJ)
   - `unknown` — failed for a reason we don't understand yet; this is the interesting bucket

4. **Recommendations** — based on the patterns, what (if anything) should change in the CLI? Examples:
   - "Our assumption that archive.is is the right primary for these 6 categories holds" → done, no changes needed
   - "Soft paywalls X and Y fail silently — the silent-redirect detector should be widened" → new plan item
   - "Non-English sites have garbled text in the tldr output — `extractReadableText` may need UTF-8 awareness" → new bug
   - "Wayback fallback is actually coherent for site Z — the paywall warning is over-aggressive for that domain" → tune the domain list

**Patterns to follow:**
- The shipcheck report from the first session (`proofs/2026-04-10-233336-fix-archive-is-pp-cli-shipcheck.md`) — similar three-section structure, similar "human-scannable scoreboard" pattern.

**Test scenarios:**
<!-- This unit produces a report. No code. -->
- Test expectation: none — non-code artifact. Report quality judged by Matt's ability to spot-check 5 URLs in 10 minutes and either confirm the CLI works or find something new to fix.

**Verification:**
- The report has a scoreboard summarizing results by category.
- The spot-check list has 5-7 clickable archive URLs with expected content hints.
- Any failures are classified.
- Recommendations section either says "no changes needed" or lists specific, actionable follow-up items.
- Matt actually opens 3-5 URLs from the spot-check list and either confirms them or finds a problem.

## System-Wide Impact

- **No code changes.** This plan runs existing CLI commands and writes markdown reports. Zero impact on other parts of the CLI, the MCP server, or the Printing Press generator.
- **Runs against live internet.** Each `read`/`tldr` invocation hits archive.today and possibly web.archive.org and possibly the LLM API. Total network load is ~18 URLs × ~2 API calls each = ~36 HTTP requests plus ~18 LLM calls. Minor cost, mostly on Claude API tokens.
- **Consumes LLM budget.** `tldr` calls Claude (via CLI or API) once per URL. With 18 URLs, that's ~18 summaries × a few thousand tokens each. Measured in cents, not dollars.
- **Respects the persistent cooldown.** Phase B gate check prevents us from hammering a rate-limited API.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Link rot — some matrix URLs 404 by the time we run them | Default matrix uses stable URLs (Wikipedia, AP, BBC). Unit 1 lets Matt swap in fresh URLs before Phase A. Unit 2's scenarios cover the 404 case. |
| Rate limit hits during Phase A (even though Phase A is timegate-only) | Timegate isn't throttled for normal-volume use. 18 requests over ~10 min is fine. If it happens anyway, Phase A results for remaining URLs are deferred, report flags the issue. |
| tldr produces incoherent summaries because the LLM hallucinates | Matt's spot-check in Unit 3 catches this. The whole point of picking 5-7 URLs for manual verification is that Claude summaries aren't ground truth. |
| Wayback fallback succeeds with teaser-only content and Claude summarizes the teaser as if it were the full article | The tldr output for a teaser looks different from a full article — shorter, more generic, often mentions "sign up to read more." Matt's spot-check catches this. We should also note this in the failure patterns section of Unit 3. |
| Phase B can't run today because cooldown doesn't clear in time | That's fine. Phase A alone is 80% of the value. Phase B results can come in a second run tomorrow. Report explicitly marks deferred items. |
| Matt picks URLs he knows archive.is will fail on (bad-faith test) | He's the evaluator. If he picks torture-test URLs, the report shows them failing and we learn the boundaries. That's the point. |

## Documentation / Operational Notes

- **Outputs are three files in `docs/tests/`:** the matrix, the raw results, the scannable report. Matt can keep them or throw them away after reading.
- **No git commits expected.** The archive-is CLI is not a git repo; the docs just sit on disk.
- **Results feed the next plan.** If Unit 3 surfaces patterns like "silent-redirect detection misses case X" or "paywall warning is wrong for domain Y," those become new implementation-plan items (against `cli-printing-press` or `archive-is` depending on scope).
- **Not a regression suite.** This is a one-time smoke test to establish a baseline and find obvious failures. It's not something we run on every CLI change. If we want continuous testing, that's a separate plan.

## Sources & References

- **Origin conversation:** session on 2026-04-11 after debugging the WSJ capture failure. Matt's quote: "I feel like we need to test your ability to archive on a non wsj/nytimes site. I feel like THAT feature needs work."
- Previous session plans:
  - `docs/plans/2026-04-10-001-feat-archive-is-polish-improvements-plan.md` — the initial polish pass (rate limits, UX, silent-redirect detection)
  - `docs/plans/2026-04-11-001-feat-agent-hints-stderr-plan.md` — the agent hints feature
- Notes: `.notes/improvements.md` — especially section #8 (DataDome, corrected) and #9 (predictive URL on submit)
- Related code:
  - `internal/cli/read.go` — `newReadCmd()`, `newTldrCmd()` wiring
  - `internal/cli/silent_redirect.go` — the heuristic this test exercises
  - `internal/cli/paywall_domains.go` — the warning this test probes
