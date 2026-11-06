# DataForSEO: Uplumo cost, customer value and owner reporting

Reviewed against commit `92d2cabd237d8b0491550f166ca5e27de53734a7` on 6 November 2026. This is an internal, code-based decision guide. It changes no paid calls, credits, plan limits, schedules, persistence or UI. No real provider call was made for this review.

Read [the practical provider guide](README.md) before changing an integration. Consult the exact endpoint in [the full official reference snapshot](reference.md) before choosing fields, limits, modes or paid options. Published prices are research snapshots, not a second runtime price catalog; production estimates stay in `pricing.go`, customer prices/limits in `plans/catalog.go`.

## Answer to the owner

**Nahi, Uplumo DataForSEO ka saara raw output show ya retain nahi karta.** Selected fields typed structs mein map hote hain; kuch data cache/database mein bachta hai, kuch analysis/CSV mein milta hai, aur kuch decoding, local filtering ya row caps par lose hota hai. Rank Tracking mein 20 organic competitors retain hote hain, hover mein 5; Domain Overview organic metrics ka subset map karta hai; Keywords agent researched candidates se at most 20 recommendations present karta hai. Backlinks CSV screen se zyada mapped fields expose karta hai. Details and evidence are below.

Har raw field ko main screen par chipkana customer value nahi. Har paid field ko bina reason discard karna bhi acceptable nahi. Uplumo ka aim: **customer ke intended decision ke liye complete, honest, reusable evidence, with the fewest justified paid calls**. A concise recommendation can be valuable while the supporting dataset stays available through an existing detail/export/analysis path. An invented metric or hidden reduction in coverage is not a saving.

Do not report a percentage of "all DataForSEO data shown" from this review. There is no captured live-response denominator, and counting field names treats a timestamp and an entire competitor list as equally valuable. This review proves specific omissions; it is not an exhaustive production field-parity audit. OnPage, content briefs, every AI feature and every export remain unverified here unless explicitly listed.

## Mandatory workflow for every agent

Before proposing or implementing DataForSEO work:

1. Read this file and the complete practical guide in `README.md`; read the relevant request **and response** sections of `reference.md`, including nested fields, conditional requirements and pricing caveats. Refresh current official docs for new endpoints or changed prices. Do not load a 28 MB snapshot into every prompt; locate the endpoint and read its complete relevant contract.
2. Trace the feature from UI action to handler/service/shared provider, spend guard, usage recorder, persistence/cache and UI/export. Review existing behavior before claiming a new optimization. Follow `CLAUDE.md` authorization rules for runtime/money changes.
3. Give the owner the report template below, with baseline calls, proposed calls, provider dollars, customer credit effect, monthly workload and value tradeoffs. Label observed vs estimated facts. If the cheaper method changes the product promise, state the change explicitly.
4. Account for fetched, stored, used, shown/exported, filtered and dropped fields/rows. "It's in the Go struct" does not prove it is saved or shown; "it's in Redis" does not prove durable retention. Link code for every confirmed claim.
5. Implement only the authorized change. Prove fewer purchases or more reusable value with realistic provider fixtures and paid-call counts at the public entry point. Double click, concurrent request, cached revisit, failure and scope changes matter. Production-paid verification needs the owner's authorized scope/budget; never buy data just to validate docs.
6. Tell the owner what changed, what remains missing and what must be measured after rollout. Do not declare "full value delivered" until the supported user workflow and coverage have actually been verified.

Notify the owner in the working conversation and the PR. Do not create a new email/Slack notification integration or expose internal provider economics to customers. Customer-facing messages should explain freshness, scope, completeness and credits when they affect the purchase decision.

## Owner report: required contents

Use this in the proposal and PR when a DataForSEO integration is added or changed. For this documentation-only PR: provider calls **0**, customer credits **0**, no runtime saving shipped; the opportunities below are proposals.

```text
User job / feature:
Code evidence and baseline date:
Technique to try:
Already implemented safeguards (do not count them again as new savings):
Current -> proposed paid calls per action / scheduled run:
Current -> proposed rows, pages, keywords, targets, devices, locations:
Current -> proposed provider USD (rate source/date + billing unit):
Monthly workload and estimated monthly provider saving:
Customer credits: unchanged / lower / changed quote or hold (with reason):
Customer value preserved: exact decision, evidence and accessible output:
Data not requested / filtered before billing:
Data received but not persisted / used / shown / exported:
Freshness, completeness, latency, scope and accuracy tradeoffs:
Unknowns / pricing disagreement / actual-cost reconciliation needed:
Validation: fake-provider calls, coverage fixture, cache/concurrency behavior:
Owner decision needed, if this changes scope, price or promised behavior:
```

Never put API credentials, customer payloads, tokens or PII into a PR. Use synthetic or authorized redacted fixtures. Internally retaining reusable data should have tenant isolation, a retention period, access control and a storage budget; it is not permission to retain full raw responses forever.

## Raw-data accountability: distinguish the loss stage

| State | What it means | Economic meaning | Required explanation |
| --- | --- | --- | --- |
| Not requested | No paid option, smaller depth or fewer targets | Can reduce provider bill where units/flags are billable | What user question can no longer be answered? |
| Server-filtered before returned rows are billed | Provider filters candidates | Can reduce per-row charges; often does not remove task fee | Which candidates are excluded and why? |
| Received then discarded | Typed decoder, array slicing, local filter | Usually **zero refund**; bytes/rows were already bought | Is the lost evidence useful? Preserve/map it or justify its omission. |
| Preserved but not primary UI | Cache/DB retains mapped data; details/export or analysis may use it | Can keep user value without UI overload | Name the actual access path and retention; do not assume one exists. |
| Used only for analysis | Input to an evidence-backed derived metric/recommendation | May be valuable despite no raw column | Explain methodology, uncertainty and provenance. |
| Neither shown nor used | No verified consumer | Potential paid-data waste | Stop requesting a paid option if dispensable, or propose a useful consumer. |
| Unavailable or null | Provider did not supply it | Not an application drop by itself | Do not turn unknown into zero or invent replacement data. |

Not every omitted field has its own removable charge. Omitting `metrics.paid` from Domain Overview decoding does not reduce the endpoint's bundled price. Turning off a paid clickstream flag may. Slicing Google Ads ideas **after** a flat-fee response saves no DataForSEO money. A locally filtered view of already purchased rows can be free; a new server-side sort/filter across a wider dataset can be a new paid query.

## What the reviewed code retains and drops

Paths below are relative to the repository root. Line links can move; use the named struct/function as well. These are source-reviewed facts, not a production UI test.

### 1. Domain Overview, Keywords and Pages

Evidence:

- `backend/internal/domain/provider.go`: `organicMetrics`, `providerMetrics`, `rankedKeywordItem`, `relevantPageItem`, `overview`, `rankedKeywords`, `relevantPages`.
- `backend/internal/domain/service.go`: `OverviewResult`, `PositionBands`, `KeywordRow`, `PageResult`, `Overview`, `loadOverviewSnapshot`, `saveOverviewSnapshot`.
- `frontend/src/client/features/domain/components/OverviewBreakdown.tsx`, `DomainKeywordsTable.tsx`, `DomainPagesTable.tsx`.
- `frontend/src/client/features/domain/utils.ts`: keyword/page export helpers.

Retained/mapped: organic ETV, organic keyword count, estimated traffic value, position bands 1 / 2–3 / 4–10 / 11–20, new/up/down/lost counts. Keyword rows retain keyword, rank, URL, relative URL, traffic, volume, CPC, difficulty, intent, result type and selected rank-change values. Page rows expose page URL, organic traffic and keyword count. Frontend tables/breakdowns consume the mapped fields; this is an improvement already on main, not new savings from this PR.

Domain overview maps and persists a normalized snapshot, not the complete provider JSON. The normal Redis TTL is 12 hours; Postgres overview fallback can serve up to 30 days with the existing `fetchedAt`. A cache TTL by itself does not prove the displayed data is 12 hours fresh.

Confirmed unrepresented response data: `providerMetrics` only has `organic`; `metrics.paid` is not decoded. `organicMetrics` stops the position bands at 11–20; documented `pos_21_30` through `pos_91_100` are not mapped. Relevant Pages narrows its page output to traffic/count even though the endpoint has more nested metrics. Unmapped provider fields are not recoverable from the normalized snapshot.

Business assessment: paid-search metrics might be outside an organic view, which is a valid **scope choice** if named. They should not be marketed as included. Position bands below 20 could support a useful "outside top 20" distribution without another provider call. Review UI need and correct total arithmetic before adding a card. Do not purchase a second lookup just to obtain fields already included in a required response.

Additional scope risk: overview source is hostname/subdomains regardless of the selected display scope (documented in the service). Never label a domain-level count as an exact subfolder count.

### 2. Keyword Research and Keywords agent

Evidence:

- `backend/internal/keywords/research_provider.go`: `wireLabsItem`, `wireInfo`, `LabsKeywords`, `AdsIdeas`, `AdsVolume`, `SerpLive`.
- `backend/internal/keywords/research.go`: `ResearchRow`, `ResearchResult`, cache/store/persist path.
- `backend/internal/sam/keywords_mode.go`: `keywordsResultLimit`, `maxPresentedKeywords`, `keywordsModePrompt` and the `present_keywords` tool.
- `frontend/src/client/features/keywords/agent/keywordReport.ts`: `metricsSchema`, `reportSchema`, `reportTotals`.

Mapped research data is already richer than the final agent report: volume, CPC, difficulty, monthly history, bids, competition, intent, backlink averages, language, word count, trend changes, source update timestamp, SERP feature types and core keyword grouping are represented in the research models. Research cache stores normalized results; the agent report's metrics schema carries only selected metrics (volume, difficulty, CPC, intent, monthly series/latest month, yearly change, SERP features and top-page domain rank). Other fields can feed research/tools without becoming a report column; that needs a named verified consumer before claiming user access.

Durable retention is narrower again: `ResearchService.persist` (`research.go:556`) converts rows to `Metric`; `saved_store.go:469` stores volume, CPC, competition, difficulty, intent and monthly history in `go_keyword_metrics`. Bids, backlink averages, source-update timestamps and SERP feature metadata are not included in that durable metrics upsert. Their presence in the richer research cache is not proof they survive cache expiry. The separate SERP snapshot stores normalized organic results (`research_serp.go:199`), not all raw SERP features.

Upstream exclusions: `LabsKeywords` always filters `search_volume > 0`; selected sources also get a must-contain term. This is before returned-row billing, unlike local slicing. It can make discovery cheaper/more focused, but estimated zero volume is **not proof nobody searches**; niche, new and local queries can still matter. `OtherLanguage` is marked for exclusion in research. Provider-accepted language labels are fallible. Do not silently broaden these omissions into a promise of exhaustive multilingual research.

Local loss: `AdsIdeas` receives flat-fee Google Ads ideas and returns only `items[:min(len(items), limit)]`. More returned ideas are lost from this path with no provider refund. The shared keyword SERP parser maps selected fields; rich nested PAA, local, AI and other SERP feature details are not represented by `SerpItem`. `wireLabsItem.SerpInfo` retains feature types, not every documented SERP-info field.

Agent selection: the current turn can research 150 candidate rows per seed, with at most 3 seeds, while `maxPresentedKeywords` is 20. The prompt intentionally picks recommendations, removes some jobs/courses/salary/other brands unless requested, and returns an actionable plan. **This is a curated shortlist, not full research export.** Counts are configured ceilings, not proof every run buys 450 rows or discards 430. Deduplication, filters, fallbacks and tool choices change actual counts. Report rows researched/presented and where the remaining useful candidates can actually be recovered; do not promise an export that has not been built.

Business assessment: an action plan for 20 good keywords can beat a noisy 450-row dump. Preserve the supporting candidates where worthwhile and identify a no-rebuy access path. Reducing research blindly to 20 candidates may hurt selection quality. Test shortlisted relevance and user outcomes alongside cost. Exact keyword variants sharing a volume need grouping/provenance, not misleading summation.

### 3. Rank Tracking

Evidence:

- `backend/internal/ranktracking/serp.go`: `CheckLive`, `CheckResult`, `parseSerp`.
- `backend/internal/ranktracking/serp_queued.go`: queued tasks and shared parsing.
- `backend/internal/ranktracking/store_runs.go`: `go_rank_snapshots` insert.
- `backend/internal/ranktracking/results.go`: snapshot/device result payloads.
- `frontend/src/client/features/rank-tracking/RankTrackingTableParts.tsx`: `SerpFeatureTags`, `competitorsAbove`.

Already economical: Live checks use `stop_crawl_on_match` and restrict matching to organic results. Queued support also exists. Do not sell these as entirely new changes. A target found early can mean later pages were **never purchased**, not lost after purchase. Verify actual task cost/refunds for savings measurements.

Retained: target position/URL, unique returned item types and at most **20** organic competitor summaries (`rank`, `url`, `domain`, `title`). Snapshots persist those normalized values. `parseSerp` does not preserve the full JSON; per-result descriptions, nested feature/citation details and organic competitor rows beyond the first 20 are not retained by that parser.

Shown: rank table/feature badges and a hover with at most **5** competitors above the target. Feature badge rendering uses a fixed map and hides unknown types. Therefore **"not shown in hover" and "dropped before storage" are different issues**. The first 20 may still be in the API/snapshot even when only 5 appear in a hover; rows after 20 are gone from the parsed result. No complete raw-SERP export was verified here.

Business assessment: stop-early is appropriate for a target rank check, but cannot support a claim of a complete top-100 competitor SERP. A compact hover is fine if the customer can reach the useful retained evidence elsewhere or the feature explicitly promises a shortlist. If deeper paid results are collected, justify the 20-row retention cap or propose retaining the useful returned rows. Do not buy a second SERP merely to show already received competitors. Treat the provider comment "whole SERP" as narrower than actual code behavior.

### 4. Backlinks

Evidence:

- `backend/internal/backlinks/provider.go`: `providerSummary`, `providerBacklink`, `providerDomain`, `providerPage`.
- `backend/internal/backlinks/service.go`: normalized caching, summary mapping and `mapBacklink`.
- `frontend/src/client/features/backlinks/BacklinksTableColumns.tsx`: main table, text context and flags.
- `frontend/src/client/features/backlinks/export.ts`: `buildBacklinksTabExport`.

Retained/displayed/exported: summary counts and breakdowns, source/target URLs, anchor/context, link/domain/page rank, spam, first/last seen, status flags, link count, source title, placement, target status code and country. The CSV contains **Source Page Rank**, full text-before/after and rel attributes even though the primary table focuses on a subset and truncates context visually. This is evidence of more value than the main screen alone shows. It is **not** proof of complete raw parity.

Confirmed omitted wire fields include documented `domain_from_platform_type`, `page_from_external_links`, `page_from_internal_links`, `page_from_size`, `page_from_encoding`, `page_from_language` and `page_from_status_code`; the reviewed `providerBacklink` does not decode them. They therefore do not flow into the normalized cache/CSV through this model. Some are useful for prospect quality/language or source-page diagnosis; others may be low-value technical detail. Scope the decision, do not request a separate enrichment for data already bundled.

Default backlink rows request live links (the service explicitly removes the ineffective Hide Lost toggle from the cache identity). A zero lost count in that row list is not proof the full link history has no lost links. Main already avoids rebuying identical live rows for this toggle. Full-list reuse, local views and query caches are also existing safeguards; verify the specific pathway before promising that a new sort/filter is free.

Internal trust issue: the table calls DataForSEO `domainFromRank` **DA** without naming the metric/scale clearly. That can be confused with other vendors' Domain Authority; it is not verified equivalent. The separately fetched Ahrefs DR is a different metric and an additional provider path, not a free field of the DataForSEO response.

### 5. Shared transport and billing provenance

Evidence: `backend/internal/platform/dataforseo/client.go`, `envelope.go`, `usage.go`, `pricing.go`.

The transport receives raw JSON, checks billing/task metadata, records cost and returns raw bytes to the feature parser. `Results` returns task result messages. `UsageRecorder` persists organization + path + cost in `go_dataforseo_usage`; that insert is **not a full-response archive or a field-completeness ledger**. Individual feature storage must be traced. Do not mistake `json.RawMessage` in memory for durable raw-response retention.

The shared price estimator is a ceiling/hold, not the final provider invoice. Labs generic task+row assumptions are safe only for supported matching endpoints; historical/category endpoints have documented exceptions. `serpCeiling` handles normal depth/mode/operator costs and queued AI Overview, but it is not a complete estimator for every paid SERP parameter. A newly enabled option must update the hold/quote/tests before rollout. A hold that is too high can block legitimate customer work; too low can leave Uplumo paying unrecovered spend. Never silently lower holds using uncertain documentation.

## Techniques to try: savings and what the user gives up

These are scenarios, **not deployed savings or new account pricing**. Confirm actual baseline usage, current rates and intended outcome before changing behavior. USD rates come from the dated provider guide/reference; no conversion to INR/customer credits is assumed.

| Technique | Example provider economics | User value preserved | What changes or must be disclosed | Current status |
| --- | --- | --- | --- | --- |
| Reuse an already bought response for details/analysis/export | 0 additional DataForSEO calls for derivations from retained data | More evidence for the same purchase | Requires fields to survive mapping/storage; storage/compute still cost money | Best first candidate; exact omitted fields above |
| Scheduled Google Organic Standard vs Live | 10,000 one-page tasks: $6 vs $20; $14/run lower | Same query scope if same supported parameters | Queue latency; not interactive immediate results | Rank Tracking already has queued support; measure actual mix |
| Google Ads compatible batch fill | 10,000 volume keywords: 10 full Standard tasks $0.60 vs 10,000 tiny Standard tasks $600 | Same valid keyword/location/date coverage | Similar variants may need distribution; no incompatible geo/time mixing | Current keyword code already supports arrays; 10,000 single-task baseline not observed |
| Reduce an unnecessarily small split in required Labs results | Need 1,000 rows: 1 call $0.132 vs 10×100-row calls $0.24 | All required 1,000 rows | No saving if customer only needs 100; then overfetch is waste | Proposal for an observed compatible pagination/batch use case only |
| Turn off unconsumed clickstream option | Supported Labs request without clickstream avoids documented 2× total | Base metrics remain | Clickstream-refined estimates disappear; cannot call data equivalent | Agent already refuses clickstream; other callers need audit |
| Base crawl + selected browser pass | 10,000 base pages + 200 separate rendered pages = $2.52 vs $51 all-rendered | Basic audit plus rendered evidence for selected templates | Only 200 have rendered coverage, not 10,000 | Scenario only; Uplumo renderer path needs its own audit |
| Weekly rather than daily monitoring | 28 daily checks vs 4 weekly = 85.7% fewer runs at equal per-run cost | Periodic trend monitoring | Slower detection and 24 missing daily observations | Owner decision is already pending for AI tracker; do not change silently |
| Preserve more of an already returned flat-fee Ads ideas result | Slicing fewer received rows saves $0 provider dollars; reuse may avoid a later rebuy | More candidate support without fresh provider work | Need bounded storage, relevant selection and an access path | Confirmed local truncation; suitable candidate |
| Prefer known business IDs when suitable | Extended Reviews keyword vs place_id/cid has different published surcharges | Exact intended entity/reviews | Block sizes, identifier validation and current pricing still matter | Documentation opportunity; not an audited Uplumo runtime flow |

First try **recovering useful purchased data and avoiding duplicate work**. Lowering depth, supported languages, devices, refresh frequency or model quality is a product change when it reduces what the user bought. Savings are only real if the intended outcome remains acceptable and the tradeoff is visible.

## Think like Uplumo's business owner

Optimize contribution margin per **successful customer job**, not the cheapest isolated endpoint or the largest JSON response.

```text
expected provider cost/job = sum(endpoint task fees + billable units + paid options
                                 + provider-model usage where applicable)
monthly provider saving = equivalent old monthly provider cost - new monthly provider cost
net monthly saving = provider saving - added storage/compute/support cost
contribution/job = attributable net revenue - provider cost - other variable costs
```

Use measured actual settled cost when available. Track reserved/quoted cost separately. Do not count a returned refund and the reduced final task cost twice. A cache hit is 0 new provider purchases but may use older data; disclose the source timestamp. Data already paid for is not automatically free to store, share across tenants or process through another AI provider.

Track these together, with baseline and follow-up period:

- Paid calls and settled USD per completed action, plus failures/unknown outcomes.
- Cache/in-flight reuse rate, duplicate paid purchase rate, paid rows received vs useful/retained rows.
- Successful jobs and customer-visible scope/freshness/completeness; never count a cheaper incomplete result as an equivalent success.
- Useful evidence accessible from UI/detail/export/analysis and any hidden caps.
- Quoted/held/charged customer credits and refunds, with current plan/catalog rules.
- Added storage/egress/LLM analysis cost, customer wait time and support/refund impact.

Do not assume lower provider spend should automatically change subscription prices or credit prices. Report whether the benefit goes to lower user credit spend, more included useful evidence, higher reliability or improved company margin; the owner decides pricing. Conversely, do not keep charging for a paid option that was disabled while presenting it as unchanged value. Quote/charge semantics must remain honest and match the purchase terms.

Global scope matters: preserve selected country, language, device, currency, date range and entity. US/English fallback must be an explicit visible default where used, not a silent worldwide assumption. Report provider costs in USD and customer currency through the actual catalog; do not hard-code an FX rate for this analysis.

## Owner findings and follow-up priority

The documentation PR records these without changing runtime behavior. Line numbers refer to the reviewed base.

| Priority | Evidence | Finding / business consequence | Recommended next step |
| --- | --- | --- | --- |
| High | `ranktracking/serp.go:139`; frontend `RankTrackingTableParts.tsx:125` | Only 20 competitor rows survive parsing; hover uses 5. This is not the whole bought SERP. | Agree competitor evidence scope; retain/access useful returned rows without rebuying, then fixture-test field/count coverage. |
| High | `keywords/research_provider.go:460`; `sam/keywords_mode.go:29`; frontend `keywordReport.ts:6` | Flat-fee ideas truncated; 20-pick agent report narrower than research dataset. | Measure received/retained/presented counts; design access to relevant candidates or reduce unnecessary purchases with outcome tests. |
| High | `platform/dataforseo/pricing.go:150`, `:178` | Generic Labs/SERP estimator is not a universal endpoint/add-on price ceiling. | Block new endpoint/paid-option adoption until estimator, quote, spend guard and realistic cost tests match supported inputs. No proven current overcharge is claimed. |
| Medium | `domain/provider.go:26`, `:41` | Only organic and top-20 position bands mapped; paid block/lower bands dropped. | Declare organic scope; evaluate below-20 evidence from same response before any new call. |
| Medium | `keywords/research_provider.go:369` | Zero-volume discovery candidates always excluded; not an exhaustive keyword list. | Explain filter/scope; preserve exceptions for authorized niche/local goals when justified. |
| Medium | `domain/service.go:26`, `:27` | US/English fallback is hard-coded. | Verify caller/UI supplies explicit targeting and labels defaults for global customers. |
| Medium | frontend `keywords/agent/keywordReport.ts:81`, `:102` | Fixed 10% CTR forecast and fixed KD labels are heuristics, not provider observations. | Owner/data-backed configurable methodology and uncertainty; no guaranteed visits or rankability claims. |
| Medium | frontend `backlinks/BacklinksTableColumns.tsx:297` | DataForSEO rank labeled DA risks confusion with another vendor's metric. | Name/scale accurately and keep third-party DR separate. |
| Medium | `backlinks/provider.go:52` | Some useful source-page quality/language fields are not decoded. | Prioritize by prospect/audit use; map bundled fields before buying enrichment. |

Do not implement all these together automatically. The owner asked for this one documentation/rules PR. Bring the next concrete feature proposal with the owner report above and the existing authorization workflow. No separate ongoing audit/status document is needed; maintain this provider business rule and the feature's four standard docs when that feature is actually changed.

## Verification and limits of this PR

Changes are agent instructions, provider reference/playbook and the narrow documentation-map allowance for the full reference. No provider/backend/frontend feature behavior was modified. Source/retention/UX claims above are static-code findings, not a live production parity test. No real customer or provider balance was spent. Broader discovered issues are reported for owner prioritization, not silently fixed or labeled resolved.

A future feature PR must include a synthetic response with relevant extras and nested items, prove which data survives, check UI/export/analysis access, and assert no extra paid call on reading retained evidence. Adding tests that merely check this document contains words would not prove any of those outcomes.
