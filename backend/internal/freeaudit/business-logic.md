# Business rules

- **No paid provider, ever.** Lighthouse, DataForSEO, OpenRouter and PageSpeed stay out of the free audit. The only outside data is Chrome field data from the free CrUX API, and only when `CRUX_API_KEY` is set; without it the Core Web Vitals check says "not measured".
- **One fetch serves both engines.** A fresh check reads the site once, then `ScoreSEO` and `ScoreAEO` score that snapshot. Both reports are cached together, so a visitor who switches engines never causes a second read of the site.
- **Switching is never a check.** The tab switch asks for the cached report only (`cached=1`). It does not count toward fresh checks. The only time a switch runs a check is when the 24 h cache has expired, and then it is an explicit visit to that engine's report.
- **No automatic retries of anything that costs money.** The free audit has no paid calls, so its retries only cover its own requests.
- **Cache per page for 24 hours**, shared by all visitors. Unreachable pages are cached 10 minutes.
- **Limits** protect our server and the checked site: 30 requests/hour and 10 fresh checks/day per IP, 600 fresh checks/hour overall, 4 at once. The Word download and the tab switch count toward the 30/hour lookup limit but never toward fresh checks. If Redis is down, limits are skipped (still 4 at once).
- **The Word download only serves cached reports.** A visitor cannot use it to start a check, so it cannot be used to load the checked site.
- **SSRF guard**: private and internal addresses are refused, every redirect is revalidated.
- **Fetch identity.** Every request uses the audit's own user agent (`Uplumo-Audit/1.0`). We do not send bot user agents that pretend to be Googlebot or ChatGPT; robots.txt rules for those bots are read and reported, not impersonated.

## Scoring (both engines)

- A warning earns 35% of a check's weight; a pass 100%; a fail 0%; info is not scored. Check weights come from severity: critical 8, high 5, medium 3, low 1.
- Each engine averages its categories by their weights. The category weights are the engine's own (below).
- Caps keep serious problems visible. A cap only lowers the score; the report says when one applied. "Excellent" (90+) is impossible with a critical failure.
- Why the caps: the owner's WordPress + Yoast site scored 96 ("Excellent") on the first version, because the checks only tested page setup, which Yoast passes by default. The engines are now stricter and separate, so a page that does well on one cannot hide what is wrong with the other.

## SEO engine: categories and caps

| Category | Weight | Checks (severity) |
| --- | --- | --- |
| Crawl and index | 30 | HTTPS (critical), noindex (critical), Googlebot allowed by robots.txt (critical), HTTP 200 (high), redirects before the page (medium), canonical (medium), duplicate title/description/canonical tags (high), hreflang self and x-default (medium), XML sitemap (medium), robots.txt (low), HSTS (low), mixed content (medium) |
| On-page SEO | 30 | Title 30-60 characters and shares a word with the H1 (high), meta description 70-160 (medium), one meaningful H1 (high), heading order (low), H2 sections on pages of 300+ words (medium), clean URL (low), image alt text (medium: fail above 20% missing), internal links (medium: fail under 3), generic link text above 30% (medium), Open Graph tags (low), lang attribute (low) |
| Performance and mobile | 20 | Core Web Vitals from CrUX at the 75th percentile (high), server response 400 ms / 1 s (medium), viewport (high), image width and height (medium: fail above 10 images), HTML size over 500 KB (low), blocking scripts in the head (medium: warn above 3, fail above 8), lazy-loaded images after the first 3 (low, warn above 10) |
| Content and structured data | 20 | Word count (high: fail under 300, warn under 600), sentence length (medium), JSON-LD that parses (medium: warn at 1-2 broken blocks, fail at 3+), any schema.org type (medium), date within 18 months (medium) |

**Cap, SEO:** any critical failure caps at **49**: the page is not indexable, Googlebot is blocked, or it is not on HTTPS. Why: a page Google cannot crawl or index cannot rank, whatever its on-page work.

## AEO engine: categories and caps

| Category | Weight | Checks (severity) |
| --- | --- | --- |
| AI access | 25 | AI search crawlers may read the site (critical; blocking training-only bots is information, not an error), readable without JavaScript (critical), quoting allowed (high), no hidden instructions to AI (critical), llms.txt (low) |
| Answers AI can quote | 30 | Question headings: 3 or more (medium), answer paragraph of 25-80 words right under a question heading (medium: 2 pass, 1 warn), clear "X is ..." definition under a heading (medium), sections of 100-150 words that stand alone (medium: 30% of sections), lists and tables: 2 or more (low) |
| Evidence and trust | 25 | Figures: 3 or more (medium), links to government, academic or standards sources: 2 or more (medium), quotations (low), author and date (medium), date modified within 12 months (medium: warn to 24 months, fail after or undated), links to About and Contact pages (medium) |
| Entity clarity | 20 | Organization or LocalBusiness schema with name, logo, 2 or more sameAs links and contact or address (high: fail when none, warn when incomplete), brand name in the title matches og:site_name or the schema name (medium), FAQPage, HowTo, Article or Product schema (medium) |

**Cap, AEO:** any critical failure caps at **39**: an AI search crawler is blocked, the page is an empty shell without JavaScript, or it hides instructions aimed at AI. Why: an assistant that cannot fetch the page, or that is asked to read instructions meant only for it, cannot cite it.

## Sources the rules are based on

These are the sources named in the owner's brief for this release. They were
not re-read or re-verified in this change; check them again before the next
change to the thresholds.

- SEO: 2026 technical and on-page SEO audit guides (crawlability first; Core Web Vitals LCP under 2.5 s, INP under 200 ms, CLS under 0.1; on-page match to intent; structured data; content quality).
- AEO checklists: AirOps' 48-factor AEO checklist, RankPrompt, AI Rank Lab, Answerly.
- Citation studies named in the brief, figures not re-verified: Profound's citation study; Ahrefs' brand-mention study; GEO research on clear definitions, statistics, quotations and citations.
- llms.txt is treated as a weak signal: few assistants read it yet.

## Other rules

- **Prices on upgrade cards come from `plans/catalog.go`** in the visitor's currency (INR in India, USD elsewhere). Each report offers different tools: the SEO report leads with the full site audit and rank tracking; the AEO report leads with the AI visibility tracker. The content editor follows either when the page's writing is weak.
- **Honest scope.** Each report says what its engine measures and what it does not (rankings, links, AI mentions). Do not add promises to the page copy that the score does not back.
- **The full report is free**, no email gate.
- The report is `noindex` and `no-store`; `/free-seo-audit` and `/free-aeo-audit` are indexed.
