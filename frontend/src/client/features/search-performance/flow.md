# Search Console performance: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. User selects the GSC property, calendar date range, search type, device, country and query/page filters. Date and filter state stays in the URL so the view is shareable.
2. `searchPerformanceQueries.ts` keys requests by the full filter set and `searchPerformanceApi.ts` calls Go GSC routes.
3. Go verifies project access and selected property, validates dates and filters, and runs up to four bounded Search Analytics reads (three when no full prior period is available inside the 16-month history). Results include current totals, an optional period comparison, daily trend, country rows and up to 1,000 returned query-page rows.
4. The page groups those returned rows into striking-distance and content-refresh candidates. Candidate pages can be opened for review, or the strongest query can seed a new Content Editor draft. This does not call AI, publish, or change the site.
5. URL inspection can load up to 10 priority URLs with an explicit click. It displays Google's stored index/crawl result and a cautious next step; it does not request a crawl, index a live page, or guarantee ranking.

## Implementation map

- `SearchPerformancePage.tsx`
- `searchPerformanceQueries.ts`
- `searchPerformanceApi.ts`
- `backend/internal/gsc/performance.go`
- `backend/internal/gsc/inspection.go`
- Content Editor handoff: `ContentRefreshQueuePanel.tsx`, `content-editor.tsx`, `ContentEditorPage.tsx`

## API and service dependencies

Go package: `backend/internal/gsc`; tokens and selected-property ownership are managed by `backend/internal/google` and `gsc`.

## Content refresh and draft handoff

The queue reuses the report's query-page rows; it makes no extra Search Console request. `contentRefreshQueue.ts` keeps rows at average position 5-20 with at least 50 impressions, groups them by page, sums clicks/impressions and computes impression-weighted position. It shows the three strongest query themes per page and the first 25 candidate pages. Search Console can omit rows, so the queue is incomplete by design. A row can open the page or start a new Content Editor draft seeded with its strongest query. The seed is a plain draft topic; no AI call or publish happens.

## Cost, security and failure boundaries

- Search Console has per-site, per-user and per-project quotas; keep the four report reads and exports bounded, and avoid refetch loops.
- Every request uses the project's selected property and OAuth grant; never accept a property from another project.
- Google returns sampled/limited rows; report totals and candidate rows are different views and candidate lists are not exhaustive.
- URL inspection is an explicitly clicked read-only call, capped at 10 URLs per request.

The independent Search Growth panel invokes `ga4/growth/report` only on click.
Saving an action records its server report ID, page and note; the backend owns
the metric baseline. Mark applied starts a fixed post-change observation window.
Daily click averages permit 14/28-day comparison with the labeled baseline.
Measurement/device/citation drill-downs run only when explicitly requested.
Stored AI evidence preserves the run date and collection coverage separately
from GSC/GA4 dates; it never buys new answers.

AI handoff writes a 30-minute, tab-local, project-scoped draft and opens SAM.
Send explicitly enables server-enforced search_growth mode: read-only Google
and guarded site tools; ordinary SAM model metering still applies. Editor
handoff carries the source URL, query cluster, date range and finding evidence
through validated tab storage, preserving existing drafts.

Load goals -> choose a registered successful outcome -> explicit analysis -> top
three candidates. Settings and reports preserve separate goal labels, and report
changes reset page-level saves/diagnosis state. Why clicks are low -> two exact-page
query reads; Check stored index -> one URL Inspection read. Attribution, funnels,
sitemaps and device checks remain separate explicit clicks. No drill-down calls a
paid provider. Weekly configuration reads only on panel open; scheduling requires
a separate enable click and shows its own stored goal/date scope.

Investigation navigation opens one topic at a time and does not fetch by itself.
Button actions retain their existing endpoints and call counts. Journey step
inputs submit the same event-name array; changing viewport only switches result
presentation between mobile metric cards and desktop tables, without refetching.

## Intelligence workspace

Analytics SEO intelligence and Search Performance share `SearchGrowthPanel`.
Explicit analysis uses the existing <=5 standard report reads (+ optional goal
metadata), <=25 expanded reads and server cache. A disabled TanStack query caches
by project and full analysis input; remount/focus/reconnect do not initiate reads.
Metric cards, ten-page finding lists and selecting a page add zero calls.
Analyst questions append to the existing project-scoped editable SAM draft;
Send retains normal server metering. Detailed GSC report reads retain their
independent filter/date scope. All findings remain observational evidence.
