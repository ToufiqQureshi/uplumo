# Staff console API: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

1. `WithSession` resolves the user; `staffOnly` asks `customers.Service.IsStaff` (verified email in `STAFF_EMAILS`). Anyone else: 404 `not_found`.
2. Reads are plain Postgres queries, `Cache-Control: private, no-store`. Money is in subunits per currency; MRR = active subscriptions × `plans` catalog price; collected = `go_billing_payments` amount − refunded; provider cost = `go_provider_spend` (USD).
3. **Block:** one transaction inserts `go_user_blocks` (idempotent), deletes the user's sessions and logs `go_staff_actions`. From then on session lookups, new sign-ins (password and Google), API keys and MCP OAuth tokens refuse the user. Unblock deletes the row.
4. **Delete:** refuses staff/self, a mismatched typed email, a live subscription (`authenticated`, `active`, `pending`) and a shared workspace; then `gdpr.Eraser` erases the user and owned workspaces in one transaction (payments kept) and the action is logged by user id.
5. **Sync:** `billing.SyncSubscriptionPayments` (read-only Razorpay): copies paid subscription invoices, then lists Razorpay refunds since the first ledger payment and raises each payment's refunded amount to the sum of its processed refunds (never lowers it). 503 when Razorpay is not configured. Logged with counts.
6. **Failed payments:** `GET /api/v1/staff/payments/failed` calls `billing.ListFailedPayments`, which reads Razorpay `GET /payments` for the last 30 days (100 per page, at most 10 pages, `truncated` past that), keeps `status = failed` and marks `paidLater` when the same order or email has a captured payment after it. Nothing is stored. 503 without Razorpay, 502 when Razorpay errors.

## Cost boundary

No paid provider call. Razorpay invoice and payment reads are free.
