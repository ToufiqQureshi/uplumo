# Plans, credits and usage

Shows subscription status, plan limits, credit balances and usage; starts checkout or cancellation only after user action.

If a downgrade leaves more active sites, seats or tracked keywords than the
new plan allows, Billing shows the exact counts. Existing data stays intact;
paid provider actions resume after the workspace is brought within the limits.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `plan/planApi.ts`
- `billingApi.ts`
- `usageApi.ts`
- `backend/internal/billing`
- `backend/internal/plans`
- `backend/internal/razorpay`

`credits/TopUpsCard.tsx` lists server-configured one-time packs. Owners/admins can
open Razorpay; payment verification polls the organization-scoped order status
and refreshes credits only after `paid`. Subscription and one-time order checkout
share the lazy-loaded provider script. Daily limits still apply after purchase.
`CreditsCard.tsx` explains upcoming scheduled usage and the estimated balance
remaining after checks; negative estimates warn before schedules exhaust it.
Billing period end dates render as UTC calendar dates so a workspace west of
UTC does not see the previous day for a midnight refill.

The owner website tracking loader records checkout-button intent as
`subscription_checkout_<plan>` through `data-uplumo-event`. These click counts
are not confirmed payments; see `decision.md` for the boundary.
