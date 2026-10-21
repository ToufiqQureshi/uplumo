# Rank tracking: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Configures keywords, devices/locations and manual or scheduled rank checks with history.

## Decisions to preserve

- Rank checks consume provider credits; require a visible estimate/ceiling and enforce it again on the server. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Scheduled checks must stop or pause on limits; resuming must not duplicate tasks or charges. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- A partial provider error remains a partial run, not fabricated position data. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Opening tracker screens must not start a check. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.

## Verification pointers

- `RankTrackingFilters.test.ts`

## Optional suggestions need their own action

- **Decision:** Saving a tracker opens a choice between Find existing keywords (with credit guidance) and Skip/manual entry. Keep its TanStack query disabled until explicit refetch, join rapid double clicks and retain no automatic retries. Show API failure details without blocking manual entry; a clearly warned, explicit retry can recover cached failures without an automatic purchase.
- **Why:** A real browser regression observed a paid Labs lookup as soon as the setup step mounted. Creating a configuration does not authorize buying a separate suggestions report.
- **Preserve:** Supported-country handling and empty keyword selection remain. Opening, skipping, reload and focus buy no suggestions; an explicit action makes at most one request. Rank checks keep their separate selected-keyword/schedule/credit controls.
- **Trace:** `KeywordSuggestionStep.tsx`, `frontend/tests/rank-suggestions-spend.spec.ts`; [request contract](../domain/decision.md) and [server legacy/cost/concurrency guards](../../../../../backend/internal/domain/decision.md). API fixtures reproduce timing and payloads; no live provider spend was used.
