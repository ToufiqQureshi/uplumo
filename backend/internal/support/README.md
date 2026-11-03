# Support reports

Saves customer bug, billing and suggestion reports from the Help page, then emails support@uplumo.com from a background loop.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for data and security invariants.

## Source map

- `handler.go`
- `support.go`
- `notifications.go`
- `../database/migrations/00051_support_reports.sql`

## API/trigger

POST /api/v1/support/reports; GET /api/v1/support/reports/{ticketID}/screenshot; `RunNotifications` every minute when `LOOPS_API_KEY` and `LOOPS_TRANSACTIONAL_SUPPORT_REPORT_ID` are set.
