# Domain overview and research

Shows organic estimates, keyword/page tables and filtered domain research.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `DomainOverviewPage.tsx`
- `domainApi.ts`
- `domainRouteState.ts`
- `backend/internal/domain/handler.go`
- `backend/internal/domain/provider.go`

The suggestions wrapper adds the Go explicit-action marker. Its only UI caller
is Rank Tracking's optional Find existing keywords action; request fields and
legacy-client behavior are covered by `domainApi.test.ts` and Go handler tests.

Successful overview/keyword/page snapshots remain fresh and retained for the
backend's 12-hour cache lifetime. Remounting a previously failed paid query
does not retry automatically; the user can use the existing Retry action.
An overview without data stays stale, preserving the backend's retry policy.
