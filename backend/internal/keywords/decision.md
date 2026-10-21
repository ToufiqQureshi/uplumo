# Keyword and local business research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Owns keyword/serp research, location resolution, saved keywords and local business provider operations.

## Decisions to preserve

- Validate every billed task before provider call; no blind retries of paid task-post endpoints. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Cache keys include project/organization, geography and all result-changing options. _Why:_ A cache can be stale or absent, so it must not replace the authoritative source or change the result silently.
- Keep saved keyword rows and tags project-scoped; guard batches/caps and null metrics. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.

## Dependencies

DataForSEO; Google Ads for selected markets; Postgres saved rows; Redis caches; platform market registry.

## Preserve paid seed data and coalesce SERP purchases

- **Decision:** Parse `seed_keyword_data` ahead of suggestions, retain measured
  clickstream zero, and invalidate older research snapshots with cache version 6.
  Concurrent equivalent SERPs at the same depth share the existing singleflight
  group, under the organization/project/market cache key plus depth.
- **Why:** The provider returns seed data outside `items`; ignoring it removed
  exact metrics the customer already paid for. Eight concurrent cold SERPs
  previously bought eight tasks. A measured zero is not missing data.
- **Preserve:** No extra overview call, no automatic paid retry, separate
  organization/market/depth identities, bounded detached work for active waiters,
  and no new purchase for an already-cancelled caller. Deeper empty results must
  not evict a useful shallower snapshot. Coalescing is per server process.
- **Trace:** `research_provider.go`, `research.go`, `research_serp.go` and the
  provider/SERP regression tests. Raw provider-shaped fixtures are synthetic;
  post-deployment comparison with live raw responses is still required.
  `research_test.go:TestRowMapping` distinguishes measured clickstream zero from
  missing clickstream volume: zero must remain zero, while only absent volume
  falls back to the national figure. This corrects the old opposite assertion
  that failed the merged PR's Go CI; it does not relax the missing-data case.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Cancellation before a new paid operation

- **Decision:** `Research` rechecks cancellation after `cached` and before `flights.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `research_cancellation_test.go` interrupts the cache read and uses real saved-keyword test Postgres; the former billed suggestions call must remain at zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).
