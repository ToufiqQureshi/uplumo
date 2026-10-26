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

## Customizable cards that answer "what does this mean"

- **Decision:** the dashboard renders cards from a member's saved layout
  (`dashboardLayout.ts`, `DashboardCustomize.tsx`) instead of the old
  "cards with data first" sort, and adds three cards built from saved data:
  visitors and AI traffic, AI visibility, rank tracking. Each card shows two
  to four numbers and, where it can, one plain sentence on what to do next
  (for example "AI names you but links to other sites").
- **Why:** owner review on 23 Oct 2026 of the live dashboard: it showed GSC
  numbers dropping with no next step, and the AI tracker, rank tracker and
  Uplumo Analytics data a customer already had never appeared. The owner
  asked for a dashboard customers find useful and can shape.
- **Preserve:**
  - Every card reads stored data. No card may call an endpoint that buys
    provider work on page load; `dashboard-spend.spec.ts` lists the paid
    paths and fails if a visit reaches one.
  - A card the member never saw (added in a later release) appears, visible,
    after their saved cards; ids the client no longer knows are ignored.
  - Card failures stay inside the card; the layout falling back to default
    must not block the page.
  - Up/down buttons, not drag and drop: they work with a keyboard and a
    screen reader and need no new dependency.
- **Trace:** `DashboardPage.tsx`, `dashboardLayout.ts` (+ test),
  `DashboardCustomize.tsx`, `InsightCards.tsx`, `dashboardApi.ts` (+ test),
  `frontend/tests/dashboard-spend.spec.ts`; backend
  [`dashboard` decision](../../../../../backend/internal/dashboard/decision.md).
  Checked in a local browser with mocked APIs (cards, Customize, save).
  Not checked against production data yet.

## Keep action compatibility and project state isolated

- **Decision:** Send `userInitiated: true` only through the explicit refresh wrapper; remount the dashboard on project changes using its project key.
- **Why:** Old cached tabs still submit the previous empty automatic request. A completed mutation error otherwise survives a project switch and misrepresents the new project's state.
- **Preserve:** Server authorization, plan/credit checks and zero automatic retries remain required. Internal project switching clears only local view/mutation state; server queries remain project-scoped.
- **Trace:** `dashboardApi.ts`, route `p/$projectId/index.tsx`, API unit test and the real SPA project-switch browser regression in `dashboard-spend.spec.ts`. The marker blocks legacy callers, not malicious clients claiming a click.

## AI visibility card answers who AI cites instead

- **Decision:** AI visibility is the first default card. It reads the latest check that has answers, completed or partial, and shows mentioned and cited shares with the change from the previous check, the top 3 domains AI cited instead of the site (from that check's saved citations), and a next step with a button to the page that fixes it (Content Editor or the tracker). The visitors card shows pageviews in place of goal completions.
- **Why:** The owner wants the dashboard to answer the product's core question (`FOUNDER.md`): is the site cited, who is cited instead, and what to do. Production checks end `partial` when one engine fails, so the old `completed`-only filter showed an empty card over real data. Goal data was removed from analytics with the Missed Enquiries Finder, which broke the type check after merging `main`.
- **Preserve:** Opening the dashboard buys nothing: the answers read is a stored GET shared with the tracker page's query key. `dashboard-spend.spec.ts` matches the paid `tracking/run` POST only, not `tracking/runs/{id}` reads. Own-brand citations never count as "cited instead".
- **Trace:** `aiVisibilityInsight.ts` and its test (partial-run and own-brand mutations both fail it), `InsightCards.tsx`, `dashboardLayout.ts`, `../../../../tests/dashboard-spend.spec.ts`. Not run: the Playwright spec (not in CI) and a check against production data.
