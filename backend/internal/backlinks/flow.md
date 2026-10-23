# Backlink research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/backlinks/{overview|rows|domains|pages|...}
- **Source:** `handler.go`, `service.go`, `provider.go`, `target.go`

## Request or job flow

Handler validates scope, filters and plan before service; service normalizes the domain, requests provider tasks and maps supported data; UI/MCP callers share these services.

Paid calls per click: a new target buys the overview (summary + history, 2 tasks; exact URL 1; subfolder 2 list tasks). Only when the overview found links does the screen buy the selected table: 1 task for a block of 1,000 rows (about $0.06: $0.024 a task plus $0.000036 a row). Page sizes 50/100/200 and every page inside the first 1,000 rows are free; the next block is one more task. A target with no links buys nothing after the overview.

Whole list (`whole_list.go`): when the cached overview says the target has at most 1,000 backlinks (Backlinks tab) or referring domains (Referring Domains tab), the tab buys that whole list once, unfiltered and `as_is`, and cuts every view from it: sort, Best links, one per domain, filters, pages, domain expansion. Top pages become a whole list when their unfiltered first block holds everything. A longer profile asks the provider for the exact view, one task per 1,000-row block. Subfolder scope always asks the provider (its scope is a URL filter).

The speculative Rows/Domains whole-list lookup goes through the normal
unfiltered first-block cache. If its returned total exceeds the overview's
estimate (or is unavailable), it is not considered complete, but its bought
block is still reusable. An identical unfiltered first-page request buys one
task, not a speculative task plus the same block again. Other filtered views
can still require a distinct task when the full profile does not fit.

Concurrent cold table-block requests on one server share one bounded purchase
per organization and complete query/block identity. A cancelled caller returns
without cancelling a purchase another waiter needs. Cancellation before flight
registration holds no credits and sends no task.

## Serve complete overviews from durable cache

After the Redis cache and singleflight recheck both miss, the shared lookup reads
the organization/target/scope snapshot from PostgreSQL. A fresh complete
snapshot refills Redis; a miss or database error falls through to provider work.
Writes happen after the paid result and are best-effort. Subfolder totals count
as complete. Summary-only responses after history failure use the short Redis
TTL and do not overwrite the durable complete snapshot. Rows and detail pages
remain provider-backed and unchanged.

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

If history is refused after a successful summary, return/cache that summary briefly rather than discarding paid work. The refused history is not sent; a repeat visit during the partial cache window buys no task.
