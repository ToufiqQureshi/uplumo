# AI Visibility and Prompt Explorer: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Runs Brand Lookup and Prompt Explorer, shapes answer-engine data, and caches successful results.

## Decisions to preserve

- Plan and request validation must happen before provider spend. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Keep all billed calls attributed to the project organization; never blindly retry paid POSTs. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Cache only successful complete results; preserve per-platform partial failures and safe citation URLs. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- A cached prompt may contain user-entered text; retain expiry and privacy-erasure limitations in the feature README. _Why:_ Prompts can include customer research, so cache expiry and erasure behavior limit retention of that content.
- Save history only from a complete, data-bearing Brand Lookup and key it with organization, project and every lookup option that changes the result. _Why:_ Partial results must not be frozen as a full monthly trend or mixed across customer scopes.
- History reads use Redis aggregates and are best effort; they must not call the paid provider or fail the current lookup. _Why:_ Viewing an existing result should not create another charge or turn a secondary chart outage into a failed core lookup.
- Retain the 13-month limit and 400-day Redis expiry. _Why:_ A fixed retention window bounds storage and avoids retaining user research indefinitely.

## Dependencies

Redis cache; DataForSEO AI endpoints; plans/billing.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Use built-in integer bounds

- **Decision:** Replace the local clamp helper with Go built-in `min`/`max` at its four callers.
- **Why:** The wrapper duplicated language functionality.
- **Preserve:** Keep identical limits (1–1000, 1–20, 1–10 and 256–4096), provider call counts, billing guards, retries and cache behavior.
- **Trace:** `provider.go` and existing AI search provider tests; no live paid calls.

## Cancellation before a new paid operation

- **Decision:** `BrandLookup` rechecks cancellation after `cacheGet` and before `flights.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `cancellation_test.go` interrupts the first Redis GET with a hook and drains the flight; the former six AI tasks must remain at zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).

## Share concurrent Prompt Explorer purchases

- **Decision:** Coalesce identical Prompt Explorer cache misses per server and
  organization/project before budget reservation, using the existing
  `singleflight.Group` with a three-minute detached deadline. Recheck each model
  cache inside the flight; keep caller order and brand highlighting separate.
- **Why:** Eight concurrent same-scope requests bought eight answers per model.
  A shared purchase gives one answer per model while retaining tenant isolation.
- **Preserve:** Failed answers are not cached, cancelled requests do not start
  new purchases, existing bounded web-search behavior remains explicit, and
  neither prices nor model selection change. Multiple processes remain a gap.
- **Trace:** `service.go`, `prompt_concurrency_test.go`; fake-provider tests use
  real Redis and assert one task per model for one organization, two for two
  organizations, caller order/highlight independence and no cancelled purchase.
  No live paid-provider comparison or deployed verification was performed.

## Preserve the original Gemini citation source

- **Decision:** Decode annotation `direct_url` and prefer it when safe, with
  safe `url` fallback. Version only Gemini's prompt cache namespace.
- **Why:** DFS returns a Vertex redirect in `url`; dropping `direct_url` made
  publisher domains and brand highlighting describe Google’s redirect instead
  of the cited source. Previously cached payloads cannot recover the lost URL.
- **Preserve:** Do not fetch citations, allow credentials/unsafe schemes, add
  paid calls during shaping, change other model caches or alter prices. A
  fresh explicit Gemini lookup may buy an answer rather than use old cache.
- **Trace:** `types.go`, `prompt.go`, `service.go`, `citation_source_test.go`;
  provider-shaped HTTP regression failed before the fix and checks the original
  source, absent/unsafe/credentialed fallback and both URLs unsafe. Official
  contract: https://docs.dataforseo.com/v3/ai_optimization-gemini-llm_responses-live/.
