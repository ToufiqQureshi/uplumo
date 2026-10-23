# Keywords and SERP locations

## Behavior and edge cases

- Research accepts 1–200 seeds, modes `auto`, `related`, `suggestions`, `ideas`, result limits 150/300/500, optional clickstream and grouping. Auto blends suggestions and ideas, alternating rows and deduplicating keywords; it falls back when fewer than five non-seed rows appear.
- Labs countries use Labs keyword data. Countries marked Google Ads-only use Google Ads, which lacks difficulty and search intent. Local research validates the canonical city/region before spending; local volume replaces national volume rather than silently falling back. National metrics remain available for saved keywords. Labs rows also carry the top-of-page bid range, the average referring domains of page-one results and the 3-month volume change (`ResearchRow`, shown as the Top bid, Ref. domains and 3-mo trend columns); they come in the same call, so a field the provider already returned is shown, not re-bought. Google Ads-only rows leave them empty.
- When the seed names a place from the market's location registry (a city, state, district, region or county, such as "lonavala" in "best resort in lonavala"), the Labs `ideas` and `related` tasks send a DataForSEO `filters` clause (`keyword ilike %place%`), so every billed row mentions that place instead of drifting to the whole category. The result carries `placeFilter`; a customer who wants wider ideas searches without the place name. Suggestions are phrase matches and are never filtered. Google Ads markets are never filtered and skip the registry lookup.
- Every Labs research row carries `serpFeatures`, the result types Google shows for the keyword (`ai_overview`, `local_pack`, `hotels_pack`, `people_also_ask`, ...), from `include_serp_info`, which DataForSEO returns in the same task at no extra cost.
- Research and SERP cache keys include organization, project, market, area, seed, mode, depth, and options that change results. A deeper 100-result SERP snapshot answers a 20-result request; an empty deeper response must not evict a useful shallow snapshot.
- Concurrent equivalent research and same-depth SERP cache misses share in-flight provider work
  inside one Go process, keyed by the same organization/project/market/input
  cache key. Different organizations cannot join a flight. Separate server
  processes are not coalesced. A caller that cancels before registration starts
  no purchase; an existing bounded flight can finish for its other waiters.
- Saving normalizes keywords by trim and lowercase, deduplicates, stores supplied metrics, and returns the exact saved row IDs. Tags can append or replace only on those rows; replace requires nonempty tags. Metric rows and tag assignments must remain project-scoped.
- Saved keyword listing supports search, include/exclude terms, volume/CPC/difficulty ranges, tag IDs/names, seven sort fields, pagination at 50/100/250, and an export with the same filters. Rows preserve all nullable metrics, monthly searches, fetched timestamp and tag fields. Malformed monthly-search JSON becomes an empty array.
- Tag assignment/removal supports batches; a tag rename/color edit is project-scoped, and deleting an in-use tag returns the assignment count instead of deleting it. Saved keyword removal accepts at most 2,000 IDs and affects only that project.
- SERP location search accepts a two-letter ISO country and a 1–100 character query, returning at most 50 canonical city/county/municipality/DMA/region rows. Postal codes, states, airports and universities are excluded. Exact first-segment matches rank above prefixes and DMA rows; US/Canada/Australia region abbreviations expand after the place token. A prewarm request fills the same country cache.
- DataForSEO's full per-country registry is large; cache successful, filtered country data for 30 days, coalesce concurrent cold fills, and bound provider calls. A failed or empty provider response must not poison the cache. Invalid input must never reach a billed provider task.

## Go API (all `POST /api/v1/projects/{projectId}/keywords/...`, contract in `api/keywords.yaml`)

`research`, `serp`, `saved/refresh` (billed, paid-plan gated when billing is wired), `saved/save|list|export|remove`, `saved/tags/assign|update|delete`.

- Tables: `go_saved_keywords`, `go_saved_keyword_tags`,
  `go_saved_keyword_tag_assignments`, `go_keyword_metrics` (migration 00011)
  and `go_keyword_serp_snapshots` (migration 00037).
- Env: needs `DATAFORSEO_*` for research/serp; Redis for caches (research v7
  24h, SERP 12h). On Redis SERP cache misses, Postgres can serve a non-empty
  snapshot up to 7 days old without another provider charge.
- Never break: validate before any provider call (failed tasks are billed); tag delete locks the tag so a concurrent assign cannot orphan rows; metrics persist under the requested language even when Labs served another.
- Deliberate differences from legacy: NULL metrics sort last in saved lists; research language is normalized to the one Labs serves.

Suggestions return exact seed metrics separately in `seed_keyword_data`.
Preserve that row before suggestions and deduplicate during row normalization;
never buy an overview just to retrieve data already in the response. A measured
zero in clickstream data stays zero rather than reverting to national volume.

## Markets

Market tables and resolution are in `platform/market`. Labs covers 94 countries; the rest are served by the Google Ads endpoints, which have no keyword difficulty or search intent.


The Go MCP `get_serp_results` tool uses the shared keyword SERP provider for
market and canonical local-location validation. It returns bounded live result
rows per query and captures individual query failures without discarding
successful results. 


The Go MCP `get_keyword_metrics` tool fetches known terms from Labs or Google Ads by market, supports optional clickstream refinement and monthly trends, and returns nullable metrics sorted by the requested field. 


The Go MCP `research_keywords` tool calls the shared Go research service per seed, honors local-area markets and the requested research options, and returns a separate result for each seed. 


Auto research buys two legs in parallel (suggestions and ideas). If one leg
fails but the other returned rows, `oneLegResult` answers with that leg
(`source` is `suggestions` or `ideas`) and caches it for ten minutes
(`partialResearchCacheTTL`) so a retry costs nothing; a credit refusal, a
replay, a customer who left, or a working leg with no rows is still an error.

Cancellation during cache/project preparation is rechecked before detaching
paid work. Active callers retain cache-outage fallback; existing shared work
can finish for other waiters. See the cancellation decision and regressions.


The three MCP keyword data tools (`mcp/serp_results.go`, `mcp/keyword_metrics.go`,
`mcp/research_keywords.go`) do not yet have handler-level contract tests;
`mcp/registry_test.go` covers plan gating. SERP status 40102 is a valid,
billable empty result; 40501 is an invalid-field error and is refunded. See
`../platform/dataforseo/decision.md`.

Successful empty keyword research is reused for ten minutes (`partialResearchCacheTTL`) instead of buying the same tasks again on every revisit. Null/malformed results, invalid source identities and provider errors remain cache misses/errors; non-empty research keeps its 24-hour TTL.
