# Google integrations: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Connects Google accounts, selects GA4 and Search Console properties, and disconnects account grants.

## Decisions to preserve

- OAuth state, redirect allowlists, token encryption and refresh-once behavior protect account access. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- Request only the read-only scopes the product needs; never expose access/refresh tokens to React. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- Property listing and reporting consume Google quota, so use bounded requests and avoid refetch loops. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Account removal is destructive; preview impact and scope it to the owner. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.

## Verification pointers

- `ga4ConnectionsApi.test.ts`
- `googleLinkError.test.ts`

## GA4 picker says why properties are missing

- **Decision:** the property picker shows "Google did not give Uplumo
  permission to read Analytics… Reconnect and tick the Google Analytics box"
  with a Reconnect button for `missing_permission`, a contact-support line for
  `api_disabled`, and the old "Couldn't load properties / Try again" otherwise.
- **Why:** owner could not connect GA4 and the picker gave no reason.
- **Preserve:** the reasons come from the API (`unavailableReason`); the UI
  never promises an alert that does not exist.
- **Trace:** `GooglePropertyPicker.tsx`, `ga4ConnectionsApi.ts`,
  `googleProviders.ts`; backend `backend/internal/ga4/decision.md`.
