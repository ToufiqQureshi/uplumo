# Staff console: flow

1. `StaffLayout` waits for the session guard, then asks `/api/v1/admin/access` (no retry). Not staff: `NotFound`. Staff: tabs render.
2. Each tab loads one `GET /api/v1/staff/*` endpoint through TanStack Query with `retry: false`; errors show inline.
3. Customers: **Block** posts `/api/v1/staff/users/block` (optional reason); **Unblock** posts `/unblock`; **Delete** requires typing the user's email and posts `/delete` with `confirmEmail`. Success refreshes every `staff` query and the customer list. Your own row shows no actions; the server also refuses staff and self.
4. Finance **Sync from Razorpay** posts `/api/v1/staff/payments/sync` once per click (no retry) and refreshes the console.
5. Finance **Failed payments, last 30 days** loads `GET /api/v1/staff/payments/failed` when the tab opens (no retry). The server reads Razorpay's payment list live (free, up to 10 pages of 100) and stores nothing.
6. Nothing here calls a paid provider. Money values arrive in paise/cents and are shown per currency, never converted.
