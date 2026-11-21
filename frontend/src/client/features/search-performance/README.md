# Google Search page

One screen of the customer's own Google Search Console data: totals and
their change, a daily trend, a short "What changed" list, and Keywords,
Pages, Countries and Devices tables, plus an IndexNow tab
(`features/indexnow/IndexNowCard.tsx`, shown even without Search Console). Route: `/p/$projectId/search-performance`.

## Code map

| File                               | What it does                                                                 |
| ---------------------------------- | ---------------------------------------------------------------------------- |
| `SearchPerformancePage.tsx`        | The page: connection card, filter bar, totals, trend, insights, tabs         |
| `SearchPerformanceSelects.tsx`     | Date presets (rolling and calendar month), custom dates, type/device/country |
| `SearchPerformanceTextFilters.tsx` | Page and keyword text filters, including regular expressions                 |
| `SearchPerformanceTotals.tsx`      | Clicks, impressions, CTR, position with change against the previous period   |
| `SearchPerformanceTrend.tsx`       | Daily clicks and impressions chart                                           |
| `searchInsights.ts`                | Up to four "What changed" lines built from the loaded report                 |
| `SearchRowsTable.tsx`              | Paginated Keywords or Pages table with export                                |
| `SearchBreakdownTable.tsx`         | Countries or Devices table with share of clicks                              |
| `SearchPerformanceParts.tsx`       | Keywords/Pages table body and export helpers                                 |
| `SearchPerformanceColumns.tsx`     | Report types, table columns, number formatting                               |
| `searchPerformanceApi.ts`          | `report`, `table` and `export` calls with Zod response schemas               |
| `searchPerformanceQueries.ts`      | TanStack Query options for the table                                         |
| `../../lib/countryNames.ts`        | Search Console country code (`ind`) to a name in the viewer's language       |

## Routes

All under `/api/v1/projects/{id}/gsc/search-performance/`, all free Search
Console reads (see `backend/internal/gsc/flow.md`):

| Route    | Used for                                                 |
| -------- | -------------------------------------------------------- |
| `report` | Totals, trend, insights, Countries and Devices (9 reads) |
| `table`  | One page of Keywords or Pages (1 read)                   |
| `export` | Full Keywords or Pages download (1 read)                 |

No tables of our own; nothing is stored.
