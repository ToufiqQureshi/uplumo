# Uplumo Analytics

A first-party tracker and project analytics dashboard with short tab sessions.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `AnalyticsPage.tsx`
- `TrackingCodeCard.tsx`
- `analyticsApi.ts`
- `backend/internal/httpapi/tracker.js`
- `backend/internal/analytics/service.go`

## Missed Enquiries Finder

`EnquiriesPage.tsx` and `enquiryApi.ts` expose the existing Analytics events as a
page-level investigation and measurement workflow at `/p/$projectId/enquiries`.
TanStack Query owns free reports/comparisons; explicit mutations select the goal
and save/delete a change baseline. No provider calls or automatic refresh loop.

The screen explains visitor-day counting, instrumentation gaps, evidence
thresholds and association versus causation. Lead Flow Guard is an in-app flag
on report load. Users test their actual mobile form/button before attributing a
rate drop to a broken flow. New pages can record zero baselines without a claim
of recovered leads. The post-publish link pre-fills only the page path, never
query strings or tracking parameters.

## Daily recognition and click destinations

Optional browser recognition (`data-recognition="true"`) uses a 180-day site-keyed localStorage ID after required consent, with only its site-scoped hash/first-seen timestamp stored server-side. Default snippets retain anonymous visitor-days. Daily new/returning recognized browsers and unrecognized pageview coverage are separate measures; old visits cannot be classified. See backend analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt` for precise counting, consent, retention and verification.

`GET /api/v1/projects/{projectId}/analytics/summary?from&to` adds `audience` rows: `date`, `newBrowsers`, `returningBrowsers`, `unknownPageviews`. `GET /api/v1/projects/{projectId}/analytics/clicks?from&to&page&limit` returns `rows` (`page`, `name`, `host`, `path`, `type`, `clicks`, `visitorDays`, `arrivals`) and `hasMore`. Both require session/project access. Dates are inclusive UTC days, at most 366. Click pages are 1–100 rows, page 1–1000; UI uses 20 rows and project/date-keyed one-minute TanStack caching. No new server cache or paid call.

Collector additions are optional `b` (32 lowercase-hex browser ID), `c` (click boolean), `x` (absolute HTTP(S) destination without credentials). Clicks require event/session/pageview IDs. Configured clicks and automatic outbound links share the same deduplicated event record. Destination query strings/fragments are stripped; JavaScript buttons can specify `data-uplumo-destination`. Internal arrivals are matching later same-tab pageviews within 30 minutes, not proof of causation. External subsequent activity is unavailable.
