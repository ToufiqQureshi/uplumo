# Business logic

- **Free by design.** The report only reads data the project already has.
  It never starts a rank check, audit or backlink refresh, because those cost
  money and the user clicked "report", not "buy data".
- **No empty or generic sections.** A section without data is left out of the
  document and returned in `skipped` with a reason the user can act on. If no
  chosen section has data, nothing is saved and the user sees every reason.
- **Highlights come from numbers.** Each highlight is computed from the
  snapshot (change vs previous period, top-10 count, critical issues, new and
  lost linking sites). There is no generic advice text.
- **No growth from nothing.** If the previous period has no Search Console
  impressions (a new site), the report shows no "was 0" comparison and no
  rising or falling searches: against an empty period every number would
  look like growth.
- **Charts show every day.** Google leaves out days with no data; the report
  fills them with 0 so the gap between two dates is drawn to scale. Axis
  labels are whole, evenly spaced counts.
- **One failing source does not fail the click.** A Google error skips that
  section and is logged; the rest of the report is saved.
- **Double clicks save once.** Concurrent requests with the same title share
  one run. A second click later with the same title gets the existing
  "title already exists" error from the reports service.
- **Safe to share.** The document has no scripts and loads nothing. Links are
  limited to http(s) URLs. It is served under the existing report sandbox and
  CSP, and the public link can be switched off from the Share dialog.
- **Global copy.** Plain English, `1,234` number style, dates as `7 Oct 2026`;
  nothing assumes a country.
- **White label.** The workspace's branding accent colours the report; the
  existing brand bar and "Prepared by" header come from the report viewer.
