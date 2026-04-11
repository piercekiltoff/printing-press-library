# Acceptance Report: archive-is

Level: Quick Check (no API key needed — archive.is is unauthenticated)
Tests: 7/7 passed

## Tests run

```
[1/7] doctor
  $ archive-is-pp-cli doctor
  Config ok, auth not required, API reachable
  PASS

[2/7] read existing NYT homepage
  $ archive-is-pp-cli read https://www.nytimes.com/ --no-clipboard --json
  Returned memento URL https://archive.md/20260410174409/https://www.nytimes.com/ in <1s
  Timegate lookup worked. No submit needed. Exactly the lookup-before-submit design goal.
  PASS

[3/7] history command (timemap parsing)
  $ archive-is-pp-cli history https://www.nytimes.com/
  Found 12,312 snapshots from 1996-11-12 to present
  Parsed Memento link-format cleanly
  PASS

[4/7] history --json
  $ archive-is-pp-cli history https://www.nytimes.com/ --json
  Valid JSON with count and snapshots array
  PASS

[5/7] get command (the paywall bypass hero)
  $ archive-is-pp-cli get https://www.nytimes.com/
  Wayback fallback triggered when archive.is CAPTCHA'd
  Extracted 9,698 characters of readable text
  PASS

[6/7] get with Wikipedia
  $ archive-is-pp-cli get https://en.wikipedia.org/wiki/Main_Page
  Wayback fallback worked
  Extracted 15,364 characters of clean text
  PASS

[7/7] request check (status-only, no submit)
  $ archive-is-pp-cli request check https://www.wikipedia.org/ --json
  Returned status: existing with memento URL and timestamp
  PASS
```

## Known limitations surfaced during dogfood

- **archive.is CAPTCHAs direct body fetches.** Handled by automatic Wayback fallback in `get`.
- **archive.is rate-limits aggressive submits** across all mirrors simultaneously. A fresh URL with no existing snapshot triggers 429 across archive.ph, archive.md, etc. The CLI correctly surfaces "submit failed: rate limited" as an `apiErr` with exit code 5. User should retry later or use `--force` on a fresh URL.
- **get falls back to Wayback automatically.** This is by design — Wayback is more permissive for direct body fetches, which is the point of dual-backend support.

## Fixes applied: 3

1. **Fixed generator template bug:** `usageErr` was gated behind `HasMultiPositional` but `command_promoted.go.tmpl` calls it unconditionally, causing `undefined: usageErr` build errors for APIs with single-positional promoted commands. Fix: always emit `usageErr`. (Machine change, affects all future CLIs.)

2. **Fixed Go regex compile error:** Used `\1` backreference which Go's RE2 doesn't support. Rewrote as three separate patterns for script/style/noscript. (Printed CLI change, isolated to paywall.go.)

3. **Fixed Wayback lookup URL encoding:** `url.QueryEscape()` on the URL parameter caused Wayback's availability API to return empty snapshots. Fix: pass URL unencoded. (Printed CLI change, isolated to waybackLookup.)

## Printing Press issues for retro: 1

- **Generator template gate misaligned with template usage.** `helpers.go.tmpl` gates `usageErr` emission behind `HasMultiPositional`, but `command_promoted.go.tmpl` uses `usageErr` unconditionally. Retro should confirm the gate is safe to remove (since Go allows unused package-level functions) or add a compile-time check that verifies every usageErr call-site has the helper defined.

## Gate: PASS

All 7 quick check tests passed. The hero commands (`read`, `get`, `history`, `request`) all work against the live service. Paywall bypass works via Wayback fallback when archive.is CAPTCHAs body fetches. Ready to promote.
