# Uplumo Analytics

A cookie-free first-party tracker and project analytics dashboard.

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
