# Business rules

- **No paid provider, ever.** Lighthouse, DataForSEO, OpenRouter and PageSpeed stay out of the free audit. The only outside data is Chrome field data from the free CrUX API, and only when `CRUX_API_KEY` is set; without it the Core Web Vitals check says "not measured".
- **No automatic retries of anything that costs money.** The free audit has no paid calls, so its retries only cover its own requests.
- **Cache per page for 24 hours**, shared by all visitors. Unreachable pages are cached 10 minutes.
- **Limits** protect our server and the checked site: 30 requests/hour and 10 fresh checks/day per IP, 600 fresh checks/hour overall, 4 at once. The Word download counts toward the 30/hour lookup limit but never toward fresh checks. If Redis is down, limits are skipped (still 4 at once).
- **The Word download only serves cached reports.** A visitor cannot use it to start a check, so it cannot be used to load the checked site.
- **SSRF guard**: private and internal addresses are refused, every redirect is revalidated.
- **Scoring and caps.** A warning earns 35% of a check's weight; a pass 100%; a fail 0%. Categories are weighted SEO 30%, AI 30%, Technical 20%, Content 20%. Caps keep serious problems visible: any critical failure caps at 59, any high failure at 79, five or more warnings at 89. "Excellent" (90+) is therefore impossible with a serious failure. Why: the owner's WordPress+Yoast site scored 96 ("Excellent") because the old checks only tested page setup, which Yoast passes by default.
- **Honest scope.** The report says what it measures (readiness of the page) and what it does not (rankings, links, AI mentions). Do not add promises to the page copy that the score does not back.
- **Prices on upgrade cards come from `plans/catalog.go`** in the visitor's currency (INR in India, USD elsewhere).
- **The full report is free**, no email gate.
- The report is `noindex` and `no-store`; `/free-seo-audit` is indexed.
