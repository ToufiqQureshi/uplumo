# Project dashboard

The first page of a project: shortcuts, the setup checklist and a grid of
cards (Google Search, visitors and AI traffic, AI visibility, rank tracking,
site audit, backlinks, Google Analytics). Each member can hide, show and
reorder the cards with **Customize**; the layout is saved per member per
project.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `DashboardPage.tsx`: loads activation, overview and layout; renders cards in
  the saved order.
- `dashboardLayout.ts`: the card list (`DASHBOARD_CARDS`, default order) and
  `resolveLayout` / `moveCard`.
- `DashboardCustomize.tsx`: the Customize sheet (switch per card, up/down,
  Save, Reset to default).
- `InsightCards.tsx`: visitors and AI traffic (Uplumo Analytics), AI
  visibility (latest saved AI tracking check), rank tracking (first active
  tracker).
- `DashboardCards.tsx`, `Ga4Card.tsx`: Search Console, site audit, backlink
  pulse, Google Analytics.
- `dashboardApi.ts`, `ga4DashboardApi.ts`
- Backend: `backend/internal/dashboard` (layouts), `dashboardoverview`
  (audit and backlinks), `activation`, `ga4`, `analytics`, `aisearch`,
  `ranktracking`.

## Adding a card

Add its id and label to `DASHBOARD_CARDS` and a renderer to `renderCard` in
`DashboardPage.tsx`. The server needs no change. The card must read saved
data only: nothing on the dashboard may buy provider work on load.

Backlink snapshots refresh only after a button click, with credit guidance and
no automatic retry. Missing provider, stale data and refresh errors are visible;
opening/reloading a dashboard never buys a backlink task.

Refresh sends an explicit-action marker; older automatic empty requests are rejected.
Switching projects resets local refresh/error state.
