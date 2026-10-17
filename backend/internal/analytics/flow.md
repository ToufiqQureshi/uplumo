# Uplumo Analytics: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/analytics/site; POST /collect; GET /api/v1/projects/{projectId}/analytics/summary
- **Source:** `handler.go`, `service.go`, `repository.go`

## Request or job flow

Analytics UI creates a project site key; the published tracker posts bounded events; service validates bot/domain/rate rules; repository stores events; summary queries aggregate them.

## Dependencies and cost boundary

Postgres analytics tables; Redis daily salt/rate-limit counters; no paid SEO provider.

## Failure, security and data invariants

- Do not store cookies or raw IP addresses.
- Public ingestion must keep same-origin/domain validation, bot filtering, payload bounds and per-IP/site limits.
- Custom events must not inflate pageview totals or bypass ingestion checks.

## Enquiry workflow

Member selects an event → free report aggregates matched visitor-days → member
records a baseline before editing a page → repository freezes that week's
metrics under the project lock → comparison becomes ready after seven complete
post-change days. Missing instrumentation and small samples stay explicit;
there is no background email or automatic site edit.
