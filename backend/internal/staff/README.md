# Staff console API

Serves `/api/v1/staff/*`: business numbers (users, payments, subscriptions, marketing funnel, provider cost) and account actions (block, unblock, delete, Razorpay payment sync). Every route answers 404 unless the session user is verified and in `STAFF_EMAILS`.

This is an internal developer/AI-agent guide. Read [flow.md](flow.md) and [business-logic.md](business-logic.md) before changing it.

## Source map

- `handler.go`: routes, staff gate, error mapping.
- `stats.go`: Overview, Subscriptions, Marketing, Finance queries.
- `actions.go`: Block, Unblock, Delete (via `../gdpr`), sync log, activity list.
- `staff_test.go`; migration `../database/migrations/00053_staff_console.sql`.
- Block enforcement lives where sessions are read: `../auth/repository.go`, `../auth/current_session.go`, `../emailauth/store.go` (`insertSession`), `../mcp/auth.go`, `../mcp/oauth.go`.

## API

GET overview, subscriptions, marketing, finance, actions. POST users/block {userId, reason}, users/unblock {userId}, users/delete {userId, confirmEmail}, payments/sync. GET payments/failed (live Razorpay read, last 30 days).
