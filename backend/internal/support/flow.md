# Support reports: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/support/reports; GET /api/v1/support/reports/{ticketID}/screenshot; `RunNotifications` loop
- **Source:** `handler.go`, `support.go`, `notifications.go`

## Request or job flow

1. The session handler requires a signed-in user with a workspace and decodes a body capped at 2 MiB with unknown fields rejected.
2. `Create` validates kind, description, page path, app version, request ID, submission UUID and an optional PNG/JPEG screenshot.
3. Inside one transaction holding a per-user advisory lock, an existing row with the same submission ID is returned (same payload) or rejected (different payload); otherwise the hourly quota is checked and the report is inserted. The ticket ID is the row ID.
4. `NotifyPending` locks one due row at a time with `SKIP LOCKED`, sends the Loops template to support@uplumo.com and marks it notified, or backs off up to an hour for at most 10 attempts.
5. The email links a screenshot only by URL; the GET route serves it to the submitter or verified staff.

## Dependencies and cost boundary

Postgres `go_support_reports`; auth session; Loops transactional email. No paid provider or credit spend.

## Failure, security and data invariants

- A report is saved before any email is attempted; email failure never loses it.
- Five reports per user per hour, counted across workspaces.
- Screenshots are private: returned only to their submitter or a verified staff owner, never cached.
