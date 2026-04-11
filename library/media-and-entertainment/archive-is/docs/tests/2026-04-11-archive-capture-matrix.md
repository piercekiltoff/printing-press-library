# Archive Capture Test Matrix — 2026-04-11

Generated from plan: `docs/plans/2026-04-11-002-test-archive-capture-matrix-plan.md`
Approved by Matt: 2026-04-11, ~09:27 local
Binary under test: `~/go/bin/archive-is-pp-cli` (built from 2026-04-11 polish + agent-hints session)

## The 20 URLs

| # | Category | URL | Expected content |
|---|----------|-----|------------------|
| **Category 1: Open web / no paywall** |
| 1 | Wikipedia | `https://en.wikipedia.org/wiki/Go_(programming_language)` | Google's Go language, 2009 release, goroutines, static typing |
| 2 | Daring Fireball | `https://daringfireball.net/` | John Gruber's blog, Apple commentary, linked list |
| 3 | Simon Willison | `https://simonwillison.net/` | Simon's blog, LLM tooling, datasette, recent posts |
| **Category 2: Mainstream news, no paywall** |
| 4 | BBC News | `https://www.bbc.com/news` | BBC front page, world headlines |
| 5 | NPR | `https://www.npr.org/` | NPR front page, US+world news mix |
| 6 | Reuters | `https://www.reuters.com/` | Reuters front page, wire service headlines |
| **Category 3: Tech news / blogs** |
| 7 | TechCrunch | `https://techcrunch.com/` | TechCrunch front page, startup + venture news |
| 8 | Ars Technica | `https://arstechnica.com/` | Ars front page, deep tech coverage |
| 9 | GitHub blog | `https://github.blog/` | GitHub product + engineering blog |
| **Category 4: Soft paywall / newsletter / metered** |
| 10 | Guardian US | `https://www.theguardian.com/us` | Guardian US section |
| 11 | Platformer | `https://www.platformer.news/` | Casey Newton's Substack |
| 12 | Stratechery | `https://stratechery.com/` | Ben Thompson's site |
| **Category 5: JS-overlay paywall — archive.is sweet spot** |
| 13 | The Atlantic | `https://www.theatlantic.com/` | The Atlantic front page |
| 14 | The New Yorker | `https://www.newyorker.com/` | New Yorker front page |
| 15 | The Economist | `https://www.economist.com/` | Economist front page |
| **Category 6: International / non-English** |
| 16 | Le Monde | `https://www.lemonde.fr/` | Le Monde front page, French |
| 17 | Der Spiegel | `https://www.spiegel.de/` | Der Spiegel front page, German |
| 18 | NHK News | `https://www3.nhk.or.jp/news/` | NHK News front page, Japanese |
| **Category 7: DataDome controls** |
| 19 | NYT article | `https://www.nytimes.com/2026/04/11/world/middleeast/iran-peace-talks-demands.html` | Iran peace talks demands article (Matt verified working earlier today) |
| 20 | WSJ article | `https://www.wsj.com/world/middle-east/free-seas-iran-strait-hormuz-toll-3404b2e1?mod=WSJ_home_mediumtopper_pos_1` | Iran Hormuz toll article (failed submit earlier, Wayback had real content) |

## Execution gates

- **Phase A (timegate + Wayback):** Run now. Not rate-limited.
- **Phase B (fresh submits):** Deferred. Rate-limit cooldown active until **10:09 local** (42 minutes from matrix creation).

## Result file location

`docs/tests/2026-04-11-archive-capture-results.md` (populated by Unit 2)
