# Plans, credits and provider spend: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Owns plan access, credit ledger/reservations, subscriptions, usage and verified billing webhooks.

## Decisions to preserve

- The plan catalog is the only source for price/limits; never trust client-supplied amounts or entitlements. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- Credit holds, settlement/refund and webhook processing must be idempotent under retries and concurrency. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Monthly plan grants also deduplicate the verified charge body, not only Razorpay's unsigned delivery header. Persist the body fingerprint in the same transaction even when the rollover cap prevents a grant, so replay after spending cannot refill credits. Keep delivery-ID deduplication for state events. Historical records without a body fingerprint cannot provide this additional replay protection retroactively. Scope is byte-identical verified bodies; different valid event envelopes for one payment are not deduplicated by a payment ID. Trace: `service.go`, `repository.go`, `TestHandleWebhookConcurrentRedeliveriesApplyOnce`, `TestChargedBodyReplayCannotRefillSpentCredits` and `TestChargeAtRolloverCapCannotGrantOnReplayAfterSpending` in `service_test.go`. The transaction regressions require real Postgres and have not been run locally without `TEST_DATABASE_URL`.
- Scope spend to the responsible workspace; emergency global limits must not punish unrelated customers. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Preserve customer data on downgrade and pause only new paid provider work while count-based limits are exceeded. _Why:_ silently deleting sites, people or ranking history would destroy customer data; allowing new paid work would make the lower plan limits unenforceable.

- Preserve catalog prices, grants and limits during this replay fix. The current 4.35-week affordability assertion does not prove five weekly runs fit: Business may require 15,400 credits versus its 15,000 grant at the deepest allowed desktop check. See README for the calculation; no plan-policy change is implied.

## Dependencies

Postgres ledger/reservations/subscriptions; Razorpay adapter; plan catalog.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.
