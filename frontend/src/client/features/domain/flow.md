# Domain overview and research: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. User submits a domain, market and scope in `DomainOverviewPage`.
2. Query hooks load overview, ranked keywords and pages through `domainApi.ts` and keep filters in route state.
3. Go `domain` resolves project market, validates scope and filters, gates paid calls, then fetches and shapes provider data.
4. The client renders domain-level estimates separately from exact URL/subfolder views.

## Implementation map

- `DomainOverviewPage.tsx`
- `domainApi.ts`
- `domainRouteState.ts`
- `backend/internal/domain/handler.go`
- `backend/internal/domain/provider.go`

## API and service dependencies

Go package: `backend/internal/domain`; provider requests use the shared DataForSEO client.

## Cost, security and failure boundaries

- Domain research can incur provider charges; validate target, country and filter budget before calling the provider.
- Keep project market defaults explicit and reject unsupported country/language pairs.
- Never imply provider estimates are exact Search Console measurements.
- The paid overview, keywords, pages and rank-tracking suggestions queries
  disable automatic 5xx retries. `domain-paid-retry.test.ts` guards the
  overview path; manual refetch is an explicit user decision.
- Overview/keywords/pages retain successful snapshots for 12 hours, matching
  the backend. A tab remount reuses a fresh result and does not retry a previous
  paid error automatically. Empty overviews stay stale for a later lookup.

The suggestions wrapper adds the Go explicit-action marker. Its only UI caller
is Rank Tracking's optional Find existing keywords action; request fields and
legacy-client behavior are covered by `domainApi.test.ts` and Go handler tests.
