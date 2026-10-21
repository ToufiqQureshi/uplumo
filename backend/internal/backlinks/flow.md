# Backlink research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/backlinks/{overview|rows|domains|pages|...}
- **Source:** `handler.go`, `service.go`, `provider.go`, `target.go`

## Request or job flow

Handler validates scope, filters and plan before service; service normalizes the domain, requests provider tasks and maps supported data; UI/MCP callers share these services.

Paid calls per click: a new target buys the overview (summary + history, 2 tasks; exact URL 1; subfolder 2 list tasks). Only when the overview found links does the screen buy the selected table: 1 task for a block of 1,000 rows (about $0.06: $0.024 a task plus $0.000036 a row). Page sizes 50/100/200 and every page inside the first 1,000 rows are free; the next block is one more task. A target with no links buys nothing after the overview.

Whole list (`whole_list.go`): when the cached overview says the target has at most 1,000 backlinks (Backlinks tab) or referring domains (Referring Domains tab), the tab buys that whole list once, unfiltered and `as_is`, and cuts every view from it: sort, Best links, one per domain, filters, pages, domain expansion. Top pages become a whole list when their unfiltered first block holds everything. A longer profile asks the provider for the exact view, one task per 1,000-row block. Subfolder scope always asks the provider (its scope is a URL filter).

## Dependencies and cost boundary

DataForSEO paid endpoints and usage metering; project plan/credit checks.

## Failure, security and data invariants

- Validate target/filter limits and verify plan before any billed provider task.
- Preserve null/unsupported metrics by target scope; never present them as zero.
- Do not retry a paid task automatically; partial provider limits must be visible.
- Tables never load before the overview, and never when it reports zero backlinks.
- The client overview and selected tab are separate paid operations. Their
  query hooks disable automatic retries. Concurrent identical overview cache
  misses share one in-flight service call on a server instance; that call
  is detached from the first caller's context (capped at three minutes).
- A history failure after a billed summary answers with the summary and no
  trend, cached ten minutes; a credit refusal still fails the request.

Preparation boundary: `Overview` rechecks cancellation after `readCache` and before `flights.DoChan`. A cancelled request returns its context error
without starting a new provider operation. Existing shared work is unaffected.
