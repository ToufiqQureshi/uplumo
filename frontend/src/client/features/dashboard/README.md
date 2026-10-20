# Project dashboard

Combines project onboarding state, audit/backlink summaries and GA4 overview cards.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `DashboardPage.tsx`
- `dashboardApi.ts`
- `ga4DashboardApi.ts`
- `backend/internal/dashboardoverview`
- `backend/internal/activation`
- `backend/internal/ga4`

Backlink snapshots refresh only after a button click, with credit guidance and
no automatic retry. Missing provider, stale data and refresh errors are visible;
opening/reloading a dashboard never buys a backlink task.
