# Plan catalog: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Defines official Uplumo prices, entitlements and limits for pricing, billing and every feature gate.

## Decisions to preserve

- Never duplicate price/limits in frontend or feature code. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- Every advertised feature must be enforced, and Razorpay amount/plan ID must match the catalog. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- Plan limit changes need cost/unit-economics review and test coverage. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- **`IndexNow` is a Basic, Pro and Business feature.** _Decision:_ owner's choice (20 Oct); Free does not get it. _Why:_ a paid-plan benefit even though IndexNow costs Uplumo nothing per call. _Preserve:_ the pricing page rows and the `indexnow` routes read `plans.IndexNow`, never a copied plan list. _Trace:_ `catalog.go`, `catalog_test.go`, `billing/plans_test.go`, `backend/internal/indexnow/`.

## Dependencies

Go catalog and Razorpay plan configuration.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Three plans, credit packs and a site add-on (27 Oct 2026)

- **Decision:** sell Basic ₹999 (1 site, 4,000 credits), Pro ₹2,499 (10 sites, 10,000 credits) and Business ₹4,999 (20 sites, 15,000 credits, white-label). Agency is gone from the catalog and `Paid()`. Credit packs (`credits_5000` ₹1,499, `credits_15000` ₹3,999) are a Pro and Business feature (`CreditPacks`); the site add-on (`sites_5`, +5 websites for 30 days, ₹999) is Business only (`SiteAddOn`). Display names follow the owner's words; ids stay `basic`/`pro`/`business`.
- **Why:** the owner wants usage to be paid in credits ("jitna use utne credit"), not by site count, and customers who run out to buy more instead of waiting a month. Agency had no customers. Basic cannot buy packs so it is not a cheaper Pro.
- **Preserve:** one source for prices and limits (this package); a pack never sells a credit below the cheapest plan that may buy it and never costs over a third of its price at the provider; a plan id left out of the catalog unlocks nothing. The Business card's add-on line must equal the pack (`TestBusinessCardStatesTheSiteAddOnPrice`).
- **Trace:** `catalog.go`, `catalog_test.go`; selling and granting in `../billing/topups.go`; the pricing page in `../site/site.go`.
