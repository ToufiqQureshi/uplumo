# Customers (staff): flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** GET /api/v1/admin/access, /api/v1/admin/customers, /api/v1/admin/customers.csv
- **Source:** `handler.go`, `customers.go`
- **Caller:** `frontend/src/client/features/staff-console/`

## Request flow

1. Session middleware resolves the user; no session answers 401.
2. `IsStaff` checks the user row is `email_verified` and its email is in `STAFF_EMAILS`; otherwise 403. `/admin/access` returns `{staff}` so the sidebar can show the link.
3. `List` runs one read-only query: each user with the workspace they own (else first joined), the active Razorpay plan (else `free`), active websites excluding the domain-less `Default` placeholder, GSC/GA4 connection flags and the `user_onboarding_answers` row. Newest signups first.
4. JSON for the page, or CSV with formula-looking cells prefixed by `'`.

## Dependencies and cost boundary

Postgres only. No paid provider call, no writes.

## Failure, security and data invariants

- Responses are `Cache-Control: private, no-store`.
- Non-staff, including members of the staff workspace, never get rows.
