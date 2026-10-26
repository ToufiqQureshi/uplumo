# Reports and share links

Creates report templates, lists/renders project reports and manages optional public share links.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `ReportsList.tsx`
- `ReportTemplateForm.tsx`
- `ReportViewer.tsx`
- `reportApi.ts`

- `ManualReportForm.tsx` (manual reports and client work logs)
- `manualReportHtml.ts` (escaped report rendering)

- `backend/internal/reports/share.go`

## Visual report builder

`ReportBuilder.tsx` owns editor state and save/import mutations. `ReportDataPanel.tsx`, `ReportCanvas.tsx` and `ReportVisualSettings.tsx` render the three editor panels; `ReportBuilderControls.tsx` shares field selectors. `reportDefinition.ts` validates bounded datasets/visuals and performs filters/aggregation. `reportDocument.tsx` uses the same renderer for canvas and self-contained HTML export. `reportSources.ts` reads existing analytics/GSC/GA4 endpoints only after Add project data.

The Reports route uses `?builder=true` for a new report and `?builder=true&edit=<id>` for editing. Saved visual reports expose Edit report in the viewer. Existing text/work-log reports and AI template briefs remain available.
