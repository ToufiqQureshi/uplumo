# Domain overview and research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/domain/...
- **Source:** `handler.go`, `provider.go`, `repository.go`, `filters.go`

## Request or job flow

Project request resolves the default market; handler rejects invalid scopes/filters and checks access before provider work; provider code calls DataForSEO and response filters are applied to normalized results.

## Dependencies and cost boundary

DataForSEO paid endpoints, project market and plan checks.

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
