# Cite Fix (frontend): flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **Route:** `/p/$projectId/cite-fix` → `CiteFixPage`
- **API:** `POST /api/v1/projects/{projectId}/cite-fix/report` with `{}` (`citeFixApi.ts`)

## Screen flow

1. On open, one TanStack Query (`["cite-fix", projectId]`) loads the report.
2. `running` set: a status line shows and the query refetches every 15 s until
   the scan finishes; otherwise it does not poll.
3. `no_scan`: empty state with "Scan my site" (link to Site Audit).
   `needs_rescan`: empty state with "Scan again". The button hides while a
   scan runs.
4. `ready`: score card (score, band, pages checked, scan date, category bars),
   "What to fix first" (first row open; each row shows impact, points, pages,
   then why, how and up to 20 affected pages), then "Pages, weakest first".

## Paid calls per user action

None. Opening the page, polling and expanding rows read free data. Scans start
on the Site Audit page, which makes no paid call either.

## Failure paths

A failed load shows `QueryError` with Retry. The endpoint is free, so the
default query retry is safe.
