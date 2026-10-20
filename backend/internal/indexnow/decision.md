# IndexNow: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Keeps one IndexNow key per project, checks that the customer hosts it, and
submits new or changed URLs by hand or after a live WordPress publish.

## Decisions to preserve

- **Paid plans only (Basic and above).** The owner's call: IndexNow costs
  Uplumo nothing per call, but it is a paid-plan benefit. _Why:_ owner
  decision; the gate lives in the plan catalog so the pricing page and the
  check cannot disagree.
- **Per-workspace hourly limit, not a global one.** _Why:_ all submissions
  leave from Uplumo's servers; a shared limit would let one heavy workspace
  block every other customer (the mistake recorded in `CLAUDE.md`).
- **Setting up the key turns automatic submission on.** _Why:_ a separate
  toggle would add a setting and a column for no customer need; a project
  without a hosted key would only get engine refusals.
- **Bing first, others only when Bing is unavailable.** _Why:_ the protocol
  asks sites to submit to one engine because participants share; sending to
  all would triple the load for nothing.
- **Check the key file before submitting.** _Why:_ engines answer 202 while a
  key is unverified, so a missing file would look like success; the check
  catches it with a message the customer can act on.
- **Body capped at 2 MB, with its own error.** _Decision:_ a submission body over 2 MB answers `413 too_large` ("split the list"), even when it holds fewer than 10,000 URLs. _Why:_ 10,000 URLs of 2,048 characters would need 20 MB per request held in memory; typical URLs (about 200 characters) fit 10,000 in 2 MB. Before, an oversized body got a misleading "Send a JSON object" (found in review). _Trace:_ `handler.go` (`decode`), `TestHandlerRefusals` "body over 2 MB".
- **No submission history.** _Why:_ Bing Webmaster Tools already shows what
  IndexNow received; the screen shows the latest result only.
- **No MCP tool yet.** _Why:_ not needed for the first customers; add one
  that calls `Service.Submit` when agents need it.

## Dependencies

`projects` table, billing feature gate, Redis, `audit.Guard`,
`sitepublish` (calls `SubmitPublished` through its `PageSubmitter`
interface).

## Verification

Start the Redis integration-test deadline after the clock-hour rollover wait.
The guard can wait 15 seconds, longer than the 10-second request budget; starting
the context first made healthy Redis fail at the hour boundary. Preserve the
workspace-isolation assertions and bounded Redis deadline. This changes only
test setup in `repository_integration_test.go`, not runtime limits or submissions.

`go test ./internal/indexnow/` (fake engines and sites through an in-process
transport), with `TEST_DATABASE_URL` and `TEST_REDIS_URL` for the key store
and the limiter. `TestPublishSubmitsLivePagesToIndexNow` in `sitepublish`
covers the hook.
