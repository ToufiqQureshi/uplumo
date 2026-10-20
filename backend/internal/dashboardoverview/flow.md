# Dashboard overview: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/dashboard/overview; POST .../dashboard/backlinks/refresh
- **Source:** `handler.go`, `service.go`, `repository.go`

## Request or job flow

Dashboard read gathers project domain, latest audit and stored backlink snapshot from Postgres; an explicit refresh mutation checks access/billing/provider availability and updates the snapshot.

## Dependencies and cost boundary

Postgres snapshots; optional DataForSEO backlinks provider; plans and billing.

## Failure, security and data invariants

- Opening a dashboard must remain read-only and must not trigger paid provider requests.
- Refresh is an explicit action with server-side plan/credit checks and a visible stale/failure state.
- All database reads and refresh writes are project-scoped.

Snapshot refresh: explicit button → existing Backlinks plan gate → scoped
in-flight operation → re-check saved 24-hour snapshot → at most one metered
summary task → Postgres save → invalidate overview. No history task, automatic
retry or page-load purchase. Failures retain saved data and surface an error.
