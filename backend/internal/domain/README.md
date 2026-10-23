# Domain overview

Domain Overview page: organic traffic and keyword count for a domain, plus two
tabs (ranking keywords, top pages) and the keyword suggestions used by rank
tracking. Ported from the TypeScript `DomainService`.

## Endpoints

All are `POST /api/v1/projects/{projectId}/domain/<name>` and need a session
and membership of the project's organization.

| Name                  | Returns                                                        |
| --------------------- | -------------------------------------------------------------- |
| `overview`            | traffic, keywords, `hasData`, echoed `scope` and label         |
| `keyword-suggestions` | top 100 ranking keywords by traffic (no paging)                |
| `keywords`            | one page of ranking keywords, with ranking `url`/`relativeUrl` |
| `pages`               | one page of ranking pages                                      |

Errors: `400` bad input, `402` free plan (only when billing is configured),
`503` no `DATAFORSEO_API_KEY` or a provider balance problem, `502` any other
provider failure.

## Flow

`session -> project access -> plan gate -> decode + validate -> service -> Redis -> Postgres snapshot -> DataForSEO Labs`

The plan gate runs before the body is read, so a free plan never costs a task.
Validation runs before the provider: failed provider tasks are billed.

## Rules that must never break

- **Validate before spending.** Host, scope, paging, sort and filters are
  checked in the handler or service; an invalid request reaches the provider
  zero times (tests assert the call count).
- **Filter budget = 8.** The scope consumes slots (keywords: exact URL 4,
  subfolder 4, domain 1; pages: 4/4/2) and the search box costs 2. The slot
  numbers must equal `RESEARCH_SCOPE_FILTER_SLOTS` in
  `frontend/src/shared/researchScope.ts`, which the UI uses to shrink its own budget.
  Counting is done by `countConditions`, not hard-coded.
- **Cache keys carry organization and project**, plus every input that changes
  the answer (scope, path, market, page, size, sort, filters, search).
- Successful empty overviews and suggestions are reused in Redis for five minutes, so revisits do not buy the same empty task repeatedly. Empty overviews are not persisted for 30 days; malformed cached overviews without a fetched timestamp are ignored. PostgreSQL is consulted only after both Redis checks miss.
- Redis down is soft: log, try a fresh persisted snapshot, then call the provider.
- Concurrent equivalent overview/keyword-page/page-page misses share one
  provider task per server process. Organization, project, market and every
  result-changing request input remain in the identity; overview scope labels
  are reapplied for each caller after sharing hostname-level metrics.
- Shared report purchases finish for other waiters if the first caller leaves,
  with a 90-second bound; already-cancelled preparation buys nothing.

## Differences from the TypeScript version

- **The server resolves the market.** A project-scoped SQL lookup reads
  `projects.location_code` and `language_code` for the authorized organization.
  A location override selects that country's default language unless the request
  also supplies a language. An unsupported project default falls back to US/English.
  Explicit unserved Labs locations and language pairs are rejected before billing.
- Request schemas moved from Zod on the server to Go validation; the response
  contract is unchanged and checked by Zod in `domainApi.ts`.
- The Playwright fixtures (`VITE_E2E_DOMAIN_FIXTURES`) now short-circuit in
  `domainApi.ts` instead of the server function.

## Tables, env

Reads the legacy `projects` table, filtered by both project and organization.
Persists non-empty paid overviews in `go_domain_overviews` for 30 days so a
Redis cache miss can reuse already-paid data. The table is server-only, tenant
scoped and cascades with project deletion. Needs `DATAFORSEO_API_KEY`; Redis is optional.

## Edge cases covered by tests

`www.` stripping, IPs and fake TLDs rejected, path kept for subfolder/exact URL,
LIKE wildcards escaped in filter text, UTF-16 length limits (emoji), rows with
no keyword or no address dropped, relative URL derived from the ranking URL,
`hasMore` with and without a provider total.


The Go MCP `get_ranked_keywords` tool uses the same DataForSEO Labs ranked-keyword endpoint with host/path scope filters, bounded filter conditions, market validation, sort modes, result-type selection, and pagination.


The Go domain provider also exposes a bounded Labs SERP competitor lookup for the MCP registry, with location, language, item-type, and pagination validation. See the MCP package tests for tool-contract coverage.

The optional suggestions route requires `userInitiated: true` from an explicit
Rank Tracking action. `suggestions.go` shares identical organization/project/
target/scope/market work for up to 90 seconds, rechecks the 12-hour nonempty
cache and buys one task on a miss. See `decision.md` for failure/cancellation
and one-replica constraints.

Cancellation during cache/project preparation is rechecked before detaching
paid work. Active callers retain cache-outage fallback; existing shared work
can finish for other waiters. See the cancellation decision and regressions.
