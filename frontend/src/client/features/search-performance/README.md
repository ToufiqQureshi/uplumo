# Search Console Page

The Search Console page shows keywords, pages, countries, and devices from Google Search Console.

## Code Map

| File                           | Purpose                                                                      |
| ------------------------------ | ---------------------------------------------------------------------------- |
| `SearchPerformancePage.tsx`    | Main page component; renders the UI layout and tabs                          |
| `SearchPerformanceSelects.tsx` | Filter bar with date range presets, search type, device, and country filters |
| `SearchPerformanceTrend.tsx`   | Daily clicks and impressions trend chart                                     |
| `SearchPerformanceTotals.tsx`  | KPI cards showing totals and % change vs previous period                     |
| `searchInsights.ts`            | Computes 4 key insights from the report data                                 |
| `searchPerformanceApi.ts`      | API client for fetching report and table data                                |
| `searchPerformanceQueries.ts`  | TanStack Query hooks for data fetching                                       |
| `SearchPerformanceColumns.tsx` | Table columns and formatting helpers                                         |
| `SearchPerformanceParts.tsx`   | Export helpers for tables                                                    |

## Tabs

| Tab       | Dimension | Data Source                                      |
| --------- | --------- | ------------------------------------------------ |
| Queries   | query     | Paginated endpoint `/table` with dimension=query |
| Pages     | page      | Paginated endpoint `/table` with dimension=page  |
| Countries | country   | Backend report endpoint; `report.countries`      |
| Devices   | device    | Backend report endpoint; `report.devices`        |

## Backend Data Flow

The page fetches one report from `/api/v1/projects/{id}/gsc/search-performance/report`:

- Fetches up to 9 Search Console queries in parallel (all free)
- Returns `connected: true` with totals, trends, and all dimension breakdowns
- Returns `connected: false` if no Search Console account is linked

No paid provider calls. No spend guard.

## User Flow

1. Connect Google Search Console account (via integrations page)
2. See real-time metrics: clicks, impressions, CTR, position
3. Filter by search type, device, country, or custom date range
4. Click rows in Keywords/Pages/Countries/Devices tabs to drill into filters
5. Export Keywords or Pages as CSV/Google Sheets
