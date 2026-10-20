# Uplumo Analytics: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/analytics/site; POST /collect; GET /api/v1/projects/{projectId}/analytics/summary
- **Source:** `handler.go`, `service.go`, `repository.go`

## Request or job flow

Analytics UI creates a project site key; the tracker assigns a random tab session, measures visible page time and posts bounded events; service validates bot/domain/rate rules and extracts the five allowed UTM fields; repository stores only site-scoped hashes; summary queries aggregate first-touch sources, engagement and selected goal events.

## Dependencies and cost boundary

Postgres analytics tables; Redis daily salt/rate-limit counters; no paid SEO provider.

## Failure, security and data invariants

- Do not store cookies, raw IP addresses, raw session/pageview/event IDs, full URLs or raw click IDs.
- Public ingestion must keep same-origin/domain validation, bot filtering, payload bounds and per-IP/site limits.
- Pageviews and custom events must stay separate and remain idempotent across tracker retries.
- Engagement contains visible time only; browser-hidden time and generic no-activity sessions must not be counted.

## Enquiry workflow

Member selects an event → free report aggregates matched visitor-days → member
records a baseline before editing a page → repository freezes that week's
metrics under the project lock → comparison becomes ready after seven complete
post-change days. Missing instrumentation and small samples stay explicit;
there is no background email or automatic site edit.

## Daily recognition and click destinations

Optional browser recognition (`data-recognition="true"`) uses a 180-day site-keyed localStorage ID after required consent, with only its site-scoped hash/first-seen timestamp stored server-side. Default snippets retain anonymous visitor-days. Daily new/returning recognized browsers and unrecognized pageview coverage are separate measures; old visits cannot be classified. See backend analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt` for precise counting, consent, retention and verification.

`GET /api/v1/projects/{projectId}/analytics/summary?from&to` adds `audience` rows: `date`, `newBrowsers`, `returningBrowsers`, `unknownPageviews`. `GET /api/v1/projects/{projectId}/analytics/clicks?from&to&page&limit` returns `rows` (`page`, `name`, `host`, `path`, `type`, `clicks`, `visitorDays`, `arrivals`) and `hasMore`. Both require session/project access. Dates are inclusive UTC days, at most 366. Click pages are 1–100 rows, page 1–1000; UI uses 20 rows and project/date-keyed one-minute TanStack caching. No new server cache or paid call.

Collector additions are optional `b` (32 lowercase-hex browser ID), `c` (click boolean), `x` (absolute HTTP(S) destination without credentials). Clicks require event/session/pageview IDs. Configured clicks and automatic outbound links share the same deduplicated event record. Destination query strings/fragments are stripped; JavaScript buttons can specify `data-uplumo-destination`. Internal arrivals are matching later same-tab pageviews within 30 minutes, not proof of causation. External subsequent activity is unavailable.
