# Project dashboard: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. Opening a project dashboard starts project-scoped queries: activation,
   overview (audit, backlinks) and the member's layout
   (`dashboard/layout/get`). The page waits for all three so cards do not
   reshuffle.
2. `resolveLayout` turns the saved layout (or `null`) into the card order:
   saved cards first, unknown ids dropped, new cards appended and shown.
3. Each visible card loads its own data from saved records:
   - Search performance: GSC report, last 28 days.
   - Visitors and AI traffic: `analytics/summary`, last 30 days.
   - AI visibility: `ai-search/tracking` state (latest completed or partial check and the one before), then `GET ai-search/tracking/runs/{runId}` for that check's saved answers to list the domains AI cited instead. Both are free reads; 0 paid calls.
   - Rank tracking: `rank-tracking/configs/summaries`, then
     `results/latest` for the first active tracker with keywords.
   - Site audit and backlinks: `dashboard/overview`.
   - Google Analytics: the linked GA4 property.
4. **Customize** edits a local copy; **Save layout** sends one
   `dashboard/layout/save`; **Reset to default** sends an empty list.
5. A user-triggered backlink refresh can call the backend provider; ordinary
   dashboard reads use stored snapshots.

## Paid calls per action

| Action                            | Paid provider calls           |
| --------------------------------- | ----------------------------- |
| Open or reload the dashboard      | 0                             |
| Save or reset the layout          | 0                             |
| Click "Refresh backlink snapshot" | at most 1 (backlinks summary) |

`frontend/tests/dashboard-spend.spec.ts` asserts that a visit and a reload
reach no paid endpoint.

## API and service dependencies

Go packages: `dashboard`, `dashboardoverview`, `activation`, `ga4`, `gsc`,
`analytics`, `aisearch`, `ranktracking`; optional backlink refresh uses
`backlinks`.

## Cost, security and failure boundaries

- Dashboard loads must not trigger expensive provider refreshes implicitly; refresh stays an explicit action.
- Refresh cost and plan checks stay server-side; stale snapshots should be labeled, not silently refreshed.
- Activation mutations are user/project-scoped and first-use timestamps remain stable.
- A layout that fails to load falls back to the default; a card whose data
  fails shows its own error without hiding the others.

Snapshot refresh: explicit button + `userInitiated: true` -> strict request
validation -> existing Backlinks plan gate -> organization/project/domain-scoped
in-flight operation -> re-check saved 24-hour snapshot -> at most one metered
summary task -> Postgres save -> invalidate overview. No history task, automatic
retry or page-load purchase. Failures retain saved data and surface an error.
