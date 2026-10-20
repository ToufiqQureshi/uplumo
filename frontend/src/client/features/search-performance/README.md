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

## Goal-led investigations

`GrowthSettings.tsx` owns the controlled analysis settings. The goal selector discovers event-specific metrics only after Load my GA4 goals.
Changing settings does not fetch reports. Analysis keeps all-event counts separate
from the selected goal rate and labels previous-period vs 364-day comparison.
Expanded coverage is explicit and bounded to 5,000 rows/source; optional BigQuery
setup stays an external link with its Google Cloud cost caveat.

`GrowthClickDiagnosis.tsx` shows possible click-loss patterns and returned query
comparisons, plus a separate one-URL stored-index inspection. Shared guidance is
in `indexingGuidance.ts`. `GrowthSitemaps.tsx` explains sitemap-level status without
claiming indexing. `GrowthFunnel.tsx` uses editable actual event names, ordered
users and optional device breakdown; example events must be instrumented first.
`GrowthCitationEvidence.tsx` separates saved prompt samples from Google's native
AI report (external property/help links; no automatic import).

`GrowthMonitor.tsx` reads persisted configuration on opening its panel and enables
or disables weekly in-app checks explicitly. It displays the last dated digest,
three next actions and meaningful drop alerts. It does not send email or paid AI.
`GrowthPageCard.tsx` can freeze a user-selected unchanged comparison page alongside
the saved goal/baseline; `GrowthActions.tsx` explains descriptive relative movement.

## Responsive investigation UI

`GrowthInvestigation.tsx` groups each topic into a native named disclosure, so
opening one closes its sibling without a new request or losing loaded results.
`GrowthDeviceEvidence.tsx` owns the device drill-down; `GrowthMobileResults.tsx`
shows fully labeled journey/device/query cards below 640px, with desktop tables
above that breakpoint. Funnel steps use separate editable inputs with explicit
add/remove controls. Secondary interpretation and sitemap submission details are
collapsed; error/limited-data signals remain visible. Long action labels and
URLs wrap, and borders use the existing theme token instead of current text color.

## IndexNow

The IndexNow card (`indexnow/IndexNowCard.tsx`) sits at the bottom of the
page, shown whether or not Search Console is connected: it works on any
website and needs no Google connection. Rules live in
`backend/internal/indexnow/README.md`.
