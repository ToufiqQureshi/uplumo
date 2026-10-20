# Uplumo Analytics: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when behavior or an invariant changes.

## Why the feature is structured this way

A first-party tracker and project analytics dashboard with short tab sessions.

## Decisions to preserve

- Do not add cookies or persist raw IP addresses or session tokens. The browser uses a random tab-scoped ID with 30-minute inactivity and 24-hour maximum expiry; the server stores only a site-scoped hash. Visitor-day IDs remain daily salted hashes. This links engagement and goals within a short session without identifying a person or following a browser across tabs or days.
- Collection is public by design, so preserve bot/domain checks, payload bounds, and shared per-IP/per-site rate limits.
- Custom events must use the same validation and rate limits as pageviews; deduplicate retries without suppressing separate user actions.
- Respect Do Not Track and Global Privacy Control before storage or collection.
- Do not forward events to advertising or analytics vendors. This feature does not call paid SEO providers.

## Verification pointers

- `analyticsApi.test.ts`
- `presentation.test.ts`
