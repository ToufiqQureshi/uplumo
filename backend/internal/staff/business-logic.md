# Staff console API: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when behavior or invariants change.

- **Decision:** verified `STAFF_EMAILS` gate on every route, 404 for everyone else. _Why:_ the console shows every customer's email and the company's revenue; workspace role or an unverified signup with a staff address must not open it, and its existence is not advertised. _Preserve:_ new routes go through `staffOnly`.
- **Decision:** Block is enforced where credentials are checked, not in the UI: session lookups, `insertSession`, API key and MCP OAuth lookups all exclude `go_user_blocks`. Blocking deletes sessions in the same transaction. _Why:_ owner asked that a removed user cannot keep using the product by any route. _Preserve:_ any new credential type must check `go_user_blocks`. A block does not cancel Razorpay: the UI says so.
- **Decision:** Delete reuses `../gdpr` and refuses a running paid plan. _Why:_ deleting a paying account would leave Razorpay charging a customer with no account; the owner cancels in Razorpay first. Payment rows survive (`go_billing_payments`, owner decision 3 Nov) for accounting; `go_provider_spend` still cascades with the workspace, so provider cost history of deleted workspaces is lost.
- **Decision:** every action is logged in `go_staff_actions` by opaque user id, kept after erasure. _Why:_ who removed whom must stay answerable; no email is kept for an erased user.
- **Decision:** numbers are real and per currency; nothing is converted or estimated except MRR, which is labelled as catalog price × active subscriptions.

## Verification

`staff_test.go` (block ends sessions and is idempotent, unblock, staff/self protected, delete refuses paid plan and wrong confirmation then erases keeping payments, overview MRR and net revenue, routes 404 for customers/anonymous and 200 for staff); `../auth` `TestAuthenticateRefusesABlockedUser`; `../emailauth` `TestSignInRefusesAUserBlockedByStaff`. Mutations (drop the live-plan check, drop session deletion) fail tests.
