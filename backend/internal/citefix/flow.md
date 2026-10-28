# Cite Fix: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API:** `POST /api/v1/projects/{projectId}/cite-fix/report` (`handler.go`)
- **Source:** `store.go` (reads), `citefix.go` (`Build`), `checks.go` (copy and weights)

## Request flow

1. Session and project-access middleware run; the handler takes the project's
   workspace from the context and accepts only `{}`.
2. `Store.Report` (45 s total deadline) finds the newest `running` audit and the
   newest `completed` audit of the project, both joined to the workspace.
3. It also reads scoped saved AI Visibility proof and Brand Lookup mentions,
   saved Google AI Overview rows, and (when a scan exists) the latest critical
   and warning issues through `audit.Service.GetResults`. These reads do not
   submit a paid provider task.
4. For saved proof misses, inspect at most three unique cited rival URLs per
   report. Add saved Brand Lookup top pages only when their sampled question
   matches that same miss; label their source separately. Fetch only through
   `audit.NewGuard().NewClient()` after checking robots.txt, with a 12 s timeout
   per page, all stale pages fetched together under one 8 s report budget, and
   a failed fetch remembered for 24 h (no refetch on every report read) and
   bounded HTML; use `audit.AnalyzeHTML` for the same citation signals as
   the customer's page. Cache by project, URL and content hash, reuse for seven
   days, and label app shells without rendering them.
5. No completed audit: `no_scan`. Any page row with `citation_signals_json`
   NULL: `needs_rescan`.
6. Otherwise it loads up to 10,000 pages (url, title, status, indexable, word
   count, external link count, signals) and the six issue types the checks read
   (`ai-crawlers-blocked`, `missing-llms-txt`, `missing-business-schema`,
   `several-phone-numbers`, `javascript-rendering-suspected`,
   `duplicate-content`), then `Build` scores them.
7. The JSON report goes back; lists are always arrays.

The signals themselves are written by the crawler: `audit/pageanalyzer.go`
reads them from the HTML it already fetched, `audit/repository.go` stores them
with the page row. When the user opts into JavaScript rendering in Site Audit,
only flagged app shells are rendered (maximum 20), then passed through the
same analyzer.

## Agent run flow (`agent.go`)

1. Validate mode, key, URL; plan gate; replay by idempotency key.
2. Find the URL in the workspace's latest completed scan (else 404).
3. Fetch the live page (15 s, 2 MB, must be 200 HTML with text) → content hash.
4. Cached completed run for same page, mode, hash → return it, 0 credits.
5. Claim the run (advisory lock per project, stale >10 min freed, rate 20/10 min, at most 5 uncharged failures per workspace per 24 h, unique running per page+mode).
6. Evidence: Search Console queries, AI Visibility proof (free, failures logged).
7. Hold the ceiling (≤ cap) → Flash call → (Deep) Pro call → settle provider vs customer cost → save result.

## Paid calls per user action

| Action              | Provider calls |
| ------------------- | -------------- |
| Open the Cite Fix page | 0 |
| Page polls while a scan runs (every 15 s) | 0 |
| Start a scan (Site Audit) | 0 paid; PageSpeed checks use Google's free quota as the audit already does |
| Agent Quick on one page | 1 OpenRouter call (Flash), ≤ 25 credits; 0 if cached or replayed |
| Agent Deep on one page | 2 OpenRouter calls (Flash, then Pro), ≤ 60 credits; 0 if cached or replayed |
| Batch of N pages | N runs one after another; confirmed at N × cap before the first |
| Read AI mentions on Cite Fix | 0; saved AI Visibility Brand Lookup only |
| Refresh mentions | No Cite Fix purchase; explicit link to AI Visibility's existing paid action |
| Refresh Google AI Overview | 1 paid queued SERP task-post action for up to 10 questions (≤ 16 credits); zero on cache hit |
| Collect queued Google AI Overview | Up to 10 free task-get requests; report load itself makes none |
| Fetch rival evidence | 0 paid calls; up to 3 public page + robots fetches, cache hit makes no fetch |
| List “Also fix for Google” | 0 provider calls; scoped saved audit result |

The 20-shell rendering ceiling is 39 credits (`$0.0015 × 20`, converted at
1,280 credits per dollar). Twenty pages on one registrable domain are four
paid `instant_pages` POSTs of five tasks each plus one free `raw_html` POST.
Failed/unusable rendering settles zero customer credits; each URL has a
per-audit claim and is never automatically retried.

## Failure paths

- Database error or timeout: 500 `internal`, logged with the project id.
- Corrupt signals or issue details JSON: 500; the row is ours, so this is a bug
  to fix, not data to guess around.
- Reporter not configured: 503 `cite_fix_unavailable`.
- Foreign project: project access answers 404 before the handler runs; the
  store's workspace join is the second guard.
- Ambiguous paid outcomes are never automatically resubmitted. Google SERP
  task-post uses `retry:false`; same-project concurrent refreshes serialize
  under a Postgres advisory lock. The same seven-day evidence cache is free.

## Paid-call test counts

- `overview_test.go`: click/double-click/concurrent refresh buys one task-post
  total for the stale question set; the follow-up cache read buys 0. Polling may
  issue task-get only, which is free. HTTP 500, timeout and malformed response
  each dispatch one task-post and are not automatically retried.
- `rivals_test.go`: the table is assembled from local fake HTTP pages; no
  provider client is invoked. Matched saved Brand Lookup pages join proof
  citations; own-domain and unrelated-question pages are excluded. A report
  fetches at most three unique pages, a fresh-cache read makes zero requests,
  and a seven-day or cross-workspace cache miss fetches only the three-page cap.
- `agent_test.go`: Quick/Deep hold ceilings remain 25/60 with rival context.
