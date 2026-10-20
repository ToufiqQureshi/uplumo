# Uplumo Analytics: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when behavior or an invariant changes.

## Why the feature is structured this way

A first-party tracker and project analytics dashboard with short tab sessions.

## Decisions to preserve

- Do not add cookies or persist raw IP addresses or session tokens. The browser uses a random tab-scoped ID with 30-minute inactivity and 24-hour maximum expiry; the server stores only a site-scoped hash. Visitor-day IDs remain daily salted hashes. This links engagement and goals within a short session. Optional browser recognition is separately described below and requires explicit enablement.
- Collection is public by design, so preserve bot/domain checks, payload bounds, and shared per-IP/per-site rate limits.
- Custom events must use the same validation and rate limits as pageviews; deduplicate retries without suppressing separate user actions.
- Respect Do Not Track and Global Privacy Control before storage or collection.
- Do not forward events to advertising or analytics vendors. This feature does not call paid SEO providers.

## Verification pointers

- `analyticsApi.test.ts`
- `presentation.test.ts`

## Daily audience and destination reports

Decision: show a separate daily recognized-browser table and paged click-destination report, using the same selected UTC dates as the existing dashboard. New/returning deduplicate per browser per day; unrecognized pageviews are explicitly a different coverage measure. Do not relabel existing anonymous visitor-day totals as people or add daily categories to claim date-range unique users.

Why: existing page engagement, source/campaign attribution and event counts already work; the missing pieces are cross-day browser recognition and click targets. This PR deliberately excludes AI, revenue, funnels, alerts and GA4 redesign. Existing custom events are labeled as events rather than automatically implying every programmatic event was a button click.

Preserve: recognition is off in the default snippet; optional setup explains 180-day first-party storage and host-managed consent. No UI/content is injected on customer websites. Historic visits remain unclassified. External destinations show no arrival/success claim; internal arrival means only an observed matching same-tab pageview within 30 minutes. Tables scroll inside their cards on mobile; daily rows expand from the latest seven days. Click pages are bounded at 20 groups and cached by project/date for one minute with existing TanStack patterns. Error states retain available data and expose retry.

Trace: `AudienceJourneyCards.tsx`, `AnalyticsPage.tsx`, `TrackingCodeCard.tsx`, `analyticsApi.ts`, API regressions, tracker regressions and `tests/analytics-audience.spec.ts`. Backend semantics and retention are defined in `backend/internal/analytics/decision.md`; full verification handover lives in its `AUDIENCE_DESTINATIONS.txt`.
