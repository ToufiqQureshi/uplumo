# Google Search Console: decisions

Read this before changing selected properties, performance queries, sitemap
status or URL Inspection. `README.md` defines the supported routes and bounds;
`flow.md` traces requests through the scoped Google client.

| Decision | Why | Code / verification |
| --- | --- | --- |
| Resolve the selected property and OAuth grant from the authorized organization/project. | A valid grant for one site is not permission to read another customer's property; client-provided IDs are not authority. | `repository.go`, `connections.go`, connection/handler tests |
| Use bounded read-only retries and preserve safe provider failures. | Google has quotas; failure must not create a retry loop, leak tokens or look like valid empty evidence. | `client.go`, shared `google` transport tests |
| Label final Web Search windows and skip a full-period comparison that crosses the available history floor. | Incomplete or differently scoped periods can manufacture apparent growth/loss. | `performance.go`, `performance_test.go` |
| Keep growth-expanded pagination separate from existing report/table/export limits. | One explicit deeper investigation must not silently widen every user's report cost or response size. Exact bounds live in `README.md`. | `growth_rows.go`, `growth_rows_test.go` |
| Return exact-page query evidence for click diagnosis, retaining missing rows as unknown. | Provider privacy and top-row limits mean absence is not a measured zero or proof that a query disappeared. | `sitemaps.go` (`GetPageQueryRows`), GA4 `growth_clicks.go` |
| Read submitted sitemap metadata without fetching customer-supplied XML or expanding nested indexes. | The available API supports bounded sitemap status; a new crawler adds different security, quota and scope requirements. | `sitemaps.go`, real-client tests in `growth_rows_test.go` |
| Preserve errors/warnings, pending/download status and submitted counts by content type. Do not call their total unique indexed pages. | Types can overlap, and submission/download is not evidence that each page is indexed. Specific sitemap reasons remain in Google's report. | `sitemaps.go`, frontend `GrowthSitemaps.tsx` |
| Use explicit bounded URL Inspection for stored index state, with cautious canonical/fetch/indexing guidance. | This endpoint neither tests a live page nor requests a recrawl; a stored result can predate a fix. | `inspection.go`, frontend `indexingGuidance.ts` |
| Do not submit arbitrary URLs through Google's restricted Indexing API or claim access to every GSC UI report. | API enablement does not authorize general-page indexing, and native UI availability does not prove public API availability. | No general-page submission path; external report entry points are labeled |

## Future changes

Update this file in the same commit when changing a decision, supported scope or
failure/cost assumption. Preserve exact request bounds, organization checks and
missing-data semantics in provider/handler tests. Real-account reconciliation is
still required before claiming live Google reporting works for a customer.

## One active date-window implementation

- **Decision:** Remove unused `ResolveDateRange`, `PreviousPeriod` and month-subtraction helpers and their exclusive tests; retain `DateRange`.
- **Why:** Production routes use the validated implementation in `performance.go`. Keeping a second, unreachable algorithm and tests hid drift.
- **Preserve:** Live date presets, history floor, comparison semantics, authorization and Google quotas remain unchanged.
- **Trace:** `dates.go`, `performance.go`, `performance_test.go`; production deadcode scan identified the removed functions.

## Pages that lost clicks, with the reason

- **Decision:** When a comparison period exists, `GetPerformance` makes two more Search Analytics queries: the page dimension for each period, 1,000 rows each, with the report's own filters. `buildPageChanges` returns the 5 pages that lost the most clicks, biggest loss first and ties ordered by URL. A page missing from the current period counts as having lost all its clicks. Each page gets one reason, checked in this order:
  1. `rank_fell`: position worsened by 1 or more, or the page is no longer shown at all
  2. `fewer_searches`: shown at least 20% less
  3. `ctr_fell`: clicked at least 20% less
  4. `mixed`: none of the above
- **Why:** The Search Performance verdict has to name the pages to fix and why.
- **Preserve:**
  - Search Console is free (Google quota only). The report now makes 6 queries, all in parallel.
  - No comparison means no extra queries and an empty list.
  - The response field is always an array, never null.
- **Trace:** `page_changes.go`, `page_changes_test.go`. Each of these 6 mutations fails the tests: rank threshold, impressions threshold, CTR threshold, the "only losses" rule, sort order, a vanished page. `performance_test.go` asserts 6 requests and that both page periods carry the report filters.
