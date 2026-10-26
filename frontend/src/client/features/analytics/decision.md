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

## Lead with observed activity and measurement gaps

- **Decision:** put a compact traffic overview before the statistics, keep visitor
  actions in the main flow, and group secondary audience/acquisition reports in
  an expandable section. Surface failed refreshes beside cached data.
- **Why:** the live page showed one pageview with zero sessions and repeated
  event names, surrounded by empty reports. Owners needed an explanation and a
  setup action before detailed counting caveats.
- **Preserve:** no new requests or data collection, no event-name merging, no
  inferred leads/revenue, no relabeling visitor-days as unique people. AI referrals
  remain distinct from visibility and do not establish conversion performance.
  Existing scoped queries, consent settings, destinations and all reports remain.
- **Trace:** `AnalyticsOverview.tsx`, `AnalyticsPage.tsx`, `AnalyticsCards.tsx`;
  `AnalyticsOverview.test.tsx` covers missing measurement, leader selection,
  case inconsistencies and outcome labels. Audience browser regressions open the
  secondary disclosure before inspecting the existing report. Visual browser
  acceptance must be recorded separately from unit/type checks.

## Overview: fix tracking first, then enquiries

- **Decision:** `AnalyticsOverview` now shows these, in order:
  1. A "Fix this first" card with 3 steps, shown only when page views arrive but no visits are recorded.
  2. A verdict: who visited, how many tried to contact the business, where most came from, and one next step.
  3. Four tiles: visitors, enquiries, enquiry rate, and visits from AI assistants.

  `visitSummary.ts` merges event names that differ only by letter case (`whatsapp`/`WhatsApp`) and says so. Enquiries are events whose names match WhatsApp, call, phone, form, lead, contact, enquiry, book, quote, submit, sign-up, order or checkout.

- **Why:** The owner said the page gave no decision and the disclaimers ("does not prove…") confused users. The approved design is artboard 2 of the Traffic redesign.
- **Preserve:**
  - Broken tracking is still flagged first.
  - Merged event counts take the largest visitor-day count, not the sum, so the same person is not counted twice.
  - The full tables below are unchanged.
  - No new request, no AI call.
  - Enquiries are clicks on contact buttons, not confirmed sales. The tile hint says "taps".
- **Trace:** `visitSummary.ts`, `AnalyticsOverview.tsx`, `AnalyticsOverview.test.tsx`. Each of these 5 mutations fails the tests: tracking rule, case merge, visitor max, enquiry filter, booking pattern. The old tests that asserted the disclaimer text were replaced on purpose.
