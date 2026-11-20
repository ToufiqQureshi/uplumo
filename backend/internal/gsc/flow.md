# Search Console: request path and cost

Every call below is a **free** Search Console read. No provider that charges
money is called anywhere in this package, so nothing here is spend-guarded.

| Screen | Route | Search Console reads |
| --- | --- | --- |
| Search Console Page | `report` | up to 9 in parallel (daily, previous daily, pages and queries for both periods, query+page, countries, devices); 5 when the previous period is outside Google's 16 months |
| Tables | `table` | 1 per page turn |
| Export | `export` | 1 |

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
