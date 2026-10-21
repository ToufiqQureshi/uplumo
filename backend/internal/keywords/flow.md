# Keyword and local business research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/keywords/...; POST /api/v1/serp-locations/{search|prewarm}
- **Source:** `research_handler.go`, `research.go`, `saved_handler.go`, `saved.go`, `location_search.go`

## Request or job flow

Handler validates seeds, market, options and limits; service chooses supported DataForSEO endpoint; paid calls are gated/metered; results normalize/cache; save/list/tag/export operations use project-scoped Postgres.

## Dependencies and cost boundary

Equivalent concurrent keyword research and same-depth SERP cache misses join
one in-flight lookup within a server instance. A local research request can additionally require
national research and a city-scoped Ads volume call; both are still necessary.

DataForSEO; Google Ads for selected markets; Postgres saved rows; Redis caches; platform market registry.

A normal cold national Auto search buys two Labs tasks (suggestions + ideas),
then the UI requests one shallow SERP task. Thin Auto results can add one
related-keywords fallback task. The exact seed row comes from the existing
suggestions response, with no new task. Clickstream doubles Labs task pricing.
Research and SERP cache hits buy zero tasks; equivalent cold SERPs at the same
depth share one task per server instance and organization.

## Failure, security and data invariants

- Validate every billed task before provider call; no blind retries of paid task-post endpoints.
- Cache keys include project/organization, geography and all result-changing options.
- Keep saved keyword rows and tags project-scoped; guard batches/caps and null metrics.
- Auto research never discards a billed leg: one failed leg yields the other's
  rows as a short-lived partial answer, not an error and not a day-long cache.

Preparation boundary: `Research` rechecks cancellation after `cached` and before `flights.DoChan`. A cancelled request returns its context error
without starting a new provider operation. Existing shared work is unaffected.
