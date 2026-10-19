# Local SEO endpoint layer

Validates and orchestrates local business lookup/profile and bounded Google Maps rank grids.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for cost, data and security invariants.

## Source map

- `handler.go`

## API/trigger

POST /api/v1/projects/{projectId}/local-seo/{businesses|profile|rank-grid}

Rank grid: points run three at a time. If credits run out part-way the grid
answers 200 with the points already paid for, `failedPoints` counting the rest and
`summary.stoppedReason` set; with nothing searched it answers 402 or the
guard's own error as before.
