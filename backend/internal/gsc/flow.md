# Search Console: request path and cost

Every call below is a **free** Search Console read. No provider that charges
money is called anywhere in this package, so nothing here is spend-guarded.

| Screen | Route | Search Console reads |
| --- | --- | --- |
| Search Growth, any screen | `report` | up to 8 in parallel (daily, previous daily, pages and queries for both periods, query+page, countries); 4 when the previous period is outside Google's 16 months |
| Overview, Fix, Grow | `insights` | up to 6 in parallel: query+page (10,000 rows), pages and queries for the current and previous period (5,000 each), pages for the period before that (5,000) |
| Explore tables | `table` | 1 per page turn |
| Explore export | `export` | 1 |

## Insights flow

screen -> `POST .../insights` (filters + brand names) -> `Service.GetInsights`
-> `fetchSearchAnalyticsRows` (parallel reads) -> pure functions in
`insights_analysis.go` -> one JSON answer.

- The browser keeps an answer for 5 minutes per project, filters and brand
  names (`staleTime`), so switching between Overview, Fix and Grow reads
  Google once. The Indexing, Explore and Conversions screens do not request it.
- If any read fails the whole answer fails; there is no partial result.
- Nothing retries automatically.

## Paid calls per user action

0. A test (`TestGetInsightsFindsWhatIsWorthDoing`) asserts the read count.
