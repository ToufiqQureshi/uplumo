# Domain overview and research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Shows organic estimates, keyword/page tables and filtered domain research.

## Decisions to preserve

- Domain research can incur provider charges; validate target, country and filter budget before calling the provider. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Keep project market defaults explicit and reject unsupported country/language pairs. _Why:_ A silent fallback can target the wrong market or metric and produce misleading or wasted research.
- Never imply provider estimates are exact Search Console measurements. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Verification pointers

- `domainRouteState.test.ts`
- `components/domainResultLinks.test.ts`

## Reuse paid report snapshots

- **Decision:** Align overview/keyword/page query freshness and retention with
  the existing 12-hour server cache, and disable `retryOnMount` on these paid
  queries. An overview without data stays stale; manual Retry remains explicit.
- **Why:** Returning after six minutes reissued all three paid endpoint calls,
  and remounting a previously failed query retried it despite `retry: false`.
  The server cache cannot recover the cost of repeating a failed paid request.
- **Preserve:** Project/market/view query identities, no focus/reconnect retries,
  explicit manual retry, existing empty-overview retry behavior and unchanged
  request/response schemas. This does not make different page/filter views free.
- **Trace:** The three `hooks/useDomain*Query.ts` files and the mounted
  `domain-cache.test.ts` regressions (fail before fixes, pass afterwards).

## Explicit suggestions request contract

- **Decision:** Only the keyword-suggestions API wrapper adds `userInitiated: true`; overview/page requests retain their existing contracts.
- **Why:** Older Rank Tracking bundles automatically called this paid endpoint on setup. The Go handler now rejects those legacy payloads before provider work; the wrapper's payload test guards compatibility with its strict decoder.
- **Preserve:** This marker is not authorization or proof of a click. The caller must remain an explicit action with no automatic retry; existing tenant, market, plan and spend checks still apply.
- **Trace:** `domainApi.ts`, `domainApi.test.ts`; [backend cost/concurrency decision](../../../../../backend/internal/domain/decision.md) and [Rank Tracking UX decision](../rank-tracking/decision.md).


## Label organic and paid ranked keywords

- **Decision:** Keep each provider row's organic/paid result type and search intent; show both in the ranked-keyword table and exports.
- **Why:** The paid ranked-keywords task already returns both fields, and dropping them made paid rows look organic.
- **Preserve:** No extra provider call and no change to the endpoint request; unknown provider types remain explicit.
- **Trace:** `domainApi.ts`, `DomainKeywordsTable.tsx`, and `ResultTypeBadge.tsx`.
