# archive-is-pp-cli — Deferred Improvements

Notes from the 2026-04-10 dogfood session. Matt is going to come back to these.

## 1. Conversational post-action UX (was: "open in browser")

**Problem:** The current CLI is too curt. After `read <url>` it prints a URL and a few metadata lines. The user wants to read the article — they shouldn't have to switch apps, paste, or remember secondary commands.

**New goal (from Matt's feedback):** Treat the terminal as a brief conversational interface. After any successful read, the CLI should present a menu that matches the user's actual intent.

### Case A: Archive found (happy path)

```
Found it on archive.is — paywall should be skipped.
  https://archive.md/20260410221519/https://www.wsj.com/...
  captured 2026-04-10 22:15:19 via archive.ph

Copied to clipboard. What now?
  [o] open in browser (default)
  [t] tl;dr — summarize with Claude
  [r] read full text here
  [q] quit
>
```

Single-keystroke menu, no Enter needed. Defaults to browser on Enter.

### Case B: Not archived yet (submit + wait flow)

```
Not archived yet. Submitting to archive.is (typically 30-120 sec)...
  → check back in ~2 minutes

What now?
  [w] wait here with a progress spinner
  [b] run in background, notify when ready
  [q] quit, I'll check later with `request check <url>`
>
```

- "Wait here" → foreground spinner, poll timegate every 10s, on ready drop into Case A menu
- "Background" → fork a detached goroutine / spawn a helper that waits, then uses `terminal-notifier` (macOS) / `notify-send` (Linux) / console bell when ready
- "Quit" → store the request ID in a local state file so `request check` knows about it

### Case C: Rate-limited / archive.is blocked

```
Archive.is rate-limited my IP across all mirrors.
Wayback has a snapshot but it's a known hard-paywall domain (WSJ) so it's probably just the teaser.

Options:
  [m] submit manually — open archive.ph in browser with the URL pre-filled
  [b] open Wayback's teaser anyway
  [w] wait 15 minutes and I'll auto-retry
  [q] quit
>
```

### Implementation sketch

```go
// promptMenu shows a single-keystroke menu with labeled options.
// Returns the selected key rune. Respects non-interactive flags.
func promptMenu(prompt string, options []menuOption, defaultKey rune) (rune, error) {
    // ... raw mode read, ESC to cancel, Enter to accept default
}

type menuOption struct {
    Key    rune
    Label  string
    Action func() error
}
```

Menu renderer uses stderr so stdout stays clean for piping. When `--json`, `--quiet`, `--agent`, or stdin-not-TTY, skip the menu and just print the URL as today.

### tl;dr integration

"tl;dr here" means: fetch article body via `get`, pipe through an LLM, print the summary. Needs:
- Env var `ANTHROPIC_API_KEY` (or fallback to `OPENAI_API_KEY`, or shell out to `claude` CLI if installed)
- A ~200-token prompt: "Summarize this article in 3 bullet points and 1 headline"
- Print the summary, then re-show the menu (user may still want to open in browser)
- Graceful fallback if no LLM available: "tl;dr requires ANTHROPIC_API_KEY or the `claude` CLI. Install: ..."

### Background mode (Case B)

Two approaches:
- **(a) Detached child process:** spawn `archive-is-pp-cli request check <url> --wait --notify` as a background process. Parent exits immediately. Child polls, then fires a desktop notification when ready.
- **(b) LaunchAgent/systemd-user job:** overkill for one shot, but cleaner.

Go with (a). Use `terminal-notifier` on macOS, `notify-send` on Linux, `msg` on Windows.

### Flags
- `--no-menu` — skip the post-action menu (current default behavior)
- `--menu` — force the menu even when output looks non-interactive
- `--action open|tl;dr|read|none` — pre-select the action, skip the menu (for scripts and LLM-driven usage)

**Implementation sketch:**

```go
// openInBrowser opens a URL in the default browser. Cross-platform.
func openInBrowser(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    case "linux":
        cmd = exec.Command("xdg-open", url)
    case "windows":
        cmd = exec.Command("cmd", "/c", "start", url)
    default:
        return fmt.Errorf("unsupported platform")
    }
    return cmd.Start()
}

// Detect interactive mode
func isInteractive(flags *rootFlags) bool {
    if flags.asJSON || flags.quiet || flags.agent {
        return false
    }
    // Check if stdout is a TTY
    if fi, err := os.Stdout.Stat(); err == nil {
        return (fi.Mode() & os.ModeCharDevice) != 0
    }
    return false
}
```

Status line formatting:
```
https://archive.md/20260410221519/...
  captured 2026-04-10 22:15:19 via archive.ph
  copied to clipboard
  opening in browser
```

## 2. Warn when Wayback gives a paywalled-teaser result

**Problem:** When `read` falls back to Wayback for WSJ, NYT, FT, Bloomberg, Economist, etc., the Wayback snapshot is often just the paywall teaser (first 3 paragraphs) because Wayback captures after JS runs. User thinks the paywall bypass failed, but the real issue is that we need archive.is specifically for hard paywalls (archive.is strips JS before capture).

**Fix:** When `read` / `get` falls back to Wayback AND the URL is on a known hard-paywall domain, print a warning to stderr:

> Wayback snapshots of WSJ articles usually only show the teaser. For full article text, try:
>   archive-is-pp-cli save <url>
> to force a fresh archive.is capture that strips JS and bypasses the paywall.

**Hard-paywall domain list:**
- wsj.com
- nytimes.com
- ft.com
- bloomberg.com
- economist.com
- theatlantic.com
- newyorker.com
- wapo.st, washingtonpost.com
- businessinsider.com
- barrons.com
- marketwatch.com (sometimes)
- foreignaffairs.com
- hbr.org

Maintain list in a `var hardPaywallDomains = map[string]bool{...}` or as a regex.

## 3a. Actually handle rate limits (don't just report them)

**Problem:** Right now my CLI submits via `GET /submit/?url=...` with no cookie handling, no backoff, no retry. When archive.is 429s once, it gives up. During dogfood testing tonight we exhausted the per-IP submit quota after ~6 submit attempts, and every subsequent call failed for ~1 hour.

**Context:** Archive.is is NOT using Cloudflare bot management. It's a simple per-IP quota on `/submit/` with a 1-hour cooldown (evidenced by the `qki=<id>; Max-Age=3600` cookie on 429 responses). The timegate and timemap endpoints are NOT throttled — only submit.

**Layered fixes, cheapest first:**

### 3a.1 Cookie preservation (5 lines, marginal)
Visit `/` first, capture the `qki=` cookie, include it on submit. Archive.is might treat "users with a cookie" as lower-priority-for-throttling than raw-curl requests. Worth trying, low confidence.

### 3a.2 Exponential backoff with jitter (20 lines, medium)
On 429, don't immediately try the next mirror — they all share the same backend rate limit. Instead:
- First 429: wait 5 sec, retry same mirror
- Second 429: wait 15 sec, retry
- Third 429: wait 60 sec, retry
- Give up after that with a clear message
Applies to `save`, `request`, and the lookup path in `read` when it falls through to submit.

### 3a.3 Quota state file (30 lines, high impact)
Remember the 429 across runs. Write to `~/.local/share/archive-is-pp-cli/rate-limit.json`:
```json
{"last_429_at": "2026-04-11T01:30:00Z", "cooldown_until": "2026-04-11T02:30:00Z"}
```
Before any submit, check the file. If `cooldown_until` is in the future, skip the submit entirely and go straight to alternatives (Wayback, browser handoff, "try later"). This prevents the hammer-the-wall loop that makes things worse.

### 3a.4 Tor onion endpoint fallback (50 lines + Tor dependency)
Archive.is publishes a .onion endpoint: `archiveiya74codqgiixo33q62qlrqtkgmcitqx5u2oeqnmn5bpcbiyd.onion`. Different IP path, probably different rate limit budget. If the user has Tor installed (`brew install tor` / `apt install tor`), add a `--via-tor` flag that routes through `socks5://127.0.0.1:9050`.

Graceful: detect Tor at startup (connect check on 9050), offer as an option in the menu when archive.is is throttling.

### 3a.5 Headless browser submit (chromedp, 200 lines, heavy)
Last resort for power users: bundle chromedp (or shell out to Playwright). When rate-limited, spawn a headless Chrome, visit archive.ph/submit/?url=X, wait for the capture, extract the memento URL. This WILL work, always, but adds 60MB of Chrome to the dependency tree. Gate behind `--use-browser` flag so the default install stays lean.

### 3a.6 Browser handoff (0 lines, current fallback)
When all automated paths fail, open archive.ph in the system default browser with the URL pre-filled. Browser traffic has a completely separate rate-limit budget. Already sketched in item #1 under "Case C".

**Recommended combo for next session:** 3a.1 + 3a.2 + 3a.3 + 3a.6. Skips Tor and chromedp for now. Together they should handle 95% of real-world rate-limit situations without adding dependencies.

## 3b. Better rate-limit error reporting

**Problem:** When submit fails across all six mirrors with 429, the error message says `submit failed: https://archive.vn: rate limited` — implying only vn was rate-limited. Actually all six were tried and all six 429'd, but `lastErr` only shows the last one.

**Fix:** Collect all errors and report them cleanly:

```
submit failed: all six archive.today mirrors rate-limited my IP
  archive.ph:  HTTP 429
  archive.md:  HTTP 429
  archive.is:  HTTP 429
  archive.fo:  HTTP 429
  archive.li:  HTTP 429
  archive.vn:  HTTP 429

Wait 15-60 minutes or submit manually at https://archive.ph/ in your browser.
```

Also track the cooldown hint — archive.is sets a `qki=` cookie with `Max-Age=3600` on 429 responses. Surface that: "Archive.is suggests a 1-hour cooldown for this IP."

## 4. Detect "archive exists but is useless" case

**Problem:** NYT + DataDome example. Someone archived a URL earlier, but DataDome returned a 403/redirect page, so archive.is stored garbage. When user hits that URL, archive.is's resolver silently redirects to the closest real snapshot (e.g., the homepage). User gets confused: "I asked for an article and got the homepage."

**Fix options:**
- **(a) Detect via heuristic:** After `read` returns a snapshot, do a HEAD request on the memento URL and check if it redirects to a different path than the one requested. If so, warn: "The snapshot exists but archive.is served it as a redirect to the homepage. The original capture likely hit a bot wall. Try: `archive-is-pp-cli save <url> --force` to re-archive."
- **(b) Detect at submit time:** When forcing a fresh capture, fetch the result and check if the title contains the hostname ("The New York Times") but the original URL had a specific article path. If so, flag that the capture looks like a bot-wall redirect.
- **(c) Simpler:** just surface the memento's captured timestamp + original URL so the user can see the mismatch.

## 5. Background-submit error propagation for `request`

**Problem:** `request <url>` fires submit in a goroutine and returns "PENDING" immediately. If the submit fails (429, network, auth), the goroutine's error is lost. `request check` later just polls timegate and sees nothing, so it reports "PENDING" forever.

**Fix:** Persist the submit's result (success or error) to a small state file (e.g., `~/.local/share/archive-is-pp-cli/requests.json`) so `request check` can report:
- "PENDING" if submit is still running
- "FAILED: archive.today rate-limited the submit" if goroutine errored
- "READY" if snapshot is live

State file schema:
```json
{
  "https://www.wsj.com/...": {
    "submitted_at": "2026-04-11T01:07:00Z",
    "status": "failed",
    "error": "all mirrors rate-limited",
    "memento_url": ""
  }
}
```

## 6. URL shortcut at top level — generalize to the generator

**Already shipped in this CLI** (`cmd/archive-is-pp-cli/main.go`): `archive-is-pp-cli <url>` is rewritten to `archive-is-pp-cli read <url>`.

**Followup:** add this pattern to the Printing Press generator as an opt-in feature. Any CLI where the primary argument is a URL or identifier could have a "default command" shortcut. Candidates:
- A spec field: `default_command: read` or `url_shortcut: true`
- The generator inserts a shortcut block in `main.go` that routes top-level non-flag args to that command
- Useful for: archive-is (read), redfin (lookup), espn (scores)
- Don't enable by default — only when the CLI has a clear hero command

## 7. Offer a "copy the article text" option distinct from "copy the URL"

**Idea:** Some users want the URL to paste into notes/docs. Others want the actual article text pulled into their clipboard so they can paste it into a message, doc, or LLM prompt. Two different flags:
- `--copy-url` (current default)
- `--copy-text` — fetch via `get`, copy extracted text to clipboard
- `--copy-both` — URL in stdout, text in clipboard

## 8. DataDome is real, but archive.is still works against it sometimes

**Discovery (2026-04-11, corrected after Matt pushed back):** Originally I hypothesized that NYT and WSJ were fully blocked by DataDome and archive.is couldn't capture them. That was wrong. The accurate story is more nuanced:

**What's true:**
- Both NYT and WSJ return `HTTP 401/403 x-datadome: protected` to my curl from my home IP with a Chrome UA
- Archive.is's server-side scraper hits the same DataDome wall from its own IP
- On Matt's WSJ article today, archive.is's scraper genuinely couldn't capture — blank right panel forever, 429s on submit across all mirrors for a full ~8 minutes
- The NYT Iran oil article yesterday (a hypothetical URL I typed, maybe not a real article) silently redirected to the homepage when opened — Unit 8's silent-redirect detection was built for this case

**What's ALSO true and why my "NYT is blocked" conclusion was wrong:**
- Archive.is has **12,312 historical snapshots** of nytimes.com/ back to 1996 (verified via the `history` command). Someone is capturing NYT successfully.
- Matt's Iran peace talks NYT URL worked great today because the snapshot **already existed** in archive.is's database (captured earlier this morning at 08:06:59). The CLI found it via timegate, no submit needed.
- Archive.is's scraper DOES sometimes get past DataDome — different IPs, different fingerprints, different times. The success rate is non-zero even if it's not 100%.
- Once a good snapshot exists, it's indistinguishable from a direct browser capture. It's only the *first* submit for a never-before-archived URL that's a gamble.

**The corrected mental model:**

1. Archive.is **is** the primary for NYT, WSJ, FT, Bloomberg — it has decades of successful captures and is sometimes the only place the full article lives
2. Existing snapshots are reliable: timegate lookup is cheap and the result is real content
3. Fresh submits of never-captured URLs are a gamble — DataDome may block archive.is's scraper, storing either nothing or a 401 page keyed to the requested URL
4. Wayback Machine is a useful fallback when archive.is has NOTHING for the URL, not a replacement when archive.is *might* have a bad capture
5. The user's UX should still try archive.is first, fail loudly when it fails, and offer alternatives — not route around archive.is based on heuristics

**Matt's principle (this is product direction, not just a technical note):**

> "i want you to still TRY and communicate. if you fail don't give up or say 'sorry can't do this that'd suck'"

The CLI should **always try the thing that usually works, fail explicitly when it fails, and offer alternatives.** It should NOT pre-emptively skip archive.is for a domain because we think it's likely to fail. Let the user decide. "Route around" is the wrong instinct. "Try loudly, fail clearly, offer paths forward" is the right instinct.

This supersedes yesterday's Unit 6 design choice where the paywall warning suggested a specific alternative (`save`). For DataDome sites, `save` is a gamble that's likely to burn the user's submit quota. But the answer isn't "don't try save, go straight to Wayback." The answer is: **try `read`, which already does lookup-before-submit, and if nothing exists, tell the user clearly what happened and what their options are.**

**Proposed fixes (revised):**

### 8a. Pre-submit DataDome probe (keep the idea but reframe it)

Before burning 8 minutes of backoff/retry on a submit that's doomed, HEAD the original URL from our own IP first. If it returns `x-datadome` or `server: DataDome` AND the URL has no existing archive.is snapshot, tell the user upfront:

```
Checked https://www.wsj.com/... from your IP: HTTP 401 DataDome
This URL has never been archived before, and WSJ is blocking scrapers at the source.
Fresh submit may fail. Your options:
  [1] Try anyway (8 minutes of backoff if it hits rate limits)
  [2] Read what Wayback has (may be a teaser, but it's what exists)
  [3] Submit via archive.ph in your browser (separate rate-limit bucket, may get lucky)
  [4] Skip — I'll give you nothing
```

Don't decide for the user. Give them the information and the three paths.

### 8b. SCRAPPED — do NOT split the paywall list into Tier A/Tier B

Yesterday's instinct to "skip archive.is for DataDome sites and use Wayback instead" was wrong. Archive.is has more (and often better) captures of NYT and WSJ than Wayback does. Route around only when we KNOW the result will fail — don't guess based on domain.

### 8c. Unit 6 warning stays, but don't make it prescriptive

Current Unit 6 warning tells the user to try `save`. For DataDome sites, `save` is a gamble. Reword the warning to be informational, not prescriptive:

"This is a hard-paywall domain. Archive.is has the most complete captures of these sites but sometimes fails on never-before-archived URLs. Wayback may only show the paywall teaser. Options: try `save` (may fail with rate-limiting), submit via archive.ph in your browser, or use `tldr` on the current Wayback snapshot."

The user picks.

### 8d. Distinguish "source-blocked" from "rate-limited" in the error message

Still a good fix. When all 6 mirrors return 429 on submit AND the URL's host is on the known-hard-paywall list, the error message should say:

"Submit failed. This could be: (a) your IP is rate-limited by archive.today (wait 60 min), (b) the source site (wsj.com) is blocking archive.today's scraper at the edge — unrelated to your IP. Either way retry-in-place won't help quickly. Alternatives: submit via archive.ph in your browser, read the Wayback fallback, or wait."

### 8e. Meta-insight for the Printing Press research phase

Still valid: when researching an archival service for `/printing-press`, flag which source domains have DataDome/Akamai/PerimeterX. This influences the product thesis but doesn't determine it — "archive.is sometimes beats DataDome" is a real capability even when the technical explanation is "via IPs DataDome hasn't blocked yet."

## 9. Show the predictive archive.ph URL immediately on submit

**Discovery (2026-04-11):** Archive.is's web form gives you a URL immediately on submit (even before capture completes) because the form does a client-side redirect to a pending-capture page. My CLI's `submitCapture` waits for response headers (Refresh/Location/Content-Location), which only appear when the submit actually gets an HTTP 200. If submit is slow or stuck in 429 backoff, the user sees nothing at all.

**Fix:** Generate and print the predictive URL the moment submit starts:

```
Submitting to archive.today...
  URL when ready: https://archive.ph/<predicted-timestamp>/<orig_url>
  Check status:   https://archive.ph/<orig_url>
```

The `archive.ph/<orig_url>` form is archive.is's "all snapshots for this URL" page — it 404s until a snapshot exists, then shows the list. Users can refresh it in a browser to see when the capture lands.

The predicted timestamp can be `time.Now().UTC().Format("20060102150405")` — the actual archive.is timestamp may differ by a few seconds but it's close enough to set expectations.

**User quote:** "doesn't it generate a URL even if it's not ready"

Yes, the web UX does. My CLI's UX doesn't. Fix it.

## 10. The 180-second submit timeout doesn't actually fire (discovered 2026-04-11 Phase B)

**Discovery:** During Phase B test of the WSJ article after cooldown cleared, the `save` command ran for 9+ minutes before Matt killed it. The CLI set a 180-second timeout via `newArchiveHTTPClient(180*time.Second)` but the process sat there doing nothing with CPU 0.0 for 9+ minutes. Archive.is had opened a TCP connection and was keeping it alive without sending any response.

**Hypotheses:**
- Go's `http.Client.Timeout` is supposed to cut off the entire request (dial + write + read). Either that's not what it's actually doing, or my code path has a bug that resets/ignores the timeout.
- Alternative: the `newNoRedirectClient` helper is being used instead of `newArchiveHTTPClient` in some code paths. The two may have different timeout enforcement.
- Alternative: the backoff loop in `tryMirrorWithBackoff` is adding up individual timeouts such that total elapsed can far exceed the per-request value. In that case the issue isn't the timeout not firing — it's that the UX of "this can take 30-120 seconds" is wildly optimistic when we're doing 6 mirrors × 4 attempts × 180s each.

**Fix:**
1. Use `context.WithTimeout` on the HTTP request, not just the client timeout — Go's client timeout has known edge cases around slow-reading responses. A context cancellation is more reliable.
2. Cap the total submit budget end-to-end at, say, 5 minutes. After that, give up with a clear "archive.today took too long, the capture may still complete on their side — check with `request check` in a few minutes" message.
3. Show live progress to the user — print a `.` or elapsed time on stderr every 10 seconds so it doesn't look frozen.
4. Potentially retry with the `--incomplete` semantics that HRDepartment's CLI supports (non-blocking submit that returns immediately).

## 11. Wayback availability API is unreliable — use the CDX API instead (discovered 2026-04-11 Phase A)

**Discovery:** The Phase A capture-quality test found that `tldr` fails for 15 of 20 URLs with "no wayback snapshot available," but the Wayback CDX API confirms all 15 URLs ARE in Wayback's index. The `availability` endpoint is returning empty for URLs that definitely have captures.

**Evidence from test session:**
- `curl -s "https://archive.org/wayback/available?url=https://www.bbc.com/news"` → `{"archived_snapshots": {}}`
- `curl -s "https://web.archive.org/cdx/search/cdx?url=https://www.bbc.com/news&output=json&limit=3"` → returns snapshots since 1999
- Same pattern for Simon Willison, Wikipedia Go, and dozens of other sites
- Occasionally the availability API DOES return the snapshot for the same URL (inconsistent within minutes)
- Stripping `www.` prefix or omitting `https://` scheme sometimes changes the result (URL canonicalization is fragile)

**Fix (this is the single highest-value item in this notes file):** Replace `waybackLookup` in `internal/cli/read.go` to use the CDX API instead of the availability API.

Current endpoint:
```
GET https://archive.org/wayback/available?url=<X>
```

Replacement:
```
GET https://web.archive.org/cdx/search/cdx?url=<X>&output=json&limit=-1&filter=statuscode:200&fl=timestamp,original
```

Parse the JSON result (array of arrays), pick the most recent entry (last row since CDX is chronological by default with `limit=-1`), construct the memento URL as `https://web.archive.org/web/<timestamp>/<original>`.

**Estimated impact:** Phase A test was 5/20 on tldr. With CDX, expected result is ~17/20 (genuine misses only on sites Wayback has never crawled).

**Estimated effort:** ~30 lines of Go. Same function shape, different endpoint and response parser.

**Also add:** URL canonicalization before querying — try with the original URL, then with trailing slash stripped, then with `www.` stripped if present. The CDX API is less picky than availability but still benefits from normalization.

## Priority (Matt's call)

None are blockers. The CLI works. Order by impact:

1. **#1 (open in browser)** — biggest UX win, 30 lines
2. **#2 (paywall domain warning)** — prevents "why did this fail" confusion, 20 lines
3. **#3 (better rate-limit reporting)** — improves debuggability, 15 lines
4. **#5 (request state persistence)** — completes the async workflow, 50 lines
5. **#4 (archive-is-redirect detection)** — hardest, fragile heuristic, 80 lines
6. **#6 (generator URL shortcut)** — machine improvement, compounds across CLIs
7. **#7 (copy-text option)** — nice but niche
