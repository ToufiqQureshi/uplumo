# Site audit: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/audit/{start|status|results|history|progress|delete|capabilities|lighthouse/...}
- **Source:** `handler.go`, `runner.go`, `crawler.go`, `service.go`

## Request or job flow

Start validates URL/plan/capacity, creates an audit and job; worker discovers and crawls guarded URLs; checks persist page/issue rows; optional PSI/CrUX calls use quota guards; client polls stored progress/results. If the user opted into JavaScript rendering and the provider is wired, the worker then renders only crawler-flagged app shells (maximum 20), analyzes the returned HTML through the regular page analyzer, and replaces the shell's page signals and issues.

Each page row also stores its AI citation signals (`citation_signals_json`),
read from the same HTML; Cite Fix (`backend/internal/citefix`) scores the
latest completed audit from them without crawling again.

## Dependencies and cost boundary

Postgres audit tables; Redis progress/cache/quota; PageSpeed Insights and CrUX; jobs queue.

## Failure, security and data invariants

- Every initial and redirected URL passes SSRF checks.
- Per-organization crawl and PSI quotas, timeouts and page caps protect shared capacity.
- At-least-once jobs use idempotent row IDs; provider outages remain unavailable/partial, not false zeroes.
- JavaScript rendering takes one workspace-scoped spend-guard hold before any
  provider request; the maximum is 39 credits for 20 pages and the actual
  delivered price is settled. A failed/unusable render charges zero customer
  credits. The paid request uses `retry:false`, a unique page claim prevents
  duplicate rendering within the audit, and returned HTML cannot bypass the
  normal analyzer. The free seven-day `raw_html` retrieval runs after each
  batch, only for that batch's successful task IDs, so a later failed batch
  never discards pages already paid for.

JavaScript rendering is opt-in and limited to app-shell pages; without a wired
DataForSEO renderer, capabilities report false and the start request fails
before persistence or dispatch. PSI runs separately on Google.
