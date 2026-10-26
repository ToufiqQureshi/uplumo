# DataForSEO transport: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Sends validated paid task requests, parses provider responses and records usage by organization.

## Decisions to preserve

- Provider may bill rejected/failed tasks: validate request before transport and never add automatic retries to task-post calls. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Attribute every billable task to the organization causing it; usage-write failure behavior must remain explicit. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Never put credentials in URLs/logs; cap body/response sizes and enforce timeouts. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.

## Dependencies

DataForSEO API, platform spend guard, Postgres usage ledger, Redis worker/cache as caller requires.

## "No Search Results" is 40102, not 40501 — and 40102 is billable

- **Decision:** Treat 40102 ("No Search Results") as the valid empty-result
  status AND charge it to the customer (it is a billed task). Treat 40501
  ("Invalid POST field") as our own bad request: it must surface as an error and
  must not be charged.
- **Why:** The code previously labelled 40501 as "No Search Results" and
  planned to bill it. Per the official error appendix
  (https://docs.dataforseo.com/v3/appendix/errors/), 40501 is "invalid POST
  field" — a request WE malformed. The real empty-result code is 40102, and the
  official help center
  (https://dataforseo.com/help-center/what-does-the-40102-error-mean) states a
  40102 task IS billable because the search was processed even though it
  returned no items (its example shows `cost: 0.002` on the task). Charging
  40501 would bill the customer for our bug; not charging 40102 would make
  Uplumo absorb a real provider cost.
- **Preserve:** No customer charge for non-success task errors other than 40102;
  provider cost is still recorded; a 40102 is delivered as "not ranking", not as
  an error, and its provider cost is passed on.
- **Trace:** `client.go` (`billableTask`, `responseSuccessfulCost`),
  `client_test.go` (`TestResponseSuccessfulCostChargesSuccessfulAndNoResultTasks`,
  `TestDoChargesForNoSearchResultsAndNotForInvalidField`), `envelope_test.go`,
  plus `ranktracking/serp.go` and `keywords/research_provider.go` callers.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Clickstream price ceilings

- **Decision:** Double the complete Labs estimate only when the task explicitly
  enables `include_clickstream_data`, including each task of a mixed batch.
- **Why:** DataForSEO charges double for this option, but the guard previously
  reserved the ordinary price. A 75-row task held $0.021 for up to $0.042 of work.
- **Preserve:** No plan/grant/markup change, unchanged ordinary task estimates,
  no extra provider calls, and actual task-reported settlement. Requests without
  sufficient credits are refused before purchasing the more expensive option.
- **Trace:** `pricing.go`, `pricing_test.go` (enabled/disabled/mixed batch), and
  https://docs.dataforseo.com/v3/dataforseo_labs-google-keyword_suggestions-live/.

## Missing responses do not prove zero provider spend

- **Decision:** Mark dispatched requests with unconfirmed responses as `ErrOutcomeUnknown`, retain their hold and stop additional budgeted purchases. Use standard `httptrace` to distinguish failure before request dispatch.
- **Why:** Empty/invalid responses previously settled at zero and restored credits even though the provider could have billed the accepted request.
- **Preserve:** No paid POST retry; explicit rejected requests still refund; known successful costs and intentional unusable-result refunds keep their existing rules. Budgeted timeout/5xx uncertainty requires reconciliation rather than guessing the missing leg's cost.
- **Trace:** `client.go`, `uncertain_test.go`, updated budget/timeout tests. Empty/malformed JSON, missing tasks and missing task-cost regressions failed with a zero settlement before the fix; tests also cover a caller swallowing the uncertainty. No live paid request was made for validation.

Documented account refusals 40200/40203/40210 release credits even without task-cost metadata; 40203 is classified as a billing stop so budget callers stop further purchases. Other missing costs remain uncertain. Source: https://dataforseo.com/help-center/limit-my-payments and the provider error appendix. `TestAccountCapRejectionReleasesCreditsAndStopsBudgetLegs` covers top-level and task-level $2/day refusals without making a live request.

## Cached-only requests never buy

- **Decision:** A request with `X-Cached-Only: 1` (set by `requireSession` into `spend.WithCachedOnly`) may be answered from cache, but `dataforseo.Client.Do` refuses any paid call (estimate above zero) and `DoBudgeted` refuses all work with `spend.ErrRefreshRequired` before a hold is made; `spend.WriteHTTPError` answers `409 refresh_required`. Free reference lookups still run. Without the header nothing changes (MCP, schedules, SAM).
- **Why:** The owner asked that a saved Backlinks or Domain link opened after its cache expired (24 h tables, 12 h domain reports) no longer buys the report again without a click. One check in the transport covers every feature that reads through it.
- **Preserve:** The refusal happens before the hold, so no credits are reserved and nothing is recorded; `spend.Refused` counts it, so partial-answer paths stop instead of caching half a report.
- **Trace:** `platform/spend/spend.go`, `platform/dataforseo/client.go` (`TestCachedOnlyRequestsNeverBuyProviderWork`), `httpapi/httpapi.go` and `httpapi/backlinks_test.go` (409 before purchase, 200 from cache after; fails with the header wiring removed).