# Cite Fix: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API:** `POST /api/v1/projects/{projectId}/cite-fix/report` (`handler.go`)
- **Source:** `store.go` (reads), `citefix.go` (`Build`), `checks.go` (copy and weights)

## Request flow

1. Session and project-access middleware run; the handler takes the project's
   workspace from the context and accepts only `{}`.
2. `Store.Report` (10 s timeout) finds the newest `running` audit and the
   newest `completed` audit of the project, both joined to the workspace.
3. No completed audit: `no_scan`. Any page row with `citation_signals_json`
   NULL: `needs_rescan`.
4. Otherwise it loads up to 10,000 pages (url, title, status, indexable, word
   count, external link count, signals) and the six issue types the checks read
   (`ai-crawlers-blocked`, `missing-llms-txt`, `missing-business-schema`,
   `several-phone-numbers`, `javascript-rendering-suspected`,
   `duplicate-content`), then `Build` scores them.
5. The JSON report goes back; lists are always arrays.

The signals themselves are written by the crawler: `audit/pageanalyzer.go`
reads them from the HTML it already fetched, `audit/repository.go` stores them
with the page row.

## Agent run flow (`agent.go`)

1. Validate mode, key, URL; plan gate; replay by idempotency key.
2. Find the URL in the workspace's latest completed scan (else 404).
3. Fetch the live page (15 s, 2 MB, must be 200 HTML with text) → content hash.
4. Cached completed run for same page, mode, hash → return it, 0 credits.
5. Claim the run (advisory lock per project, stale >10 min freed, rate 20/10 min, unique running per page+mode).
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

## Failure paths

- Database error or timeout: 500 `internal`, logged with the project id.
- Corrupt signals or issue details JSON: 500; the row is ours, so this is a bug
  to fix, not data to guess around.
- Reporter not configured: 503 `cite_fix_unavailable`.
- Foreign project: project access answers 404 before the handler runs; the
  store's workspace join is the second guard.
