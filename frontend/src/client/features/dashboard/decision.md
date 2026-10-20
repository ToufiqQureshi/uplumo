# Project dashboard: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Combines project onboarding state, audit/backlink summaries and GA4 overview cards.

## Decisions to preserve

- Dashboard loads must not trigger expensive provider refreshes implicitly; refresh stays an explicit action. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Refresh cost and plan checks stay server-side; stale snapshots should be labeled, not silently refreshed. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Activation mutations are user/project-scoped and first-use timestamps remain stable. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.

## Verification pointers

- `dashboardApi.test.ts`
- `dashboardSteps.test.ts`

## Let customers choose a backlink refresh

- **Decision:** Remove the mount effect that purchased missing/stale snapshots. Show a pending-disabled refresh button, credit guidance and explicit stale/error state; use the server availability flag, defaulting absent flags to false. Disable automatic mutation retries.
- **Why:** Opening or reloading the dashboard must not silently spend customer credits. The new browser regression failed on the existing page-open request before the fix.
- **Preserve:** Keep stored results and project-scoped queries. No paid work on visits, reloads, focus or error; a click sends one mutation. Errors remain visible and do not remove saved metrics.
- **Trace:** `DashboardPage.tsx`, `DashboardCards.tsx`, `dashboardApi.ts`; `dashboardApi.test.ts` and `frontend/tests/dashboard-spend.spec.ts`; backend dashboardoverview decision.
