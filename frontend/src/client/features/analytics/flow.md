# Uplumo Analytics: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. A project member opens Analytics and requests the site snippet through `analyticsApi.ts`.
2. The Go analytics service creates or returns one site key per project; the UI shows the tracker snippet.
3. The customer site loads `/t.js`; the tracker keeps a random session ID in first-party `sessionStorage`, measures visible page time, and posts pageviews/custom events to `/collect`.
4. The Analytics page loads summary/country data and displays acquisition, sessions, engagement, pages, devices, custom events and geography.

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

## Daily recognition and click destinations

Optional browser recognition (`data-recognition="true"`) uses a 180-day site-keyed localStorage ID after required consent, with only its site-scoped hash/first-seen timestamp stored server-side. Default snippets retain anonymous visitor-days. Daily new/returning recognized browsers and unrecognized pageview coverage are separate measures; old visits cannot be classified. See backend analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt` for precise counting, consent, retention and verification.

`GET /api/v1/projects/{projectId}/analytics/summary?from&to` adds `audience` rows: `date`, `newBrowsers`, `returningBrowsers`, `unknownPageviews`. `GET /api/v1/projects/{projectId}/analytics/clicks?from&to&page&limit` returns `rows` (`page`, `name`, `host`, `path`, `type`, `clicks`, `visitorDays`, `arrivals`) and `hasMore`. Both require session/project access. Dates are inclusive UTC days, at most 366. Click pages are 1–100 rows, page 1–1000; UI uses 20 rows and project/date-keyed one-minute TanStack caching. No new server cache or paid call.

Collector additions are optional `b` (32 lowercase-hex browser ID), `c` (click boolean), `x` (absolute HTTP(S) destination without credentials). Clicks require event/session/pageview IDs. Configured clicks and automatic outbound links share the same deduplicated event record. Destination query strings/fragments are stripped; JavaScript buttons can specify `data-uplumo-destination`. Internal arrivals are matching later same-tab pageviews within 30 minutes, not proof of causation. External subsequent activity is unavailable.

## Overview and secondary reports

The overview derives leaders, event totals, AI referrals and measurement warnings
from the already-loaded summary. It adds zero API/provider requests. Its action
opens Tracking code or jumps to the existing visitor-action section. The native
report disclosure changes visibility only; children retain their existing query
keys, scopes and call boundaries. Opening it does not buy provider work.

Analytics route -> URL-backed Website tracking / SEO intelligence switch.
Website tracking keeps existing scoped queries; Check received data explicitly
refreshes the selected summary. SEO intelligence uses the shared combined Google
analysis, independently dated and cached; it never combines first-party events
with GA4 outcomes or runs AI on opening. Tracker initialization is once per site
per document; pushState, replaceState and popstate share URL-change deduplication.
