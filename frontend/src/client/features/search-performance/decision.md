# Search Console performance: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Uses first-party Search Console data to prioritize content review and explain index/crawl status without claiming Google will rank or index a page.

## Decisions to preserve

- Search Console has provider quotas; preserve bounded read retries and avoid redundant refetches. _Why:_ Google limits read volume, so bounded queries and no refetch loops preserve quota for useful reports.
- Every request uses the project's selected property and OAuth grant; never accept a property from another project. _Why:_ A valid Google grant for one project does not authorize reading another project's property.
- Google returns sampled/limited rows; preserve response metadata and do not claim exhaustive totals. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Build the refresh queue from the already loaded Search Console rows and keep its position/impression thresholds and page cap explicit. _Why:_ Reusing report data avoids extra quota use and keeps the queue bounded.
- Label the rows as review candidates, not confirmed stale content, and keep content edits separate from this read-only queue. _Why:_ Search metrics identify opportunities but do not prove a page is outdated or authorize a site change.
- Keep the report to at most four Search Analytics reads and reuse its query-page rows for candidate grouping; cap surfaced queries at 1,000 and exports at 25,000 rows. _Why:_ More history and surfaced candidates must not create an unbounded quota or memory cost. Skip an unavailable full-period comparison at the 16-month history floor.
- URL inspection runs only after an explicit user action and inspects at most 10 URLs; it reports stored index data and cannot request indexing or inspect live indexability. _Why:_ Customers must not mistake a diagnostic for a Google crawl request.
- Seed Content Editor drafts only from an explicit row action, and pass only the selected query as the topic. _Why:_ This carries evidence into the next workflow without generating text, spending credits or mutating a published site.
- Do not add general-page submissions through Google's Indexing API. Google documents it for JobPosting and livestream BroadcastEvent pages and requires separate service-account ownership setup. _Why:_ Enabling the API in a Cloud project does not authorize arbitrary URLs.

## Verification pointers

- `searchPerformanceQueries.test.ts`
- `contentRefreshQueue.test.ts`
- `backend/internal/gsc/performance_test.go`
