# Reports and share links: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. User opens Reports; queries load existing report records/templates.
2. Creating or editing a template/report sends explicit mutations through `reportApi.ts`.
3. Go scopes records to the project, builds the rendered view and persists share settings when enabled.
4. A recipient opens a share URL and receives only the report content allowed by its share state.

## Implementation map

- `ReportsList.tsx`
- `ReportTemplateForm.tsx`
- `ReportViewer.tsx`
- `reportApi.ts`
- `backend/internal/reports/share.go`

## API and service dependencies

Go package: `backend/internal/reports`; branding is read from `backend/internal/branding` where applicable.

## Manual reports and client work logs

The report composer supports a standard report and a client work log. A standard report takes a title, summary and plain-text body. A work log takes a title, summary, reporting month and completed work, with optional outcomes, evidence links and next steps. The browser limits the title to 120 characters, summary to 2,500, standard body to 60,000, and total work-log details to 20,000. It permits at most 10 credential-free HTTP(S) evidence URLs and escapes every field into a self-contained HTML document before saving it through `reportApi.ts` as `Manual` or `Manual work log`. This path does not call an AI or paid data provider.

Sharing a work log is a separate opt-in action after the save succeeds. If share-link creation fails, the report remains saved and the UI reports that only the link failed. Report list queries are invalidated and the user is sent to the saved report.

## Cost, security and failure boundaries

- Public sharing is opt-in; never expose a report through an unguessable-but-unchecked ID alone.
- Keep organization/project authorization on private routes and revocation effective for shared links.
- Report generation should use stored data unless a clearly labeled refresh action is requested.

## Visual reports

1. Create report opens a three-panel editor without fetching source data. Upload CSV parses locally with installed Papa Parse, preserving blanks/zeros and rejecting malformed rows or duplicate headers. Add project data explicitly reads one existing Uplumo summary, Search Console report or GA4 organic overview for the import period; TanStack caches that source/project/date request for one minute with retries disabled. No DataForSEO or AI calls occur.
2. Imports are snapshots, not live connections. A report holds up to 6 sources, each 2,000 rows/30 fields, and 24 visuals. CSV uploads are at most 200 KB; cells at most 2,000 characters. Sources without date fields retain their labelled import period; report date filters apply only to configured ISO date fields.
3. KPI, line, bar and grouped-table visuals select a source, numeric metric or row count, breakdown, aggregation, optional case-insensitive contains filter, and up to 50 groups. Filters run before grouping. Line groups sort by label; others show highest values first. Average is arithmetic, not weighted. Notes explain findings/actions. Controls reorder visuals and choose half/full width.
4. Preview hides editing panels. Save validates fields and the combined 500,000-byte HTML/editor-data ceiling, sends `definition` separately from rendered HTML to the existing save route, invalidates report caches and opens the viewer. Existing shares remain enabled and show the updated saved snapshot. Import/save failures retain the draft; navigation/reload warns before unsaved changes are discarded.
5. Edit loads `definition` only via authenticated project-scoped get with `includeHtml: true`. Raw datasets never enter exported HTML. Public links and print/PDF use the existing sandboxed renderer and only contain displayed aggregates, labels and notes.

Verification: `reportBuilder.test.ts`, `reportDocument.test.ts`, `tests/report-builder.spec.ts`, plus the existing report API tests.
