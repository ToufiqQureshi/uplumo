# Google Search Console

## Legacy behavior and edge cases captured before the port

| Behavior                                                                                     | Rule                                                                                                                               |
| -------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Each user can link several Google grants.                                                    | List each grant's sites independently; a revoked grant asks for reconnection, other provider failures mark properties unavailable. |
| A project selects one verified property from an owned grant.                                 | Keep the `siteUrl` byte-for-byte, reject unverified or unavailable properties, preserve email only for the same grant identity.    |
| Legacy mappings may have no Google account ID.                                               | Token lookup falls back to that connector user's first grant.                                                                      |
| Performance accepts 7-day through 16-month date ranges, Web/Image/Video/News type, dimensions, filters and page offset. | Dates are UTC, recent ranges stop three days before today, report rows are bounded to 25,000 and filters must be inside `dimensionFilterGroups`. |
| Search Performance page queries current/previous daily rows, query-page pairs and countries. | Use impression-weighted average position, show striking-distance queries only when their best page ranks 5–20, and chart daily clicks or impressions. |
| Queries/pages table is paginated.                                                            | Fetch one extra row for `hasNextPage`; exports are capped at 25,000.                                                                |
| URL inspection handles a list of URLs.                                                       | Capture individual URL errors while grant/token errors fail the batch.                                                             |

## Status

The Go backend serves Search Console grant status, property listing and
selection, disconnect, URL inspection, and Search Performance
report/table/export routes. It reads the existing `gsc_connections` mapping
and uses the shared Google API client. Consent is in `internal/google`. The
React integration settings and the Search Performance page call these routes.

The Search Performance page's content refresh queue groups the report's
striking-distance queries by page and highlights pages with at least 50
impressions at average positions 5–20. It reuses the existing report response,
makes no extra provider request, and presents review candidates rather than
claiming a page is stale or changing site content.

Property selection and disconnect require an owner or admin role. Selecting a
property verifies that the chosen Google grant belongs to the user, that the
exact site URL appears in that grant, and that the permission is not
`siteUnverifiedUser`. Listing sites keeps per-grant reconnect/unavailable
states independent, and a user-info failure leaves the email empty.

## Search Performance API

All routes are POST, require a session and project access, and reject unknown
JSON fields. `dateRange` defaults to `last_28_days`; reports also accept a
custom `startDate` and `endDate` selected from the page's calendar. Preset
ranges span
`last_7_days` through `last_16_months`. `searchType` defaults to `web` and also
accepts `image`, `video` and `news`. The report runs four bounded Search Analytics reads, preserves the 3-day data lag
and 16-month history floor, and returns current totals, optional previous-period totals, daily rows,
countries and up to 1,000 striking-distance queries from Google's returned
query-page sample. Filters use one AND group; country is omitted from the country
breakdown. The UI groups those rows into a bounded review queue and can seed a
content draft with the page's strongest query; this adds no provider call. When a full previous period would cross the 16-month history floor, the comparison read is skipped and the UI says so.
The table accepts `query` or `page`, pages 1-based at sizes 25/50/100, and uses
one extra row to compute `hasNextPage`. Export is capped at 25,000 rows.

Routes and request/response contracts are in `backend/api/gsc.yaml`.

The MCP `get_search_console_performance` tool uses the same connection and
Google client. It accepts up to four dimensions, one AND-combined filter list,
bounded pagination, and optional position/impression thresholds. Thresholds
are applied to at most 1,000 provider rows because Search Console cannot filter
those metrics server-side. `searchAppearance` must be the only dimension.

## Internal link source

The content editor's free link suggestions read the project's Search Console
pages with impressions in the last 90 days and their top queries. The
`linksuggest` service applies its own 200-page bound, ranks candidates by
clicks and impressions, and falls back to the latest site audit when GSC has
no pages. The lookup is requested explicitly in the editor and does not
charge credits.

## URL inspection

`POST /api/v1/projects/{projectId}/gsc/url-inspection/inspect` inspects 1–10
URLs against the project's selected property. Each URL's result or a safe
generic per-URL error is returned in input order; revoked grants fail the
request so the client can prompt reconnection. The Search Performance page
shows index coverage, crawl and indexing state, last crawl time, and Google vs
declared canonical. This API reports the version in Google's index; it cannot
request indexing or inspect live-page indexability. Provider calls use the
Google API client's request timeout and bounded read-only retry policy.

The Search Performance page can load up to 10 priority URLs into this explicit
inspection action. Results include cautious guidance based on Google's fetch,
robots and indexing states. This is not a live test or an indexing request. The
separate Google Indexing API is not used for general URLs: Google documents it
for JobPosting and livestream BroadcastEvent pages and it requires service-account
site-owner setup.

## Deliberate differences

The existing provider client rejects Search Console filter expressions longer
than 1024 UTF-8 bytes before a provider call. The legacy page schema accepts up
to 4096 JavaScript characters, so Go returns a 400 for expressions over the
provider client's safe request bound. Product analytics events emitted by the
old server functions are not emitted by these Go endpoints. Changing the shared
provider bound needs a separate contract review.

## Rules

- Google first-party calls are free; they are still quota-limited and use
  bounded retries only for read-only operations.
- A provider 401 triggers one token refresh and one retry.
- Every repository query must include the project organization and user scope.
