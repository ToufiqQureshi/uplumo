# Search Console performance

Turns first-party Search Console data into a review workflow: compare periods, find pages with queries ranking 5-20, start a focused content draft, and inspect Google's stored index status.

Read [flow.md](flow.md) for the user workflow and API limits. Search Console opportunities are evidence for review, not a ranking promise or proof that content is stale.

## Code map

- `SearchPerformancePage.tsx`
- `searchPerformanceQueries.ts` and `searchPerformanceApi.ts`
- `SearchPerformanceTrend.tsx` and `SearchPerformanceIndexing.tsx`
- `ContentRefreshQueuePanel.tsx` and `contentRefreshQueue.ts`
- `backend/internal/gsc/performance.go`

## Search growth

`SearchGrowthPanel.tsx`, `growthApi.ts` and `GrowthEvidencePanel.tsx` expose the
combined Google-organic GSC/GA4 analysis above the detailed GSC report. Analysis
is explicit and independent of the detailed table's filters: 14/28-day windows,
optional brand terms, top three review candidates, evidence and one next action.
Advanced evidence, measurement setup, device rows, saved actions and stored AI
citations are drill-downs. No paid provider runs on mount, focus or refresh.

AI improvement briefs open an editable project-scoped SAM draft; nothing is
sent until Send. `growthBrief.ts` asks for measured evidence, hypotheses, a
page-specific brief, title/meta options, useful direct answers and observed
internal links. AEO readiness is distinct from observed tracked citations. The
existing editor's selection-based writer remains the review/apply surface.
