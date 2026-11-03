# Staff console API: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

1. `WithSession` resolves the user; `staffOnly` asks `customers.Service.IsStaff` (verified email in `STAFF_EMAILS`). Anyone else: 404 `not_found`.
2. Reads are plain Postgres queries, `Cache-Control: private, no-store`. Money is in subunits per currency; MRR = active subscriptions × `plans` catalog price; collected = `go_billing_payments` amount − refunded; provider cost = `go_provider_spend` (USD).
3. **Block:** one transaction inserts `go_user_blocks` (idempotent), deletes the user's sessions and logs `go_staff_actions`. From then on session lookups, new sign-ins (password and Google), API keys and MCP OAuth tokens refuse the user. Unblock deletes the row.
4. **Delete:** refuses staff/self, a mismatched typed email, a live subscription (`authenticated`, `active`, `pending`) and a shared workspace; then `gdpr.Eraser` erases the user and owned workspaces in one transaction (payments kept) and the action is logged by user id.
5. **Sync:** `billing.SyncSubscriptionPayments` (read-only Razorpay); 503 when Razorpay is not configured. Logged with counts.

## Cost boundary

No paid provider call. Razorpay invoice reads are free.
