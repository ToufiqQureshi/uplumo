# Flow

1. Visitor submits the form (landing or `/free-seo-audit`) -> `/free-seo-audit?url=X`.
2. `free-audit.js` calls `GET /free-seo-audit/report?url=X`.
3. Handler -> `Service.Run`:
   - limit: 30 requests/hour per IP (cached ones too);
   - `PageChecker.NormalizeURL`: SSRF guard + DNS + HEAD redirect probe;
   - Redis cache hit -> return (no fetch);
   - singleflight per page; limits: 10 fresh checks/day per IP, 600/hour for everyone;
   - wait up to 15 s for one of 4 slots, run `PageChecker.Check` (30 s cap), cache.
4. `site.ServeAuditReport` renders the HTML fragment (prices in the visitor's currency).

## Paid calls per click

**0.** No DataForSEO, PageSpeed, OpenRouter or any provider. A fresh check makes
at most 7 requests to the visitor's own site (page + up to 3 redirects,
robots.txt, llms.txt, sitemap). Tested in `audit/pagecheck_test.go` (4
requests for a normal page) and `freeaudit/service_test.go` (one check for
20 concurrent visitors; cache for everyone).
