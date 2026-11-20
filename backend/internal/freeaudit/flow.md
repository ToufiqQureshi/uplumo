# Request path

The free audit has two pages and one report route. Both engines read the same
fetch, so the first report of a page costs one check, and the other engine is
read from the cache.

## Landing and audit pages

- Landing form (`/`): two submit buttons with `formaction`, `/free-seo-audit` and `/free-aeo-audit`. Without JavaScript the browser opens the audit page with `?url=`.
- Audit page (`/free-seo-audit`, `/free-aeo-audit`): the page's own path sets the default tab. `free-audit.js` runs the check for `?url=` (and `&type=` when present).

## Run the audit (`GET /free-seo-audit/report?url=&type=`)

1. Handler checks the input length (2048 bytes max) and `type` (`seo` default, `aeo`; anything else is a 400).
2. `Service.Run(kind)`:
   - lookup limit per IP (30/hour, cached requests too),
   - `PageChecker.NormalizeURL`: SSRF validation, redirect probe (plain HTTP, no provider),
   - cache read (`free-audit:v3:<hash>`); a hit returns the report of `kind`,
   - singleflight on the URL; re-read cache; fresh-check limits (10/day per IP, 600/hour global),
   - `check`: one slot of 4, 30 s timeout. One `PageChecker.Check` returns a `PageSnapshot`; `ScoreSEO` and `ScoreAEO` score it; both reports are cached 24 h (10 min if unreachable).
3. `PageChecker.Check`, in parallel: fetch the page (`fetchAnalyzed`, up to 3 redirects, each SSRF-checked, the hop count is kept), robots.txt, llms.txt.
4. After the page is reachable: sitemap (1 request), then one CrUX lookup (phone) if a source is configured, 8 s timeout.
5. `ScoreSEO` / `ScoreAEO` build their checks from the same snapshot, score them (`scoreChecks`, caps in `applyCaps`).
6. `Pages.ServeAuditReport` renders the fragment (`audit-report` template) for the report's kind.

## Switch the report (`…&type=other&cached=1`)

1. The page keeps the reports it has shown for the current address in memory, so switching back is free.
2. Otherwise the request reads the cache only (`Service.Cached`). It never runs a check and never counts toward fresh checks; it counts toward the 30/hour lookup limit.
3. A 404 (the 24 h cache has expired) makes the page run the check again for that engine, as a visitor who asked for it.
4. The browser URL is updated with `history.replaceState` (`?url=…&type=seo|aeo`).

## Word download (`GET /free-seo-audit/report.docx?url=&type=`)

1. Lookup limit per IP (same as above).
2. `NormalizeURL` (SSRF check and one redirect probe; no check).
3. Cache read only. Not cached: 404 fragment. Cached: `BuildDocx` for the kind and a zip response.

## Print / PDF

Client only: `Download PDF` calls `window.print()`. The `beforeprint` listener opens every check; `@media print` in `base.html` hides the chrome.

## Paid calls per click

- Run audit (either engine, fresh): 0 paid calls. A fresh check makes at most 3 redirects + page + robots + llms + sitemap = 7 requests to the checked site, and at most 1 CrUX request (free Google API, only if `CRUX_API_KEY` is set).
- Switch SEO / AEO after a report: 0 checks, 0 provider calls, 0 requests to the checked site (one cached request to us, or none when the report is already on the page).
- Download Word: 0 checks, 0 provider calls.
- Download PDF: 0 requests.

Tested in `service_test.go` (`TestOneFreshCheckServesBothEngines`, `TestCachedNeverRunsACheck`) and `handler_test.go` (`TestToggleReadsTheCacheAndNeverChecks`, `TestDocxRouteNamesTheReportKind`).
