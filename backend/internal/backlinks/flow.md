# Backlink research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/backlinks/{overview|rows|domains|pages|...}
- **Source:** `handler.go`, `service.go`, `provider.go`, `target.go`

## Request or job flow

Handler validates scope, filters and plan before service; service normalizes the domain, requests provider tasks and maps supported data; UI/MCP callers share these services.

## Dependencies and cost boundary

DataForSEO paid endpoints and usage metering; project plan/credit checks.

## Failure, security and data invariants

- Validate target/filter limits and verify plan before any billed provider task.
- Preserve null/unsupported metrics by target scope; never present them as zero.
- Do not retry a paid task automatically; partial provider limits must be visible.
- The client overview and selected tab are separate paid operations. Their
  query hooks disable automatic retries. Concurrent identical overview cache
  misses share one in-flight service call on a server instance; that call
  is detached from the first caller's context (capped at three minutes).
- A history failure after a billed summary answers with the summary and no
  trend, cached ten minutes; a credit refusal still fails the request.
