# WordPress publishing: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Connects a WordPress site, previews article schema and publishes a saved draft after user confirmation.

## Decisions to preserve

- Publishing is a customer-site side effect: never publish from draft autosave, preview or AI generation. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.
- Retain explicit owner/admin permissions and plan checks. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- Sanitize outbound HTML/schema and do not leak WordPress credentials in logs or errors. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- A timeout after WordPress accepts a post must not cause duplicate posts on blind retry. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- **IndexNow's result is a second toast after a live publish.** _Decision:_ `PublishToSite` shows "Sent to Bing and other IndexNow search engines." or the server's warning when `indexNow` is present, and nothing when it is absent. _Why:_ the customer should see what happened to a side effect they did not click separately; absence means IndexNow is not set up or not on the plan. _Trace:_ `PublishToSite.tsx`, `sitePublishingApi.ts` (Zod `indexNow`), `sitePublishingApi.test.ts`.

## Verification pointers

- `sitePublishingApi.test.ts`
