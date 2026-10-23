# Domain overview and research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Validates target scopes and returns organic estimates, ranked terms, pages, competitors and suggestions.

## Decisions to preserve

- Validate target, market and filter budget before spend; failed provider tasks may still be billed. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Keep scope semantics explicit: domain, subdomain, subfolder and exact URL do not support identical aggregates. _Why:_ A silent fallback can target the wrong market or metric and produce misleading or wasted research.
- Escape user filters for SQL/provider syntax and cap response size. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.

## Dependencies

DataForSEO paid endpoints, project market and plan checks.

## Shared report purchases

- **Decision:** Overview, keyword pages and page pages use one instance-local
  singleflight group with their full existing cache identity, independent of
  the already-shared optional suggestions route. Recheck cache inside bounded
  90-second detached work; keep each waiter's cancellation independent.
- **Why:** Eight concurrent equivalent cold requests bought eight tasks for
  each report endpoint. Existing Redis caching did not protect simultaneous
  misses, including users with multiple tabs.
- **Preserve:** Existing input/access checks, organization/project/market/view
  separation, no automatic retries, empty-overview retry behavior, cache-outage
  fallback and per-caller scope/display labels. Distinct pages/sorts/filters
  still buy distinct views. Different server processes are not coalesced.
- **Trace:** `lookup.go`, `service.go`, `concurrency_test.go`. Tests use the real
  DataForSEO transport with a synthetic HTTP provider response, reproduce the
  eight-task bug before fixing, assert one task per organization afterwards,
  and cover cancelled preparation and a cancelled first waiter.

## Provider-field delivery audit

This inventory compares the official contracts with `provider.go`, `service.go`,
frontend `domainApi.ts`, `DomainOverviewPage.tsx`, the two domain tables and
`utils.ts` exports. It is source/contract verification, not a live captured
provider response. Nullable or type-specific fields are not guaranteed to be
populated for every target. Existing decoder coverage is intentionally stated
separately from the concurrency fixes above; the delivery gaps below remain.

| Endpoint/field family | Request and decoder | API / screen / export |
| --- | --- | --- |
| Domain rank overview: `metrics.organic.etv`, `count` | Hostname + resolved market + `limit:1`; kept by `organicMetrics` | Rounded `organicTraffic` / `organicKeywords`; two overview tiles; no overview export |
| Overview organic ranking buckets (`pos_1`, `pos_2_3`, `pos_4_10`, then `pos_11_20` through `pos_91_100`), `estimated_paid_traffic_cost`, `is_new`, `is_up`, `is_down`, `is_lost` | Discarded during Go JSON decoding | Missing from API, UI and exports |
| Overview `metrics.paid` | All documented non-clickstream metrics discarded | No paid search summary; `hasData` checks only organic count |
| Ranked keywords / optional suggestions: keyword, search volume, CPC, difficulty, absolute rank, ETV, URL | Kept; difficulty prefers `keyword_properties`; URL path can be derived | Keyword table and CSV/Sheets have seven columns; optional suggestions keep six fields and omit URL |
| Ranked keyword supplemental data | This change retains the provider result type and search intent with each ranked keyword. Other fields in the supplemental list remain undecoded. | The table shows an Organic/Paid badge and Intent column; CSV/Sheets export includes both; conditional rich snippets/sitelinks and the remaining supplemental metrics are still omitted |
| Relevant pages: address, `metrics.organic.etv`, `count` | Kept, traffic/count rounded | Page/traffic/keywords table and three-column export; relative path is derived |
| Relevant pages ranking distribution/change/traffic value and `metrics.paid` | Same narrow `providerMetrics` decoder as overview | Missing from API, UI and export |
| Transport identifiers/status/cost, echoed target/market, counts | Status/cost handled at shared transport; page `total_count` retained | Operational metadata is not a missing customer metric; `fetchedAt` is our retrieval time, not provider dataset freshness |

For overview and pages, the decoder keeps two of the 38 documented non-clickstream
metric leaf fields across the organic and paid objects. That describes schema
coverage, not the proportion of useful populated data in any live response.

**Options and scope:** UI ranked-keyword/pages requests omit `item_types`, so
the documented default includes organic and paid. Ranked keywords now retain
their provider result type and intent, show both in the table, and include them
in export; do not silently describe every returned row as organic. Clickstream is omitted
(default false): its conditional normalized volumes, traffic and demographic
fields are not purchased by these requests. Do not enable the documented double
price to fill this audit. `load_rank_absolute` controls aggregate
`metrics_absolute`; omitting it does not establish a missing row-rank defect.
MCP's ranked-keywords/competitor methods are separate direct provider paths and
are not covered by the page-query cache/coalescing change.

Official contracts:

- [Domain Rank Overview](https://docs.dataforseo.com/v3/dataforseo_labs-google-domain_rank_overview-live/)
- [Ranked Keywords](https://docs.dataforseo.com/v3/dataforseo_labs/google/ranked_keywords/live/)
- [Relevant Pages](https://docs.dataforseo.com/v3/dataforseo_labs-google-relevant_pages-live/)

**Preserve when closing gaps:** no extra paid lookup, optional data stays nullable,
zero is retained, result-type/market/scope labels remain truthful, cache versions
change when old responses lack newly delivered metrics, and provider-shaped
fixtures must reach API/schema/render/export tests. Full live raw-to-screen
comparison and implementation of this supplemental field delivery remain open.

## Label ranked-keyword rows by result type and intent

- **Decision:** Decode `ranked_serp_element.serp_item.type` (organic vs paid)
  and `keyword_data.search_intent_info.main_intent`, expose them on each
  keyword row (`type`, `intent`), and render an Organic/Paid badge plus an
  Intent column in the keyword table. The browser cache key is unchanged
  because the payload fields are additive and old cached rows simply omit them.
- **Why:** The UI request omits `item_types`, so the provider default returns
  both organic and paid rows, but the app dropped the discriminator and intent.
  Customers could not tell which rank came from ads and which from organic
  search — a correctness gap in a headline report, fixed with no extra call.
- **Preserve:** No new provider call; `type`/`intent` stay nullable in the
  schema so cached rows before this change still parse; intent is the raw
  provider label (not normalized) so it is not silently mislabelled. Known
  result types (organic, paid, featured_snippet, local_pack, people_also_ask,
  ai_overview, video, images, twitter, knowledge_graph, related_searches) get a
  short label + a theme-aware `Badge` variant; unknown types show the raw type,
  never a guessed "Organic". No hard-coded colours (dark mode must work).
- **Trace:** `provider.go`, `service.go` (`mapKeyword`),
  `service_test.go:TestKeywordsPageKeepsTypeAndIntent`,
  `frontend/src/client/features/domain/{domainApi.ts,types.ts,
  components/DomainKeywordsTable.tsx,components/ResultTypeBadge.tsx}`.

## Persist paid overviews without adding hot-cache database cost

- **Decision:** Persist only non-empty overview results in
  `go_domain_overviews`. Consult Postgres only after the initial and
  singleflight Redis reads both miss; Redis hits remain one cache read. Serve
  snapshots for at most 30 days.
- **Why:** Reusing a paid overview avoids repeat provider spend after Redis TTL,
  while a Postgres query on every hot hit would add latency and database load.
  Empty results stay retryable because they can reflect transient provider data.
- **Preserve:** Project and organization identity, market/language identity,
  server-only RLS/revokes and project cascade. A Postgres failure falls back to
  the provider and never fails an already-paid answer.
- **Trace:** `service.go` (`Overview`, `loadOverviewSnapshot`),
  `lookup.go`, `README.md`, `flow.md`,
  `00039_domain_overviews.sql`, and
  `service_persistence_test.go`.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Explicit optional ranking suggestions

- **Decision:** Rank Tracking setup does not buy keyword suggestions on mount. Show an optional Find existing keywords action with credit guidance and a free Skip/manual path. The disabled TanStack query fetches only after that action and never retries automatically; failures show the actual API error without removing the manual path.
- **Why:** A browser trace reproduced a paid Labs suggestions request immediately after saving a tracker configuration. A cache miss costs a real task even when the customer never selected a keyword.
- **Preserve:** Keyword selection starts empty and adding selected keywords remains separate from running rank checks. Supported-country checks, existing plan/credit/spend guards and tenant cache keys remain authoritative. An explicit request uses one ranked-keywords task, not a report fan-out. Backend requires `userInitiated: true` to reject older automatic clients; this marker is compatibility protection, not proof of human action or a replacement for authorization.
- **Decision:** Coalesce identical organization/project/target/scope/market suggestions requests using the full existing cache key, recheck cache inside a bounded 90-second shared operation and respect each caller's cancellation independently. Existing nonempty results retain their 12-hour Redis cache; empty/error results are not persisted as success. Coalescing is instance-local and requires the existing single replica.
- **Trace:** `KeywordSuggestionStep.tsx`, `domainApi.ts`, Go `domain/handler.go`, `suggestions.go`; browser setup/Skip/double-click regressions, payload unit contract, legacy-request test and concurrent provider-count tests. Before the fixes, a legacy paid request bought one task and eight concurrent identical requests bought eight. No live paid provider was called.

## Cancellation before a new paid operation

- **Decision:** `KeywordSuggestions` rechecks cancellation after `readCache` and before `suggestions.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `suggestions_test.go` reproduced one paid task after cancellation; it now requires zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).

## Briefly reuse successful empty paid results

- **Decision:** Cache successful empty overviews and suggestions for five minutes, preserving missing metrics as unknown and keeping empty results out of the 30-day durable snapshot.
- **Why:** Identical revisits bought the same empty Labs task every time. A short TTL bounds transient emptiness without repeatedly charging during navigation.
- **Preserve:** Tenant/market/scope keys, non-empty TTLs, error handling and concurrent coalescing. Overview cache entries still need a fetched timestamp.
- **Trace:** `lookup.go`, `service.go`, `suggestions.go`; empty-result call-count and TTL regressions in `service_test.go`.

Provider fixtures include task path and explicit cost, including zero on rejected tasks. Missing task billing metadata now keeps a purchase uncertain rather than proving it was free. Mapping and exact one-call failure assertions remain unchanged.
