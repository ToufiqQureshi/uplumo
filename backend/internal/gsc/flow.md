# Google Search Console: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/gsc/...; `/gsc/search-performance/...`; URL inspection routes
- **Source:** `connection_handler.go`, `performance.go`, `inspection.go`, `client.go`

## Request or job flow

Project request resolves saved property and grant; client validates filter grammar and API budget; Google API returns performance/inspection results; handlers normalize rows while retaining per-URL outcomes.

## Dependencies and cost boundary

Google Search Console API; encrypted Google grants; selected property in Postgres.

## Failure, security and data invariants

- Property and grant must belong to the authorized organization/project.
- Google quotas require bounded read-only retries; do not retry writes blindly.
- Treat provider row limits and inspection errors accurately; null/empty is not proof of no search demand.

Sitemap drill-down -> selected scoped property/client -> one bounded Google
`sitemaps.list` read -> first 200 entries + truncation and indexing caveats.
Customer-controlled sitemap URLs are displayed safely but never fetched by the
server. Click investigation reads two exact-page query windows on explicit action;
expanded growth pagination stops at five 1,000-row pages per source.

Search Performance report -> 8 free Search Console reads with a comparison
(date, page and query for both periods, query+page, country), 4 without ->
totals, daily trend, striking distance, countries, `pageChanges`,
`queryLosses` and `queryGains` with reason and severity. MCP/SAM query rows
add one no-dimension read for the true property total.
