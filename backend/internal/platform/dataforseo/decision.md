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

## Charge billed "no search results" (40501) instead of refunding it

- **Decision:** `responseSuccessfulCost` now treats a task with status 40501
  ("No Search Results") as billable, alongside successful (2xxxx) tasks and
  status 0. Other 4xxxx/5xxxx task errors still refund the customer and are
  absorbed by Uplumo.
- **Why:** 40501 is not a provider failure — it is a valid, billed empty answer
  (a keyword that genuinely has no SERP, so the customer learns "not ranking").
  Rank checks and keyword SERPs delivered that answer for free while Uplumo paid
  the provider on every such lookup, a systematic undercharge on a common path.
- **Preserve:** Genuine provider-billed failures keep refunding the customer;
  provider cost is still recorded; the queued rank-check path (which charges at
  post time) is unchanged; no new provider call or limit/price change.
- **Trace:** `client.go` (`responseSuccessfulCost`, `billableTask`),
  `client_test.go` (`TestResponseSuccessfulCostChargesSuccessfulAndNoResultTasks`,
  `TestDoChargesForNoSearchResults`, and the two refund tests now use a real
  failure code 40600 instead of 40501).

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
