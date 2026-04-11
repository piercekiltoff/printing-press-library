# Archive Capture Test Results — 2026-04-11 (Phase A)

Generated: 2026-04-11, ~09:50 local
Binary: `~/go/bin/archive-is-pp-cli` (build from 2026-04-11 polish + agent-hints)
Phase B (fresh submits): **deferred** — IP rate-limit cooldown until 10:09 local

## Summary

| Test | OK | FAIL |
|------|----|------|
| `read` (timegate + Wayback fallback) | 19/20 | 1/20 |
| `tldr` (read + body fetch + LLM summary) | 5/20 | 15/20 |

## Per-URL results

| # | Cat | Site | read | tldr | Memento URL | Headline (first 60 chars) |
|---|-----|------|------|------|-------------|---------------------------|
| 1 | open-web | Wikipedia | ✓ archive-i | ✗ | https://archive.md/20250707044641/https://en.wikipedia.org/w |  |
| 2 | open-web | Daring Fireball | ✓ archive-i | ✓ wayback | https://archive.md/20260304211858/https://daringfireball.net | Daring Fireball roundup skewers OpenAI's "superapp" pivot, s |
| 3 | open-web | Simon Willison | ✓ archive-i | ✗ | https://archive.md/20260402014321/https://simonwillison.net/ |  |
| 4 | news-free | BBC News | ✓ archive-i | ✗ | https://archive.md/20260411092432/https://www.bbc.com/news |  |
| 5 | news-free | NPR | ✓ archive-i | ✗ | https://archive.md/20260411014921/https://www.npr.org/ |  |
| 6 | news-free | Reuters | ✓ archive-i | ✓ wayback | https://archive.md/20260411025517/https://www.reuters.com/ | India and EU finalize landmark trade deal slashing tariffs a |
| 7 | tech-blog | TechCrunch | ✓ archive-i | ✗ | https://archive.md/20260411014440/https://techcrunch.com/ |  |
| 8 | tech-blog | Ars Technica | ✓ archive-i | ✗ | https://archive.md/20260408105906/https://arstechnica.com/ |  |
| 9 | tech-blog | GitHub blog | ✓ archive-i | ✗ | https://archive.md/20260113053207/https://github.blog/ |  |
| 10 | soft-paywall | Guardian US | ✓ archive-i | ✗ | https://archive.md/20260411000605/https://www.theguardian.co |  |
| 11 | soft-paywall | Platformer | ✓ archive-i | ✓ wayback | https://archive.md/20241001080010/https://www.platformer.new | Platformer's recent coverage spotlights AI race dynamics, co |
| 12 | soft-paywall | Stratechery | ✓ archive-i | ✗ | https://archive.md/20260324064437/https://stratechery.com/ |  |
| 13 | js-paywall | The Atlantic | ✓ archive-i | ✗ | https://archive.md/20260410030635/https://www.theatlantic.co |  |
| 14 | js-paywall | The New Yorker | ✓ archive-i | ✗ | https://archive.md/20260411024957/https://www.newyorker.com/ |  |
| 15 | js-paywall | The Economist | ✓ archive-i | ✗ | https://archive.md/20260411055340/https://www.economist.com/ |  |
| 16 | international | Le Monde | ✓ archive-i | ✗ | https://archive.md/20260329174413/https://www.lemonde.fr/ |  |
| 17 | international | Der Spiegel | ✓ archive-i | ✓ wayback | https://archive.md/20260411075101/https://www.spiegel.de/ | Artemis-2 crew returns safely to Earth after historic lunar  |
| 18 | international | NHK News | ✓ archive-i | ✗ | https://archive.md/20251013200905/https://www3.nhk.or.jp/new |  |
| 19 | datadome | NYT article | ✓ archive-i | ✗ | https://archive.md/20260411080659/https://www.nytimes.com/20 |  |
| 20 | datadome | WSJ article | ✗ | ✓ wayback |  | Iran's "Tehran toll booth" in the Strait of Hormuz ends the  |

## tldr successes (real content extracted)

### [2] Daring Fireball
- **URL:** `https://daringfireball.net/`
- **Memento:** http://web.archive.org/web/20260410014539/https://daringfireball.net/
- **Backend:** wayback
- **Headline:** Daring Fireball roundup skewers OpenAI's "superapp" pivot, slams Adobe's hosts-file hack, and flags a 49-day macOS Tahoe uptime crash bug.
  - A uint32 millisecond overflow in XNU's TCP timestamp code freezes Tahoe Macs after 49 days, 17 hours of uptime; pre-Tahoe versions seemingly unaffected.
  - Gruber pans OpenAI's $122B raise, TBPN acquisition, and merged-app strategy as panic moves, citing projected $111B+ losses and no defensible moat.
  - Other items: Adobe edits /etc/hosts to detect Creative Cloud, Anthropic withholds powerful Claude Mythos model, and Perplexity's "Incognito Mode" called a sham in lawsuit.

### [6] Reuters
- **URL:** `https://www.reuters.com/`
- **Memento:** http://web.archive.org/web/20260127075746/https://www.reuters.com/
- **Backend:** wayback
- **Headline:** India and EU finalize landmark trade deal slashing tariffs after nearly two decades of negotiations
  - PM Modi announced the deal opening India's vast market to free trade with its biggest trading partner, the 27-nation EU bloc.
  - Indian carmaker shares fell up to 5% on reports of sharp tariff cuts for European car imports, the sector's most aggressive opening yet.
  - European carmakers gain relief from US tariffs and China price wars, but face tough competition from homegrown firms and Japanese kei cars.

### [11] Platformer
- **URL:** `https://www.platformer.news/`
- **Memento:** http://web.archive.org/web/20260410014611/https://www.platformer.news/
- **Backend:** wayback
- **Headline:** Platformer's recent coverage spotlights AI race dynamics, content moderation shifts, and Big Tech governance changes
  - Meta released a new AI model nine months after an overhaul, while Anthropic's most dangerous model yet rattles cybersecurity experts.
  - Meta reportedly discussed ending Oversight Board funding, retreated from encryption plans, and Zuckerberg resumed content moderation decisions via Musk texts.
  - OpenAI faces pre-IPO scrutiny over strategy and executive reshuffling; Spotify battles AI impersonation; social media child safety verdicts alarm Section 230 advocates.

### [17] Der Spiegel
- **URL:** `https://www.spiegel.de/`
- **Memento:** http://web.archive.org/web/20260411114752/https://www.spiegel.de/
- **Backend:** wayback
- **Headline:** Artemis-2 crew returns safely to Earth after historic lunar flyby mission
  - NASA's Artemis-2 capsule splashed down in the Pacific at 2:07 AM German time, ending a ten-day mission with all four astronauts healthy.
  - The crew, launched April 1 from Cape Canaveral, traveled 407,000 km and viewed the Moon's far side, the first crewed lunar mission in over 50 years.
  - German astronaut Alexander Gerst predicts routine Moon stations soon, saying "this time we're coming to stay"; NASA called it a "perfect mission."

### [20] WSJ article
- **URL:** `https://www.wsj.com/world/middle-east/free-seas-iran-strait-hormuz-toll-3404b2e1?mod=WSJ_home_mediumtopper_pos_1`
- **Memento:** http://web.archive.org/web/20260411124216/https://www.wsj.com/world/middle-east/free-seas-iran-strait-hormuz-toll-3404b2e1?mod=WSJ_home_mediumtopper_pos_1
- **Backend:** wayback
- **Headline:** Iran's "Tehran toll booth" in the Strait of Hormuz ends the era of free seas as US Navy watches
  - A six-week Iran War has shattered the century-old system of free maritime trade, with Iran now controlling passage through the Strait of Hormuz.
  - Some 20,000 sailors and 700+ vessels carrying tens of billions in cargo are stuck as Iran dictates which ships leave and at what price.
  - Iran's navy warned vessels transiting without permission "will be destroyed," signaling America no longer rules the waves in the world's key oil corridor.

## tldr failures

- **[1] Wikipedia** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[3] Simon Willison** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[4] BBC News** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[5] NPR** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[7] TechCrunch** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[8] Ars Technica** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[9] GitHub blog** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[10] Guardian US** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[12] Stratechery** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[13] The Atlantic** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[14] The New Yorker** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[15] The Economist** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[16] Le Monde** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[18] NHK News** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available
- **[19] NYT article** — archive.is CAPTCHA and Wayback lookup failed: no wayback snapshot available