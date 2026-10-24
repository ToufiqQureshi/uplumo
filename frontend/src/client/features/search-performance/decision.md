# Search Performance and Search Growth: decisions

Read this before changing the GSC page, combined GA4 analysis or mobile
investigations. These decisions explain the intended user experience and the
constraints an implementation must preserve. Exact provider/storage bounds live
in the linked feature READMEs; do not silently change them during a UI refactor.

## Product and report scope

### Deliver a useful first step before GA4 setup

- **Decision:** show up to three GSC-only page review tasks before the optional
  business-goal analysis. Reuse `buildContentRefreshQueue`, safe HTTP URLs and
  the existing editor evidence handoff; expose exact query metrics and dates.
- **Why:** live inspection found both tested projects had real GSC rows but
  their primary analysis action stopped at missing GA4. The backend already has
  findings, baselines and weekly checks; duplicating it does not fix that first
  experience. Impression ordering selects visibility candidates, not the most
  commercially valuable pages or proven content defects.
- **Preserve:** no new report/AI/provider requests, schedule, stored action or
  site mutation on opening tasks. Query evidence remains incomplete and
  filter-scoped. Context must be preserved before navigation; storage failure
  must not open an unrelated draft. GA4 setup and verified goal instrumentation
  are still required for business-outcome analysis. Connection errors retain a
  retry, saved actions remain accessible, and no revenue estimate is invented.
- **Trace:** `SearchActionPlan.tsx`, `SearchGrowthPanel.tsx`,
  `SearchPerformancePage.tsx`; mounted regression tests and
  `tests/search-owner-actions.spec.ts`. Detailed reports start collapsed unless
  URL filters/pagination select an investigation. Expansion persists across tab
  and pagination changes without remounting table state; stored-index tools have a separate
  disclosure. Live qualified-lead reconciliation, owner interviews and
  post-change outcomes are not established by fixture-based tests.

| Decision                                                                                                                | Why                                                                                                                                | Code / related contract                                                                   |
| ----------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Start with GSC page tasks; add the business goal through the explicit GA4 analysis when connected.                      | A solo owner needs a useful decision before a large report. More metrics alone do not deliver value.                               | `SearchGrowthPanel.tsx`, `GrowthSettings.tsx`                                             |
| Keep the detailed GSC report's dates/filters in the URL, and label the growth analysis's independent date/source scope. | Shareable investigation filters must not silently reinterpret a saved GA4 finding.                                                 | `SearchPerformancePage.tsx`, `searchPerformanceQueries.ts`                                |
| Keep all-event counts, selected-goal session rates, ordered user funnels and AI evidence separately labeled.            | They have different denominators and collection scopes; combining labels can invent business outcomes.                             | `growthApi.ts`, `GrowthPageCard.tsx`, `GrowthFunnel.tsx`                                  |
| Show measured evidence, possible explanations and a next check; keep missing/limited values explicit.                   | Search averages do not establish why a person did not click, and unavailable data is not measured zero.                            | `GrowthClickDiagnosis.tsx`, backend `ga4/growth_clicks.go`                                |
| Reuse returned GSC query/page evidence for the refresh queue and brand/competing-page candidates.                       | Reusing first-party data avoids redundant calls; multiple ranking pages are not proof of harmful cannibalization or stale content. | `contentRefreshQueue.ts`, `ContentRefreshQueuePanel.tsx`                                  |
| Keep complete-period comparisons and their history-floor limitations visible.                                           | A partial previous window can look like growth even when the underlying comparison is invalid.                                     | `SearchPerformanceTotals.tsx`, `SearchPerformanceTrend.tsx`, backend `gsc/performance.go` |

## Mobile usability and investigation depth

| Decision                                                                                                                            | Why                                                                                                                                                                  | Code / verification                                                                                                  |
| ----------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| Use labeled result cards below 640px, retaining every metric from desktop tables.                                                   | The original narrow funnel table clipped the drop-off column. Users must read complete journey, device and query evidence without scrolling the whole page sideways. | `GrowthMobileResults.tsx`, `GrowthDeviceEvidence.tsx`, `GrowthClickDiagnosis.tsx`; 320px/390px Playwright assertions |
| Open one investigation topic at a time using native named disclosures. Opening a topic does not fetch data.                         | The original expanded panel exposed a wall of unrelated forms and caveats. Focused sections reduce cognitive load while keeping loaded results available.            | `GrowthInvestigation.tsx`, `GrowthEvidencePanel.tsx`                                                                 |
| Give each funnel step its own labeled input and accessible add/remove controls.                                                     | A comma-separated field truncated meaningful event names on mobile and made the sequence difficult to edit. The API still receives the same event-name array.        | `GrowthFunnel.tsx`, `tests/search-growth.spec.ts`                                                                    |
| Keep material error/limited-data signals visible; place secondary interpretation and sitemap submission details behind disclosures. | Users need the problem and next action first, with the explanation still available rather than removed.                                                              | `GrowthSitemaps.tsx`, `GrowthFunnel.tsx`                                                                             |
| Use the app's `border-border` token and wrap long action labels/URLs.                                                               | Bare Tailwind borders inherit text color, creating heavy black nested outlines. Long labels must not widen a phone viewport.                                         | `Growth*.tsx`, `SearchGrowthPanel.tsx`                                                                               |
| Browse additional candidates in bounded pages and bound comparison-page choices.                                                    | Expanded source coverage must not create thousands of cards or quadratic option lists in the browser. See `README.md` for the current bounds.                        | `SearchGrowthPanel.tsx`, `GrowthPageCard.tsx`                                                                        |

## Actions, safety and follow-up

| Decision                                                                                                                                | Why                                                                                                                                                                                       | Code / related decision                                                                                           |
| --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Provider drill-downs require an explicit button; weekly checks require a separate enable action.                                        | Viewing a panel must not trigger paid work, redundant Google reads or an unwanted schedule.                                                                                               | `GrowthEvidencePanel.tsx`, `GrowthMonitor.tsx`; backend `ga4/decision.md`                                         |
| Treat URL Inspection as Google's stored index evidence and sitemap counts as sitemap-level metadata.                                    | Neither can request general-page indexing, prove live indexability or guarantee rankings. General URL submissions through Google's restricted Indexing API are not part of this workflow. | `SearchPerformanceIndexing.tsx`, `indexingGuidance.ts`; backend `gsc/decision.md`                                 |
| Keep AI drafts and editor references validated, project-scoped and expiring in tab storage. Send/apply/publish remain explicit actions. | Another project must not inherit evidence; opening a brief must not spend credits or overwrite customer work. A basic refresh-queue keyword seeds only a blank, clean draft.              | `growthBrief.ts`, `sam/analystDraft.ts`, `content-editor/searchContext.ts`; content-editor and SAM decision files |
| Save immutable server-derived baselines and preserve the selected goal/comparison context.                                              | A later configuration change or repeated click must not rewrite the evidence used to assess a page edit. Before/after and unchanged-page comparisons remain observational.                | `GrowthActions.tsx`, `GrowthPageCard.tsx`; backend `ga4/decision.md`                                              |
| Separate saved prompt citations from Google's native AI report and optional BigQuery setup.                                             | These sources do not share coverage. External entry points must not be presented as implemented automatic ingestion or a complete AI visibility picture.                                  | `GrowthCitationEvidence.tsx`, `SearchGrowthPanel.tsx`                                                             |

## IndexNow on this page

- **Decision:** the IndexNow card sits at the bottom of the page, shown with or
  without a Search Console connection, and works for any website.
- **Why:** the owner wants manual indexing requests where customers look at
  search data, including sites that are not on WordPress.
- **Preserve:** the heading says Bing and other engines and that Google does
  not use IndexNow (Google indexing is URL Inspection in Search Console);
  nothing outbound runs on page load.
- **Trace:** `SearchPerformancePage.tsx`; rules in
  `../indexnow/decision.md` and `backend/internal/indexnow/decision.md`.

## Verification and future changes

- Preserve explicit-action/no-paid-call boundaries in `SearchGrowthPanel.test.ts`
  and `tests/search-growth.spec.ts`.
- Browser regressions must verify full mobile metric values, form editing,
  disclosure behavior and no document-wide overflow, as well as desktop tables.
- Update this file in the same commit whenever this feature changes. State the
  choice, reason, affected code, trade-off and verification/constraints; replace
  superseded statements rather than accumulating contradictory appendices.
