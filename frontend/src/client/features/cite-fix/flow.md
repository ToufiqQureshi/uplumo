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

## Agent flow

Pick pages → choose Quick/Deep → "Fix N pages" → confirm (at most N × cap,
balance shown) → pages run one at a time with live status → results refresh
→ "Open in Content Editor" creates a draft (`content/drafts/save`) and opens it.

## Paid calls per user action

Opening the page, polling, expanding rows and loading results: none. Each
confirmed page: one agent run (see `backend/internal/citefix/flow.md`).

## Failure paths

A failed load shows `QueryError` with Retry. The endpoint is free, so the
default query retry is safe.
