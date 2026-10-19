# Search Console performance

Turns first-party Search Console data into a review workflow: compare periods, find pages with queries ranking 5-20, start a focused content draft, and inspect Google's stored index status.

Read [flow.md](flow.md) for the user workflow and API limits. Search Console opportunities are evidence for review, not a ranking promise or proof that content is stale.

## Code map

- `SearchPerformancePage.tsx`
- `searchPerformanceQueries.ts` and `searchPerformanceApi.ts`
- `SearchPerformanceTrend.tsx` and `SearchPerformanceIndexing.tsx`
- `ContentRefreshQueuePanel.tsx` and `contentRefreshQueue.ts`
- `backend/internal/gsc/performance.go`
