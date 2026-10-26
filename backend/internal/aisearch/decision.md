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

## Brand Lookup: one totals task first, keep what it returns

- **Decision:** Replace the per-platform `aggregated_metrics` and the always-bought
  `top_pages` with one `target_metrics` task for every platform. Stop when no
  platform has a mention. Buy a prompt sample (up to 100, `order_by
  ai_search_volume,desc`) only for platforms with mentions and, for ChatGPT,
  only in its US English market; buy `top_pages` only when the sample holds
  fewer answers than the platform's mention total, otherwise count cited pages
  from the sample. Return the totals task's `sourceSites`, `retrievedSites` and
  `brandsAlongside`, flag each prompt `branded`, count branded/unbranded
  prompts, and cap branded and unbranded prompts separately (25 each per
  platform). Typical cost hint 0.29 USD (about 372 credits).
- **Why:** Live comparison 22 Oct 2026, India: the old six tasks billed $0.693
  for dellaresorts.com, $0.31 of it for ChatGPT data the result drops outside
  the US, and about $0.60 for revmerito.com, which AI never mentions. The new
  flow billed $0.283 (2 tasks) and $0.101 (1 task). The old code read only the
  `platform` group of the totals call it paid for and dropped the cited sites
  (Instagram 31, TripAdvisor 30, MakeMyTrip 22, Agoda 16 answers), retrieved
  sites and co-mentioned brands. 39 of 82 prompts only named the brand; the
  43 unbranded ones ("best resort in lonavala", 2,900 searches) are where AI
  recommends the business to new customers and were crowded out by the cap.
- **Preserve:** No call before validation; one hold for the largest fan-out;
  zero-mention lookups buy one task and are not cached; a failed sample or
  pages call keeps the platform's other data and is not cached; billing issues
  and cancellation still stop at the first failing call; cache and history
  keys are unchanged (old cached results lack the new fields for up to 24 h);
  the branded split is a heuristic (a domain-label prefix word of 4+ letters,
  or the keyword phrase) and is never billed or used for totals.
- **Trace:** `provider.go` (`targetMetrics`, `mentionsSearch`), `service.go`
  (`loadBrandLookup`, `fetchPlatform`, `fetchCross`), `brand_terms.go`,
  `shape.go`, `service_test.go`, `brand_terms_test.go`,
  `../billing/costs.go`, `../httpapi/aisearch_test.go`. Mutation-checked (9
  mutations, each failed a test). Run once against the live API with the Go
  code (dellaresorts.com: 2 tasks, sites/brands/branded split as above); the
  revmerito.com live run hit the DataForSEO account's $2/day limit, so its
  one-task path is covered by tests and the earlier raw replay only. Not yet
  verified on production (its DataForSEO key returned 401 on 22 Oct).

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

## Share queue supervision

- **Decision:** Use the common jobs supervisor for both AI tracking workers.
- **Why:** Audit/rank now need the same outage recovery; keeping one loop avoids drift and removes duplicate lifecycle code.
- **Preserve:** Two workers, five-second restart delay, cancellation, existing task checkpoints and paid-call limits.
- **Trace:** `tracking_worker.go`, `platform/jobs/worker.go` and its injected-database-failure integration test.

## Finished tracker answers stop the polling

- **Decision:** `parseTrackingAnswer` reads nested SERP `items` as raw JSON, so keyword-only lists (`related_searches`, `people_also_search`) no longer fail a Google AI Overview answer. A finished task that fails or cannot be read is marked failed (`provider_failed`, `unreadable_result`) and returns `errAnswerFailed`, so the run stops polling it.
- **Why:** Production runs on 24 and 25 Oct 2026 bought three AI Overview SERP tasks each, then polled for two hours and ended `collection_timeout`: every AI Overview result failed to decode, so the customer paid for answers never shown. A failed answer was also still counted as pending, so even the existing `provider_failed` path waited for the next poll.
- **Preserve:** Polling stays free and never replays a paid POST; a network error on `task_get` still retries on the next poll; only queued tasks (20100, 40601, 40602) keep a run waiting.
- **Trace:** `tracking_worker.go` (`parseTrackingAnswer`, `collect`, `failAnswer`); `TestTrackingEvidenceSeparatesCitationsFromMentions` failed on the keyword-only items before the fix; `TestTrackingUnreadableResultFailsInsteadOfPollingUntilTimeout` timed out (`collection_timeout`) with the stop-polling check removed. Not verified against a live AI Overview task yet: after deploy, the next daily run on the test workspace is the check.
