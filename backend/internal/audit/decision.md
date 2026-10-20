# Site audit: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Runs an asynchronous bounded crawl, technical checks and optional Lighthouse/CrUX collection.

## Decisions to preserve

- Every initial and redirected URL passes SSRF checks. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- Per-organization crawl and PSI quotas, timeouts and page caps protect shared capacity. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- At-least-once jobs use idempotent row IDs; provider outages remain unavailable/partial, not false zeroes. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

Postgres audit tables; Redis progress/cache/quota; PageSpeed Insights and CrUX; jobs queue.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Honest rendering capability and reachable code

- **Decision:** Keep rendering unavailable until an actual renderer is wired into the crawler. Environment flags alone cannot advertise it. Reject requests before rows, jobs or credit holds. Remove unreachable renderer pricing and orphan crawl helpers, retaining the active credit lifecycle tests.
- **Why:** The HTTP crawler never executed JavaScript; accepting rendering requests misrepresented the work performed. Test-only port helpers duplicated inactive behavior.
- **Preserve:** HTTP crawl, SSRF checks, plan limits, job idempotency and credit cleanup remain intact. Existing rendering interfaces are retained for the runner lifecycle; they do not imply a usable renderer.
- **Trace:** `rendering.go`, `service_test.go` (`TestEnvironmentFlagsCannotAdvertiseAnUnwiredRenderer`, reproduced failing before the fix); `crawleraccess.go`, `crawlwindow.go`, `htmlresponse.go`, `issues_test.go`. `handler_test.go` checks the mounted capabilities/start responses and no rows/jobs; a forced-true mutation fails both service and HTTP regressions. No browser renderer/provider was added.
