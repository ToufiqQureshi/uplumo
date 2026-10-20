# Plans, credits and usage: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Shows subscription status, plan limits, credit balances and usage; starts checkout or cancellation only after user action.

## Decisions to preserve

- The server catalog is the source of truth for price, entitlement and limits; never duplicate these rules in UI code. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- Credit reservations must settle/refund safely on provider failure and duplicate requests. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Never trust a browser-reported price, plan or provider cost. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Checkout/cancel are externally visible billing actions; require explicit user intent and preserve idempotency. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.

- **No offer page sells a plan the catalog does not have.** _Decision:_ the
  upstream `/yc` page ("YC Plan, $50, first month free") and its helpers
  (`PlanOfferCard`, `plan-offers.ts`, `checkout-url.ts`,
  `getCustomerPaidPlan`) are removed; `/yc` now falls through to not-found.
  _Why:_ its button called `attach`, which always opens the Razorpay checkout
  for Pro (₹2,499), so a customer would pay for something other than what the
  page showed. _Preserve:_ any future offer page reads its plan, price and
  checkout from `backend/internal/plans/catalog.go` through the billing API,
  never from constants in the browser. _Trace:_ `useBillingCustomer.ts`
  (`attach`), `plan-detection.ts`; checked with `pnpm ci:check` and `pnpm test`.

## Verification pointers

Checkout buttons expose `data-uplumo-event="subscription_checkout_<plan>"` to
the owner-approved website tracking loader. This measures plan-specific click
intent in Uplumo Analytics and GA4; it is not a purchase/success event. Preserve
the existing explicit checkout mutation, disabled state, idempotency and provider
metering. No billing amount, payment details or user identity is added to the
event. Shared configuration and opt-outs live in the backend site `decision.md`.

- `credits/CreditCostHint.test.ts`
- `credits/CreditsCard.test.ts`

## Remove orphan renderer conversion

- **Decision:** Remove shared `autumnSeoDataCreditsToUsd` after removing its only caller, the unsupported audit-rendering estimate.
- **Why:** Unused price conversion implied an unavailable service and duplicated a stale pricing path.
- **Preserve:** No active catalog, checkout, rank-tracking estimate, credit settlement or plan price changes.
- **Trace:** Shared `billing.ts`; audit launch decision; frontend type, unused-code and test checks.
