# Uplumo Analytics

A first-party tracker and project analytics dashboard with short tab sessions.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `AnalyticsPage.tsx`
- `TrackingCodeCard.tsx`
- `analyticsApi.ts`
- `backend/internal/httpapi/tracker.js`
- `backend/internal/analytics/service.go`

## Daily recognition and click destinations

Optional browser recognition (`data-recognition="true"`) uses a 180-day site-keyed localStorage ID after required consent, with only its site-scoped hash/first-seen timestamp stored server-side. Default snippets retain anonymous visitor-days. Daily new/returning recognized browsers and unrecognized pageview coverage are separate measures; old visits cannot be classified. See backend analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt` for precise counting, consent, retention and verification.

`GET /api/v1/projects/{projectId}/analytics/summary?from&to` adds `audience` rows: `date`, `newBrowsers`, `returningBrowsers`, `unknownPageviews`. `GET /api/v1/projects/{projectId}/analytics/clicks?from&to&page&limit` returns `rows` (`page`, `name`, `host`, `path`, `type`, `clicks`, `visitorDays`, `arrivals`) and `hasMore`. Both require session/project access. Dates are inclusive UTC days, at most 366. Click pages are 1–100 rows, page 1–1000; UI uses 20 rows and project/date-keyed one-minute TanStack caching. No new server cache or paid call.

Collector additions are optional `b` (32 lowercase-hex browser ID), `c` (click boolean), `x` (absolute HTTP(S) destination without credentials). Clicks require event/session/pageview IDs. Configured clicks and automatic outbound links share the same deduplicated event record. Destination query strings/fragments are stripped; JavaScript buttons can specify `data-uplumo-destination`. Internal arrivals are matching later same-tab pageviews within 30 minutes, not proof of causation. External subsequent activity is unavailable.

## Dashboard hierarchy

`AnalyticsOverview.tsx` leads with the selected UTC dates, observed pageviews and
custom events, leading source/page, and identified AI referrals. It flags
pageviews without sessions and event names that differ only by letter case;
Review tracking setup opens the existing snippet panel. Events remain distinct
and are not labeled verified enquiries or sales.

The main report keeps stats, the trend, sources, top pages and visitor actions.
Audience recognition, campaign/session acquisition, geography, devices and page
journeys remain available in an expandable section. A summary refresh failure
shows an error alongside the last loaded data. AI attribution describes available
signals without promising conversion uplift or recovery of all direct traffic.

## Tracking and intelligence views

`AnalyticsWorkspace.tsx` preserves Website tracking as the default and exposes
SEO intelligence as a URL-backed `view=intelligence` tab. The shared
`search-performance/SearchGrowthPanel` uses connected GSC/GA4 data with its own
complete-period analysis scope; first-party tracker totals are never mixed into
GA4 conversion rates. The Website tracking view retains its 7/30/90-day selector.
`TrackingStatusCard.tsx` shows the latest recorded pageview **day in the selected
report**, an explicit stored-data refresh and an installation/test checklist. It
does not pretend to remotely verify all browser tags, blockers or consent.
