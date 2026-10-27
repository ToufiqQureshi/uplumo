# Site audit: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/audit/{start|status|results|history|progress|delete|capabilities|lighthouse/...}
- **Source:** `handler.go`, `runner.go`, `crawler.go`, `service.go`

## Request or job flow

Start validates URL/plan/capacity, creates an audit and job; worker discovers and crawls guarded URLs; checks persist page/issue rows; optional PSI/CrUX checks use quota guards; client polls stored progress/results.

Each page row also stores its AI citation signals (`citation_signals_json`),
read from the same HTML; Cite Fix (`backend/internal/citefix`) scores the
latest completed audit from them without crawling again.

## Dependencies and cost boundary

Postgres audit tables; Redis progress/cache/quota; PageSpeed Insights and CrUX; jobs queue.

## Failure, security and data invariants

- Every initial and redirected URL passes SSRF checks.
- Per-organization crawl and PSI quotas, timeouts and page caps protect shared capacity.
- At-least-once jobs use idempotent row IDs; provider outages remain unavailable/partial, not false zeroes.

JavaScript rendering is not wired into the crawler. Capabilities report false;
rendering starts fail before persistence/dispatch even when legacy environment
hints are set. Audits inspect initial HTTP HTML; PSI runs separately on Google.
