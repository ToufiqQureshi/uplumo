# Search Console (gsc)

Reads a project's Google Search Console data with the owner's own Google
grant, and turns it into the Search Growth screens. Read-only: the grant is
`webmasters.readonly`, so Uplumo can never change anything in Search Console.

## Code map

| Part | File |
| --- | --- |
| Search Console client (sites, search analytics, URL inspection) | `client.go`, `inspection.go` |
| Property connection per project | `connections.go`, `connection_handler.go`, `connection_repository.go` |
| Report, tables, export | `performance.go`, `performance_handler.go` |
| What lost clicks and why | `page_changes.go` |
| Free analysis: segments, competing pages, CTR gaps, decay, top actions | `insights.go` (reads), `insights_analysis.go` (rules) |
| Submitted sitemaps | `sitemaps.go` |
| Growth rows for the GA4 join | `growth_rows.go` |

## Routes (all `POST`, project access required)

| Route | What |
| --- | --- |
| `/api/v1/projects/{id}/gsc/search-performance/report` | Totals, daily trend, striking distance, countries, top losses |
| `.../table` | One page of queries or pages (25, 50 or 100 rows) |
| `.../export` | Up to 25,000 query or page rows |
| `.../insights` | The free analysis (below) |
| `.../sitemaps` | Submitted sitemaps, up to 200 |

Filters (page, query, device, country, search type, dates) are shared by
every route. A page or query filter can be `contains`, `notContains`,
`equals`, `includingRegex` or `excludingRegex`; a regex that does not compile
is refused before Google is called.

## Tables

`gsc_connections` (the chosen property per project; no `go_` prefix, an older table). No analysis result is
stored: it is computed from Google's data on every read.
