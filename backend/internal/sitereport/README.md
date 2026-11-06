# Site report

One click turns a project's connected data into an interactive, client-ready
report. The report is saved as a normal project report, so it gets the
existing viewer, PDF export, white-label brand bar and public share link.

## Code map

| File                | What it does                                                                 |
| ------------------- | ---------------------------------------------------------------------------- |
| `handler.go`        | `POST /api/v1/projects/{projectId}/reports/generate` (session + project access) |
| `service.go`        | Validates the click, collects sections in parallel, renders, saves, shares    |
| `sources.go`        | Adapters: Search Console (`gsc`), Google Analytics (`ga4`), rank tracking     |
| `repository.go`     | Reads `projects`, `go_audits`, `go_audit_issues`, `go_audit_lighthouse_results`, `backlink_snapshots` |
| `insights.go`       | Data-backed highlights and number formatting                                  |
| `render.go`         | Template functions and inline SVG charts                                      |
| `report.html.tmpl`  | The report document (no scripts; CSS-only tabs, filters and expanders)        |

Frontend: `frontend/src/client/features/reports/SiteReportBuilder.tsx`
(the "Create client report" dialog) and `generateSiteReport` in `reportApi.ts`.

Tables written: `reports` only, through `reports.Service.Save` (and `Share`
when the client link is switched on). No new table or migration.
