# Customers (staff)

Lists every signed-up user with onboarding answers, plan, active websites and Google connections for Uplumo staff, as a table API and a CSV export for marketing.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for privacy and access invariants.

## Source map

- `customers.go`
- `handler.go`
- `customers_test.go`

## API/trigger

GET /api/v1/admin/access; GET /api/v1/admin/customers; GET /api/v1/admin/customers.csv. Staff = verified email in `STAFF_EMAILS`.
