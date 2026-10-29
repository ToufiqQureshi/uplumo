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
   then why, how and up to 20 affected pages), then "Pages, weakest first";
   saved mentions, proof-linked rival comparisons, Google AI Overview results,
   and latest critical/warning audit issues appear below.

## Evidence actions

- **AI mentions:** display the saved AI Visibility Brand Lookup, including top
  domains and pages. No Cite Fix refresh call exists; the link opens AI
  Visibility at its established price.
- **Rival comparison:** show up to three fetched/cached AI-cited pages for
  missed questions. Saved Brand Lookup top pages join a comparison only when
  their sampled question matches the miss; source labels distinguish them from
  AI Visibility proof. Signals compare answer-first length, author, date,
  schema, statistics, FAQ and word count against the customer's best audited page.
  Blocked/unavailable pages are omitted; app shells are labeled and never
  rendered. “Compare with rival” scrolls to the evidence for that miss.
- **Google AI Overview:** click the action to open a price confirmation. Only
  after confirmation, send one non-retrying paid refresh for at most ten tracked
  questions (up to 16 credits, actual cost charged). Results cache for seven
  days; background collection uses free task-get reads.
- **Also fix for Google:** latest audit critical/warning issues are free and
  link to Site Audit.

## Agent flow

Pick pages → choose Quick/Deep → "Fix N pages" → confirm (at most N × cap,
balance shown) → pages run one at a time with live status → results refresh
→ "Open in Content Editor" creates a draft (`content/drafts/save`) and opens it.

## Paid calls per user action

Opening the page and reading saved evidence: no paid calls. A confirmed agent
page runs once; a confirmed Overview click sends one paid task-post (see
`backend/internal/citefix/flow.md`). Collection/poll requests are free and do
not use browser retries.

## Failure paths

A failed load shows `QueryError` with Retry. The endpoint is free, so the
default query retry is safe.

The free report includes `topics` and `proof.visibility`. The topic pillar
button opens the Content Editor route with a prefilled keyword; the user writes
or explicitly chooses a paid action later. Citation history distinguishes the
saved source list from URLs actually present in answer text.
