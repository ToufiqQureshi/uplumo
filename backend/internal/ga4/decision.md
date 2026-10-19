# Google Analytics reports: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Reads project-selected GA4 reports, organic overview, measurement health and GA4/GSC search opportunities.

## Decisions to preserve

- Google calls are quota-limited; use bounded reads/retries and avoid duplicate queries. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Never expose OAuth tokens; property must belong to the connected organization/project. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- GA4 and GSC metrics differ in definitions/attribution; do not silently equate them. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

Google Analytics Admin/Data APIs and GSC; encrypted Google tokens; Postgres selected property.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

- Growth reporting uses Google organic sessions to align the source with GSC;
  clicks and sessions remain distinct measurements. Query filters classify
  evidence only, never conversion attribution.
- Limited reports suppress outcome recommendations; absent and ambiguous rows
  stay null. Finite metric validation and minimum samples precede findings.
- Baselines come from cached server reports, and repeated saves/applied updates
  cannot move an existing baseline or applied timestamp. Every SQL predicate
  carries organization and project; report storage is bounded to one per project.
- No background AI refresh, new paid research or automatic site change is
  authorized by opening an insight. The existing SAM meter bills explicit AI
  replies, and analyst mode enforces a read-only tool allowlist at execution.

- Event-specific session rate keeps all sessions in the denominator. Do not
  apply an event-name filter to the entire landing-page report or relabel all
  key events as the selected goal. Estimated goal-reaching session counts are
  only a labeled minimum-sample aid; the rate is the direct provider metric.
- Weekday-aligned year comparisons and user-selected unchanged pages provide
  context, not experimental attribution. Never replace absent queries with zero.
- Weekly checks require explicit opt-in and use no paid calls. Claim deadlines
  and revision fences protect replicas and stale writes; recheck entitlement and
  connection scope before fetching. Failure cannot trigger a fast retry loop.
- Metadata availability is not proof of correct instrumentation. Funnels use
  tracked event sequences, retain alpha/API limitations and do not synthesize
  abandonment from unrelated aggregate event counts.
