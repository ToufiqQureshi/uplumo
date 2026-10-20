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

- **Session reports read whole sessions; visible time includes the last interval.** _Decision:_ landing and exit pages pick the first and last event of every session that has an event in the range, even outside it; the tracker credits visible time up to the hide event and keeps the session across reloads of a tab that has an opener, resetting only on that tab's first page. _Why:_ review found that a session crossing midnight reported the wrong landing/exit page, a page viewed 10–14 s then hidden counted as a bounce, and every reload of a `window.open` tab started a new session. The summary SQL also had a duplicated `FROM` that failed every request. _Preserve:_ the summary runs in one batch; session metrics count only in-range events, while first/last page attribution uses the whole session; hidden time is never counted. _Trace:_ `repository.go` (`summary`), `httpapi/tracker.js`, `TestSummarizeSessionsCampaignsAndLandingPages`, `frontend/scripts/tracker.test.ts`.

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
