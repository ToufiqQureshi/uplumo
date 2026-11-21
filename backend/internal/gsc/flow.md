# Search Console: request path and cost

Every call below is a **free** Search Console read. No provider that charges
money is called anywhere in this package, so nothing here is spend-guarded.

| Screen | Route | Search Console reads |
| --- | --- | --- |
| Search Console Page | `report` | up to 9 in parallel (daily, previous daily, pages and queries for both periods, query+page, countries, devices); 5 when the previous period is outside Google's 16 months |
| Tables | `table` | 1 per page turn |
| Export | `export` | 1 |

## Report flow

screen -> `POST .../report` (filters) -> `Service.GetPerformance` ->
`fetchSearchAnalyticsRows` (parallel reads) -> one JSON answer with totals,
daily rows, losses and gains, quick-win searches, countries and devices.

- The country read drops the country filter and the device read drops the
  device filter, so those tables always list every country or device.
- If any read fails the whole answer fails; there is no partial result.
- Nothing retries automatically.
- The `insights` route (`Service.GetInsights`) has no screen since the
  page was simplified; it is kept until it is removed on purpose.

## Paid calls per user action

0. Every read above is a free Search Console read.
`TestGetPerformanceIncludesDevicesBuildRequestWithoutDeviceFilter` asserts
the report's read count.