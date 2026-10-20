# Public website, pricing and blog: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Serves Uplumo marketing pages, pricing, blog posts, robots.txt, sitemap.xml and static assets from Go.

## Decisions to preserve

- Do not publish malformed/unreviewed content or expose drafts in sitemap. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.
- Escape post fields and metadata; keep canonical URLs correct and URLs derived from trusted config. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- Pricing must be sourced from the current plan catalog. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- **The IndexNow pricing rows say it is not Google.** _Decision:_ both feature lists read `plans.IndexNow` and name Bing and other engines. _Why:_ Google does not use IndexNow; a pricing page that implied faster Google indexing would promise what the product does not do. _Trace:_ `site.go` (tools list and comparison rows), `site_test.go`.

## Owner website measurement

Use one shared `static/analytics.js` loader from `templates/base.html` and
`frontend/index.html` for GA4 `G-NTP32DNMN2` and Uplumo site
`DDVAYTG2IADAYZQSEM4GMU6HFA`, supplied by the owner. This covers both Go public
pages and the React app without duplicating initialization in components. Load
once per document, only on uplumo.com/www; skip DNT/GPC and authentication routes
so development traffic and credential-link pages are excluded. The public CSP
allows self-hosted scripts plus Google's explicit analytics hosts; do not enable
arbitrary scripts or inline execution. Privacy copy discloses GA4 cookies and
keeps website measurement separate from connected workspace Google grants.

Named `data-uplumo-event` attributes distinguish `signup_click`, `pricing_open`,
`plan_select_<plan>` and app `subscription_checkout_<plan>` intent. The shared
loader sends these names to GA4; the existing Uplumo tracker collects them
independently with session/source attribution. Do not include form values,
emails, or payment details in event names, and never present an intent click as
a completed subscription. Successful billing still requires server confirmation;
no purchase conversion or person-level identity tracking is added here.

The React router emits `uplumo:navigation` after a route resolves. The loader
rechecks eligibility on that event, so an initial sign-in page can leave tracking
off while a subsequent in-app dashboard/checkout route initializes it once,
without requiring a reload. Loading auth first must not permanently suppress
subscription-intent measurements. `owner-tracking.test.ts` covers this transition.

Trace: `templates/base.html`, `landing.html`, `pricing.html`, `privacy.html`,
`site.go`, `static/analytics.js`; frontend `scripts/owner-tracking.test.ts` tests
single initialization, opt-outs, auth exclusions and named clicks. Site tests
verify rendered assets/CSP. Owner explicitly requested a direct main push without
a PR for these tags; that exception does not change the normal PR workflow.

## Dependencies

Embedded/site templates, markdown blog content, plan catalog, static assets.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Optional recognition disclosure

Decision: privacy copy and tracker FAQ/blog disclose optional 180-day first-party browser recognition, default anonymous mode, first-seen/hash history retention until project/account deletion, host-managed consent and sanitized click destinations. Remove blanket claims that no visitor can be followed across days or that storage-based analytics never needs consent.

Why: optional recognition changes the privacy behavior of enabled customer snippets. Existing owner tracking remains on its current default configuration; this change does not enable recognition there or inject a consent UI. Preserve accurate distinctions between browser identity and people, click intent and success, browser expiry and historical database deletion. Trace: `site.go`, `templates/privacy.html`, `blog/chatgpt-traffic-google-analytics.md`, site tests and analytics feature handover.
