# Backlink research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Serves project-scoped backlink summaries, history, referring-domain and top-page data.

## Decisions to preserve

- Validate target/filter limits and verify plan before any billed provider task. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Preserve null/unsupported metrics by target scope; never present them as zero. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Do not retry a paid task automatically; partial provider limits must be visible. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

DataForSEO paid endpoints and usage metering; project plan/credit checks.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Preserve bought blocks and coalesce table purchases

- **Decision:** Rows and referring domains attempt a complete small-profile
  fetch through the existing unfiltered first-block cache. `loadPage` shares
  equivalent cold misses through the service's singleflight group with a
  three-minute detached deadline and a second cache check.
- **Why:** When a cached overview undercounted a profile, the speculative
  1,000-row purchase was discarded and bought again immediately and on later
  page-size changes. The regression observed three purchases for two equivalent
  first-block views. Eight simultaneous table requests also bought eight tasks
  instead of one per organization.
- **Preserve:** Incomplete blocks never masquerade as a complete list. Full
  query, organization, kind, sort, mode and block stay in the identity; another
  view may still need its own provider task for a large profile. No automatic
  paid retry, no plan/price/cache-lifetime change, and no credits held for a
  caller cancelled before registration. Shared work can finish for active
  waiters, but separate server processes are not coalesced.
- **Trace:** `service.go`, `whole_list.go`, `table_purchase_test.go`. Tests use a
  provider-shaped local HTTP response, reproduce undercounts/concurrent misses
  before the fix, check organization isolation and cancellation, and run with
  the race detector. Post-deployment raw-response and ledger checks remain
  separate live validation.

## Cancellation before a new paid operation

- **Decision:** `Overview` rechecks cancellation after `readCache` and before `flights.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `cancellation_test.go` reproduced both summary/history tasks after cancellation; it now requires zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).
