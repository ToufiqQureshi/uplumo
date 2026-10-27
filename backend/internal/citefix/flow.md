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

## Paid calls per user action

| Action              | Provider calls |
| ------------------- | -------------- |
| Open the Cite Fix page | 0 |
| Page polls while a scan runs (every 15 s) | 0 |
| Start a scan (Site Audit) | 0 paid; PageSpeed checks use Google's free quota as the audit already does |

## Failure paths

- Database error or timeout: 500 `internal`, logged with the project id.
- Corrupt signals or issue details JSON: 500; the row is ours, so this is a bug
  to fix, not data to guess around.
- Reporter not configured: 503 `cite_fix_unavailable`.
- Foreign project: project access answers 404 before the handler runs; the
  store's workspace join is the second guard.
