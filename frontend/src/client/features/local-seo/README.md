# Local SEO research

Looks up local businesses/profiles and runs a bounded local search rank grid.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `LocalSEOPage.tsx`
- `localSeoApi.ts`
- `backend/internal/localseo/handler.go`
- `backend/internal/keywords/local_serp.go`

A grid that stopped before every point was searched (credits ran out) shows
`summary.stoppedReason` under the grid summary; the points already searched
stay visible.
