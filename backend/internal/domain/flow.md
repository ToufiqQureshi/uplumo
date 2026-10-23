# Domain overview and research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/domain/...
- **Source:** `handler.go`, `provider.go`, `repository.go`, `filters.go`

## Request or job flow

Project request resolves the default market; handler rejects invalid scopes/filters and checks access before provider work; provider code calls DataForSEO and response filters are applied to normalized results.

## Dependencies and cost boundary

DataForSEO paid endpoints, project market and plan checks.

A cold overview buys one Labs overview task. Each distinct cold keyword-page
or page-page request buys one task; sorting, filtering, paging and changing
page size are distinct provider views. Equivalent concurrent requests to any
of those three paths share one task within a server process. Redis cache hits
buy zero tasks. On a Redis miss, a non-empty PostgreSQL overview snapshot up to
30 days old also buys zero tasks; hot Redis hits do not query Postgres. An initial
overview plus keyword tab can legitimately buy two
different tasks; this is separate from duplicate identical purchases.

`lookup.go` rechecks cancellation after the caller's cache read, then joins a
bounded 90-second shared lookup and checks the cache again before purchasing.
Individual callers can stop waiting without cancelling other waiters' work.
Errors and empty overviews retain the existing retry behavior; only non-empty
overviews are saved in Postgres and served from that snapshot.

## Failure, security and data invariants

- Validate target, market and filter budget before spend; failed provider tasks may still be billed.
- Keep scope semantics explicit: domain, subdomain, subfolder and exact URL do not support identical aggregates.
- Escape user filters for SQL/provider syntax and cap response size.

The optional suggestions route requires `userInitiated: true` from an explicit
Rank Tracking action. `suggestions.go` shares identical organization/project/
target/scope/market work for up to 90 seconds, rechecks the 12-hour nonempty
cache and buys one task on a miss. See `decision.md` for failure/cancellation
and one-replica constraints.

Preparation boundary: `KeywordSuggestions` rechecks cancellation after `readCache` and before `suggestions.DoChan`. A cancelled request returns its context error
without starting a new provider operation. Existing shared work is unaffected.

Successful empty overviews and suggestions buy one cold task, then reuse the result for five minutes. Existing non-empty TTLs and durable snapshots remain; provider errors and malformed cache entries do not become valid empty results.
