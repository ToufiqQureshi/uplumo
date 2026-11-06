# Flow

```
Reports page -> "Create client report" dialog
  -> POST /api/v1/projects/{projectId}/reports/generate
     {title, periodDays: 7|28|90, sections[], note, nextSteps, share}
  -> sitereport.Service.Generate
       singleflight on org + project + title (double click = one run)
       Store.Project                     (Postgres, scoped to the org)
       in parallel, only the chosen sections:
         search    -> gsc.Service.GetPerformance + GetPageRows + GetQueryRows
         traffic   -> ga4.Service.GetOrganicOverview
         rankings  -> ranktracking.Service.ListConfigSummaries + LatestResults (stored snapshots)
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
- Google Analytics: 3 reads (current, previous, daily trend). Not billed.
- Rankings, audit and backlinks read stored rows. No DataForSEO or OpenRouter
  call is made, so nothing goes through the spend guard and no credit moves.
- Opening the report or the public link reads the saved HTML only.
