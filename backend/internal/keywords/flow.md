# Keyword and local business research: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/keywords/...; POST /api/v1/serp-locations/{search|prewarm}
- **Source:** `research_handler.go`, `research.go`, `saved_handler.go`, `saved.go`, `location_search.go`

## Request or job flow

Handler validates seeds, market, options and limits; service chooses supported DataForSEO endpoint; paid calls are gated/metered; results normalize/cache; save/list/tag/export operations use project-scoped Postgres.

## Dependencies and cost boundary

Equivalent concurrent keyword research cache misses join one in-flight lookup
within a server instance. A local research request can additionally require
national research and a city-scoped Ads volume call; both are still necessary.

DataForSEO; Google Ads for selected markets; Postgres saved rows; Redis caches; platform market registry.

## Failure, security and data invariants

- Validate every billed task before provider call; no blind retries of paid task-post endpoints.
- Cache keys include project/organization, geography and all result-changing options.
- Keep saved keyword rows and tags project-scoped; guard batches/caps and null metrics.
