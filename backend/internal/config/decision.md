# Runtime configuration: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Loads and validates typed server configuration and environment-backed secrets.

## Decisions to preserve

- Never commit secrets or include their values in logs/errors. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- Distinguish required core settings from optional provider keys so an unavailable paid feature does not take down the whole app. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Validate URL/key formats once at startup and inject config rather than reading global env throughout handlers. _Why:_ Migration and test mistakes can prevent startup or damage customer data, so ordering and isolation must remain explicit.

## Dependencies

Environment variables and deployment secret store.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Dollar plan ids

- **Decision:** `RAZORPAY_PLAN_ID_BASIC_USD`, `_PRO_USD` and `_BUSINESS_USD` are optional. Startup fails if any Razorpay plan id is used for two variables.
- **Why:** a webhook names only the Razorpay plan; a shared id could grant the wrong plan or currency. See [billing decision](../billing/decision.md#us-dollar-billing-for-customers-outside-india).
