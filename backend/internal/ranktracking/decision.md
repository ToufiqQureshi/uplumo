# Rank tracking engine: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Schedules and executes bounded rank checks, stores runs/results and settles provider cost.

## Decisions to preserve

- Check estimated ceiling again before execution; a browser estimate is never authority. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Use idempotency and task state to prevent duplicate paid work across retries/restarts. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Scope caps to the workspace causing spend; expose partial failures and enforce schedule/pause rules. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

DataForSEO SERP; Postgres configs/runs/tasks/snapshots; Redis jobs/locks; billing/spend guards.

## Idempotent live rank checks

- **Decision:** The live check path (`lookupAll`) now attaches a stable
  `spend.WithIdempotencyKey("rank-live:<run>:<keywordID>:<device>")` to each
  lookup before calling `CheckLive`, mirroring the queued path's per-run/chunk
  post key.
- **Why:** A crash between provider billing and snapshot persistence could make a
  job retry re-buy the same live SERP pairs. The spend guard refuses a replay of
  the same key, so a retried job buys nothing for pairs already billed.
- **Preserve:** Each keyword/device within a run has a distinct key (no false
  dedup across devices or runs); no retry is added, and the key stays under the
  guard's 128-character limit. A pair the retry cannot re-check is reported on
  the run as "not checked (already paid in an earlier attempt), no charge" —
  never the browser's "already submitted" wording — because the customer did
  not submit it twice and must not be charged again.
- **Trace:** `checks.go` (`lookupAll`, `liveCheckIdempotencyKey`, `runErrorText`),
  `checks_test.go:TestLookupAllUsesAStablePerPairKeyAcrossARetry` (crash+retry:
  the same key per pair, distinct across pairs) and
  `checks_test.go:TestRunErrorTextHidesTheBrowserReplayMessage` (the visible
  message is "not checked, no charge"), plus the key-format test in
  `queued_test.go:TestLiveCheckIdempotencyKeyIsStablePerPair`.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.


- **Preserve retries across edits:** Live checks use a stable run/keyword/device hold identity, independent of the current config body. A retried pair cannot become a second billed lookup after a domain, location, language or depth edit.
