# Uplumo Analytics: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. A project member opens Analytics and requests the site snippet through `analyticsApi.ts`.
2. The Go analytics service creates or returns one site key per project; the UI shows the tracker snippet.
3. The customer site loads `/t.js`; the tracker keeps a random session ID in first-party `sessionStorage`, measures visible page time, and posts pageviews/custom events to `/collect`.
4. The Analytics page loads summary/country data and displays acquisition, sessions, engagement, goal completions, pages, devices, events and geography.

## Implementation map

- `AnalyticsPage.tsx`
- `TrackingCodeCard.tsx`
- `analyticsApi.ts`
- `backend/internal/httpapi/tracker.js`
- `backend/internal/analytics/service.go`

## API and service dependencies

Go package: `backend/internal/analytics`; ingestion route is mounted in `backend/internal/httpapi`.

## Cost, security and failure boundaries

- Do not add cookies or persist raw IP addresses or session tokens; the browser token expires after 30 minutes idle or 24 hours, and the server stores only its hash.
- Collection is public by design, so preserve bot/domain checks and per-IP/per-site rate limits.
- Custom events must use the same validation and rate limits as pageviews.
- This feature does not call paid SEO providers.

## Enquiry evidence

Navigation or the editor's published-page link opens EnquiriesPage. A saved
goal determines the free report. The user can choose a path, record a planned
edit and immutable pre-change baseline, then request its first complete-week
comparison. Each query key includes project and comparison ID. Goal edits
invalidate reports; saved changes retain their original event name.
