# Uplumo Analytics: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Collects first-party pageviews/custom events and returns project traffic summaries.

## Decisions to preserve

- Do not store cookies, raw IP addresses, raw session tokens, full URLs or raw ad click IDs. Store only a site-scoped hash of the random tab session; it expires after 30 minutes idle or 24 hours in the browser. _Why:_ This enables same-session engagement and goal attribution without cross-tab or cross-session identity.
- Public ingestion must keep same-origin/domain validation, bot filtering, payload bounds and per-IP/site limits. _Why:_ Explicit bounds prevent a large or abusive request from consuming shared memory, latency or service capacity.
- Custom events must not inflate pageview totals or bypass ingestion checks. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- Campaign reporting stores only bounded `utm_source`, `utm_medium`,
  `utm_campaign`, `utm_term` and `utm_content` values on non-internal
  pageviews. Do not persist other query parameters or raw click IDs.
- Respect Do Not Track and Global Privacy Control before session storage or
  network collection; do not send events to third parties.

## Dependencies

Postgres analytics tables; Redis daily salt/rate-limit counters; no paid SEO provider.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Enquiry interpretation

Use complete equal UTC windows and matched pageview/event visitor-days so clicks
and partial days cannot create spurious conversion lifts. Keep the selected
goal in every immutable change snapshot; changing settings must not change what
an old edit measured. Baselines use the transaction connection while holding the
project lock to avoid pool starvation under concurrent saves.
