# Flow

```
Reports page -> "Create client report" dialog
  -> POST /api/v1/projects/{projectId}/reports/generate
     {title, periodDays: 7|28|90, sections[], note, workCompleted, nextSteps, share}
  -> sitereport.Service.Generate
       singleflight on org + project + author + validated input
       Store.Project                     (Postgres, scoped to the org)
       in parallel, only the chosen sections:
         search    -> gsc.Service.GetPerformance + GetPageRows + GetQueryRows
         traffic   -> ga4.Service.GetOrganicOverview + RunReport (channels, landing pages, key events)
         analytics -> analytics.Service.Summarize (stored aggregates)
         rankings, competitors -> one shared ranktracking latest-results read
         ai_visibility -> aisearch.Tracking.State + Answers (saved answer and citation evidence)
         local_visibility -> current business and saved check, grid and review snapshots for its CID
         audit     -> go_audits / go_audit_issues / go_audit_lighthouse_results
         backlinks -> backlink_snapshots (latest stored row)
       Render (html/template, inline SVG)
       reports.Service.Save  -> reports table
       reports.Service.Share -> share_token (when share = true)
  <- {reportId, title, shareToken, included[], skipped[{section, reason}]}
  -> app opens /p/{projectId}/reports/{reportId}; client opens /s/{token}
```

## Paid calls per click: 0

- Search Console: at most 10 reads (8 for performance with comparison, 1 page
  list, 1 query list). Google does not bill them. Asserted in
  `TestGSCSourceReadsAtMostTenFreeRequests`.
- Google Analytics: up to 7 report reads (three for organic overview, one
  each for acquisition and landing pages, two for key events). Not billed.
  Asserted in `TestGA4SourceReadsSevenFreeReportsAndKeepsDashboardEvidence`.
- The remaining sections read stored rows. No DataForSEO or OpenRouter
  call is made, so nothing goes through the spend guard and no credit moves.
- Opening the report or the public link reads the saved HTML only.
