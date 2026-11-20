# Request path

## Run the audit (`GET /free-seo-audit/report?url=`)

1. Handler checks the input length (2048 bytes max).
2. `Service.Run`:
   - lookup limit per IP (30/hour, cached requests too),
   - `PageChecker.NormalizeURL`: SSRF validation, redirect probe (plain HTTP, no provider),
   - cache read (`free-audit:v2:<hash>`); hit returns the report,
   - singleflight on the URL; re-read cache; fresh-check limits (10/day per IP, 600/hour global),
   - `check`: one slot of 4, 30 s timeout, result cached 24 h (10 min if unreachable).
3. `PageChecker.Check`, in parallel: fetch the page (`fetchAnalyzed`, up to 3 redirects, each SSRF-checked), robots.txt, llms.txt.
4. After the page is reachable: sitemap (1 request), then one CrUX lookup (phone) if a source is configured, 8 s timeout.
5. `pageChecks` builds the checks from the page analysis, scores them (`scoreChecks`, caps in `applyCaps`).
6. `Pages.ServeAuditReport` renders the fragment (`audit-report` template).

## Word download (`GET /free-seo-audit/report.docx?url=`)

1. Lookup limit per IP (same as above).
2. `NormalizeURL` (SSRF check and one redirect probe; no check).
3. Cache read only. Not cached: 404 fragment. Cached: `BuildDocx` and a zip response.

## Print / PDF

Client only: `Download PDF` calls `window.print()`. The `beforeprint` listener opens every check; `@media print` in `base.html` hides the chrome.

## Paid calls per click

- Run audit: 0 paid calls. Fresh check makes at most 3 redirects + page + robots + llms + sitemap = 7 requests to the checked site, and at most 1 CrUX request (free Google API, only if `CRUX_API_KEY` is set).
- Download Word: 0 checks, 0 provider calls.
- Download PDF: 0 requests.
