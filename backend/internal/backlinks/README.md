# Backlinks

## Behavior and edge cases

| Behavior | Edge case to keep |
|---|---|
| Overview fetches the DataForSEO summary and one year of monthly history for domain scopes. | Exact page scope omits history. History dates cover UTC yesterday back one calendar year; a credit refusal on either task fails the request, but if only the history task fails after the summary was billed, the summary is returned without a trend and cached for ten minutes. |
| A subfolder cannot be targeted directly by summary/history. | Two sequential backlinks list tasks count all links and one-per-domain links with a `url_to` prefix filter; rank, spam, broken/new/lost values and trends stay null/empty. |
| Target scopes include exact URL, subfolder, domain and subdomains; persisted `page` is an alias for exact URL. | Normalize host through IDNA, remove leading `www.`, reject IPs, invalid suffixes, credentials and exact URLs with queries/fragments. A subfolder needs a non-root path. |
| Backlinks rows support all-links or strongest-per-domain mode, sort, pagination, spam hiding and include/exclude/numeric/link/lost/broken/domain filters. | Strict sort mapping, `50/100/200` page sizes, a maximum of eight provider filter conditions including scope and spam filters; include terms OR, excludes AND, escaped LIKE metacharacters. |
| Referring domains have their own aggregate filters and keep spam rows visible in the UI. | No spam clause is added. Subfolder scope returns the same validation message because this endpoint has no target-page field. |
| Top pages support pagination, sort and include/exclude/count/rank filters. | Subfolder scope is expressed as URL-prefix filters; the combined expression may not exceed eight conditions. |
| Summary, tabs and provider rows preserve nullable values and map provider spelling variants. | Keep nulls; map `new_reffering_domains` and `lost_reffering_domains`, `attributes`, legacy spam-score field, `lost_date` and `last_seen` (the provider has no `last_visited`). Every paid field the screen uses is mapped: the summary's link profile (first seen, referring IPs/subnets, nofollow domains, TLD, platform, placement and country counts, blank key preserved as "unknown") and each row's text around the link, source page title, placement, target status code, source country and `is_new`. Empty row arrays serialize as `[]`. |
| Data is shared across callers and cached for 24 hours (key prefix `backlinks:v2`), including target scope, path, organization and all filter parameters. Tables buy fixed blocks of 1,000 rows (the provider maximum) (`blockSize`) and slice pages out of them, so changing the page size or paging inside a block costs nothing. A profile of at most 1,000 links or domains is bought once as a whole list and every sort, filter and toggle is answered locally (`whole_list.go`); the local filters must match the provider's (range filters never match a missing value, include is any-term, exclude is no-term). All tasks send `rank_scale: one_hundred` so every rank is 0-100. | Redis is only an optimization; cache read/write failures log and fall through. Org ID is part of every key. Provider calls are never retried. |
| Billing classification distinguishes an account balance problem from ordinary errors. | Plan check runs before cache access or paid provider calls; free plan receives 402 when billing is configured; without billing, endpoints are open. Provider billing issue returns 503. |

## Flow

`requireSession -> requireProjectAccess -> paid-plan check -> handler validation -> service/cache -> platform/dataforseo`

Routes are `POST /api/v1/projects/{projectId}/backlinks/{overview,backlinks,referring-domains,top-pages}`. Table and row data remain provider-backed; the complete overview is also persisted in `go_backlinks_overviews` for up to 30 days so an expired Redis cache does not re-buy paid provider tasks. The table is organization-scoped, server-only and cascades with organization deletion. Project authorization supplies the organization ID used for plan checks, DataForSEO usage recording and cache isolation.
Response fields and nullability match the frontend Zod schemas in
`frontend/src/client/features/backlinks/`.

## Rules that must not break

- Never call DataForSEO before validating target, scope, filters, sort and pagination. Provider tasks are billed even on failure.
- `Client.Do(..., false)` is mandatory: a replay can charge the organization twice.
- Cache entries are organization-scoped; cache failures must not turn a valid provider call into an error.
- Concurrent identical overview and table-block cache misses on one server share in-flight work;
  the organization/scope/target key prevents cross-tenant sharing. Different
  server instances may still race. Table identities include the kind, full
  query, sort, mode and block, so different views and organizations cannot
  accidentally share results. A speculative whole-list purchase also fills the
  normal unfiltered first-block cache: an undercounted overview must not make
  the same block get bought again on every lookup.
- DataForSEO summary/history are whole-target endpoints. Subfolder counts must stay filtered and cannot claim unavailable rank/trend data.
- UI response fields and nullability match the TypeScript schemas. New API contracts are validated with Zod.

## Configuration, limits and tests

`DATAFORSEO_API_KEY` configures the shared provider transport; no new environment variable is required. Redis caches complete successful responses for 24 hours. A summary-only response after a history failure stays in Redis for ten minutes and never replaces a complete durable snapshot. Request bodies are limited to 32 KiB, filter strings to 500 bytes and provider filter expressions to eight conditions. Provider outbound timeouts and usage recording are enforced by `platform/dataforseo`.

Tests cover target normalization, filter translation and budget, handler validation/plan checks, result mapping, provider errors, cache isolation/outages and cancellation. No database tenant-isolation test applies because this package has no repository; project authorization and organization-attributed provider usage are covered at the HTTP boundary.
`service_test.go` also asserts that eight concurrent equivalent overviews issue
only the required summary and history provider calls.

The overview is two sequential provider tasks, summary then history. A credit
refusal, provider billing problem or departed customer stops after the first;
any other history failure keeps the already-billed summary (empty trend,
`partialCacheTTL` of ten minutes) instead of discarding it.

Cancellation during cache/project preparation is rechecked before detaching
paid work. Active callers retain cache-outage fallback; existing shared work
can finish for other waiters. See the cancellation decision and regressions.
