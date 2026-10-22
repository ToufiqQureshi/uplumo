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
