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
