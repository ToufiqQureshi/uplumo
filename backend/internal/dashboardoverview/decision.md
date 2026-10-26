# Dashboard overview: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Reads audit and backlink snapshots for the project dashboard; supports an explicit backlink refresh route.

## Decisions to preserve

- Opening a dashboard must remain read-only and must not trigger paid provider requests. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Refresh is an explicit action with server-side plan/credit checks and a visible stale/failure state. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- All database reads and refresh writes are project-scoped. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.

## Dependencies

Postgres snapshots; optional DataForSEO backlinks provider; plans and billing.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Require explicit, authorized snapshot purchases

- **Decision:** Keep dashboard reads free of paid work. Refresh checks the existing Backlinks plan gate; missing provider returns unavailable, and stale/provider failures remain visible. Coalesce concurrent requests per organization/project/domain and re-check the 24-hour saved snapshot inside a bounded 90-second operation.
- **Why:** A real browser trace showed a visit-triggered refresh; its route also omitted the normal Backlinks paid gate. Concurrent snapshot misses could each buy a task, and stale failures previously claimed success.
- **Preserve:** Zero tasks on ordinary reads or fresh snapshot reuse; one summary task for an explicit stale/missing refresh, no history or automatic paid retry. Retain stale data on failure and tenant predicates on reads/writes. Coalescing is instance-local, so one replica remains required.
- **Trace:** `service.go`, `handler.go`, `types.go`, real-Postgres `repository_test.go`; root-mounted plan regression in `httpapi/backlinks_test.go`; client browser regression.

The refresh payload must contain `userInitiated: true`; legacy empty requests
from an already-open older SPA are rejected before any plan/provider work. This
is a versioned action contract, not proof of a human click or an authorization
replacement. The current project domain participates in coalescing and snapshot
lookup: a previous domain finishing later cannot hide the current domain's data.
The domain-switch regression uses real Postgres and a blocked provider; malformed
and legacy request tests assert zero provider calls without an earlier plan gate.

## Read audits from `go_audits`

- **Decision:** `Repository.LatestAudit` reads the newest row of `go_audits` and
  its issues from `go_audit_issues` (by that audit's id), formatting
  `started_at` as the same UTC ISO string the client already parses.
- **Why:** `internal/audit` writes only `go_audits`; the legacy `audits` table
  from migration 1 is no longer written. Live on 23 Oct 2026 a project with two
  finished crawls (21 Oct) still showed "Run an audit" because the API returned
  `audit: null`. The old test built its own `audits` table, so it hid the bug.
- **Preserve:** Reads stay free (no provider call) and project-scoped; the
  newest crawl of this project wins, never another project's or a legacy row.
- **Trace:** `repository.go`; `repository_test.go` now seeds `go_audits` with an
  older crawl, another project's newer crawl and a legacy `audits` row, and fails
  on the old query.

## Cancellation before a new paid operation

- **Decision:** `RefreshBacklinkSnapshot` rechecks cancellation after `ProjectDomain` and before `refreshes.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `service_test.go` cancels during project lookup and drains the scoped flight; the former snapshot purchase must remain at zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).
