# Business rules

- **No paid provider, ever.** The audit is a lead magnet; it must cost only a
  few HTTP requests. Anything paid (Lighthouse, AI answers, keyword data) stays
  behind sign-up.
- **Cache per page for 24 hours**, shared by all visitors: a page shared on
  social media is checked once. Unreachable pages are cached for 10 minutes so
  a fixed site can be rechecked soon.
- **Limits** protect our server and the checked site from being used as a
  crawler: 30 requests/hour and 10 fresh checks/day per IP, 600 fresh checks
  an hour overall, 4 at once. If Redis is down limits are skipped (still 4 at
  once; nothing costs money).
- **SSRF guard**: private and internal addresses are refused, every redirect is
  revalidated.
- **Prices on upgrade cards come from `plans/catalog.go`** in the visitor's
  currency (INR in India, USD elsewhere), the cheapest plan with the feature.
- **The full report is free**, no email gate (owner: "sab details dikhey").
- The report is `noindex` and `no-store`; the page `/free-seo-audit` is indexed.
