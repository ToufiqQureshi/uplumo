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

## Reconcile abandoned credit holds

- **Decision:** Mark SpendGuard holds as potentially dispatched in their reservation transaction. Persist the first provider/customer settlement outcome before charging; commit the charge and provider-spend row together. The hourly worker recovers known outcomes, refunds stale non-provider holds and flags uncertain provider holds for operator reconciliation.
- **Why:** A crash or failed credit transaction after a billed call previously returned the whole reservation, allowing paid work to become free. A reservation's age cannot prove the provider charged nothing.
- **Preserve:** No provider retry, duplicate settlement/spend row or automatic refund of an unknown provider outcome. Customer refunds for confirmed failed results remain possible; no price, grant or plan-limit change. Migration 41 conservatively marks existing open feature holds as uncertain.
- **Trace:** `credits.go`, `spendguard.go`, migration 41, `settlement_recovery_test.go`, hourly `reconcileCredits`. Tests use disposable Postgres with an injected ledger-insert failure and verify recovery charges exactly once. Zero-estimate calls record actual spend once, including when the customer has no balance. Unknown outcomes require provider evidence and operator resolution.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Brand Lookup cost hint follows the leaner fan-out

- **Decision:** The Brand Lookup typical cost is $0.29 (about 372 credits),
  down from $0.71 (about 910).
- **Why:** Brand Lookup now buys a `target_metrics` task and one prompt sample
  instead of six tasks; measured 22 Oct 2026 at $0.283 for a brand with 82
  mentions and $0.101 for one AI never mentions
  (`../aisearch/decision.md`).
- **Preserve:** The hint is never below a measured live click
  (`costs_test.go`); holds still use per-task ceilings, so the hint is not a
  spending limit.
- **Trace:** `costs.go`, `costs_test.go`.


## Reconcile only abandoned credit holds

- **Decision:** Refresh `updated_at` while a provider request is active; stale cleanup uses that heartbeat and grants a fresh usable bucket when refunding a pre-expiry legacy hold with no allocation rows.
- **Why:** A one-hour age check alone can refund a long-running stream that is still consuming provider resources, and a legacy balance-only refund is not consumable by the grant-based ledger.
- **Preserve:** Heartbeats stop when the request context ends or the hold settles; a crashed process stops refreshing; recovery uses the recorded outcome or leaves uncertain provider credits reserved for reconciliation. Refund and settlement remain locked and idempotent.
- **Trace:** `SpendGuard.Hold`, `creditHold`, `RefundStaleReservations`, and `restoreReservationCreditsTx`.
