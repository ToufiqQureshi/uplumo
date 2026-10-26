# Backlink research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Why the feature is structured this way

Shows domain/page backlink summaries, referring domains and top linked pages.

## Decisions to preserve

- **Backlink lookups are paid: validation and plan checks must happen before provider spend** _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- **A subfolder or exact-page scope may not support every aggregate** — preserve backend scope notes and null values. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- **Never silently retry paid POST requests or present missing provider data as zero** _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Verification pointers

- `backlinksFilterTypes.test.ts`

## Show the complete paid link profile

- **Decision:** Show every returned link-type and link-attribute bucket, including the provider's unclassified bucket as “Other”.
- **Why:** The summary already paid for these categories; hiding entries beyond six drops customer data.
- **Preserve:** Keep the existing sort and total; do not turn missing categories into zero.
- **Trace:** `BacklinksProfilePanel.tsx` and backend summary normalization.

## Saved links load from cache; Search and Retry buy

- **Decision:** Report queries send `cachedOnly` (header `X-Cached-Only`) unless the customer clicked Search or Retry for that report in this browser session (`client/lib/paidIntent.ts`). An expired cache then shows the server's "refresh" message with Retry, and Retry is the purchase.
- **Why:** Opening a saved link after the cache expired bought the report again without a click (owner, 26 Oct).
- **Preserve:** `cachedOnly` is read at request time, so the refetch a Retry starts is allowed to buy; it never goes in the request body. Paging, sorting and tabs after a Search in the same session still load normally.
- **Trace:** `backend/internal/platform/dataforseo/decision.md`; tests in `client/lib/uplumoApi.test.ts`, `backlinksApi.test.ts`, `domain/domain-cache.test.ts`.
