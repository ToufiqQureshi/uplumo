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

## Retained enquiry measurement data

The enquiry measurement API and UI were retired. Keep migration 00028 and its
tables so deployed customer goals and baselines remain preserved; no active code
reads or writes them. Analytics collection and custom event reporting continue.

## Daily browser recognition and click destinations

Decision: optional `data-recognition="true"` adds a 180-day, non-renewing, site-keyed first-party localStorage ID. The server stores only a site-scoped hash and its first observed timestamp. Default snippets remain anonymous. Daily UTC new browsers were first observed on that day; returning browsers were first observed on an earlier day. Each browser occurs in only one daily category, even across ten sessions/tabs. These counts are not GA4's potentially overlapping session-based user metrics. Unrecognized pageviews report coverage and must never be added to recognized browser counts as unique people. Browser expiry, storage clearing and device changes can reset recognition; no cross-device identity or retroactive classification.

Why: the owner wants daily audience composition and clicked destinations before larger analytics features, without GA4 or payment API dependencies. Existing salted visitor-days cannot recognize a browser across days. Historical browser hashes/first-seen data remain with project history; 180 days describes browser-ID lifetime, not automatic database erasure. Site/project/account deletion cascades browser records. No raw IP, raw recognition ID or external-provider call is added.

Preserve: enable recognition only after required consent. The host consent manager can revoke with `window.uplumoConsent(false)`, which clears identifiers and stops collection on the origin; no injected UI. Secure storage and Web Locks are required for recognition; unsupported browsers remain unrecognized. Concurrent same-origin initialization uses Web Locks, while the database primary key makes first-seen registration atomic. Separate origins/subdomains remain separate browsers.

Click capture stores a bounded event label, source path and normalized HTTP(S) destination host/path. No destination query, fragment, credentials or ad ID is retained. Configured outbound links emit one click, and unconfigured outbound links use `outbound_click`; JavaScript buttons may declare `data-uplumo-destination`. Other custom events retain their semantics. Observed arrivals count clicks with a different pageview at the same host/path and tab session within 30 minutes, possibly just beyond the report boundary. This is temporal association, not proof the click caused navigation; external activity, new-tab arrivals and old events stay unknown. Reporting is paged (1–100 rows, page 1–1000), at most 366 UTC days, with a 10-second read timeout; tenant middleware remains mandatory. The collection body is bounded at 8 KiB to accommodate the three bounded URL fields.

Trace: `journey.go`, `repository.go`, `handler.go`, `httpapi/tracker.js`, migration 35, `journey_test.go`, mounted HTTP tests and frontend `scripts/tracker.test.ts`. RLS/no public policies and public/client grant revocation protect browser registry and the existing event tables now carrying these fields. Frontend query caching uses project/date keys and a 60-second stale window; there is no new server cache or paid provider. Verification and rollout details: `AUDIENCE_DESTINATIONS.txt`.

## Accurate sources, enquiries and contact tracking

**Decision:** traffic sources follow GA4's default channel rules
(support.google.com/analytics/answer/9756891) plus fixes GA4 needs manual
setup for, and the tracker records contact intent without tagging.

- A search brand counts as search only on its search front ends
  (`searchPrefixes`, two-letter country labels): `accounts.google.com` and
  `docs.google.com` are not searches. Webmail hosts (`mail.google.com`,
  Outlook, Yahoo/Zoho/Proton mail) are email.
- Sign-in and payment hosts (`passThroughDomains`: Google/Microsoft/Apple
  sign-in, Razorpay, Stripe checkout, PayPal, PayU, Cashfree, Paytm, PhonePe,
  CCAvenue, Instamojo, BillDesk) are internal navigation, like GA4's
  unwanted-referral list but automatic.
- Known Android app referrers (`appChannels`: Gmail, Google app, Facebook,
  Instagram, LinkedIn, WhatsApp...) take the app's channel instead of direct.
- `utm_medium` uses GA4's rules (`mediumChannel`): paid = contains `cp`, ppc,
  retargeting, `paid*`, display formats, affiliate, sponsored; email in medium
  or source; social and organic mediums.
- Acquisition counts every session: a session whose first page has an
  internal referrer (a tab opened from the site without the copied session)
  is direct, as GA4 treats self-referrals, so source shares add up to sessions.
- Enquiries are classified server-side (`isEnquiry`: whole words plus safe
  substrings, negative words such as `start`/`view`/`error`). The summary
  returns `enquiries {count, visitors, actions}`; `visitors` is distinct
  visitor-days across all enquiry actions, so the enquiry rate cannot double
  count or pass 100%. Sign-up, checkout and purchase events are sales steps,
  not enquiries (owner to confirm).
- The tracker sends `call_click`, `sms_click`, `email_click` and
  `whatsapp_click` (tel:, sms:, mailto:, whatsapp:, wa.me,
  api/web.whatsapp.com, whatsapp.com/send) with no destination, so numbers and
  addresses never leave the browser; `form_submit` on non-search forms (no
  field values; skipped when the form has a tagged button or
  `data-uplumo-ignore`); `file_download` for GA4's file extensions. An owner's
  `data-uplumo-event` tag still names the click. `?uplumo_ignore=1` stores an
  opt-out in localStorage so owners can exclude their own browser.
- `/collect` logs one warning when the client IP is private or in
  100.64.0.0/10 while `X-Forwarded-For` is present: `TRUSTED_PROXY_CIDRS` is
  missing.

**Why:** a live review of uplumo.com (2026-10-27) found Google sign-in
returns counted as "Search engines" (4 of 6), signup/checkout clicks shown as
WhatsApp/call enquiries by a substring regex that also matched `facebook`,
`platform` and `hotel`, source shares computed over 63 of 104 visitors, and
every stored pageview without a country while visitors (115) exceeded sessions
(83) with only 6 session-less pageviews: the server saw Railway's proxy
address, not visitors'. No migration: contact clicks keep type `unknown` and
the UI labels them by name, because 00045 is taken by an open branch and goose
refuses out-of-order migrations.

**Preserve:** no phone number, email address or form value is ever sent;
search forms are never enquiries; one visitor's several enquiry actions count
once in `visitors`; acquisition rows sum to `sessions`; pass-through and
webmail lists only grow with evidence. No paid provider is involved.

**Trace:** `classify.go`, `enquiry.go`, `repository.go` (`summary`,
`enquiryNames`), `handler.go` (`behindUntrustedProxy`),
`httpapi/tracker.js`; tests `classify_test.go`, `enquiry_test.go`,
`TestSummarizeEnquiriesCountVisitorsOnce`, `TestAcquisitionCountsEverySession`,
`TestCollectWarnsOnceAboutAnUntrustedProxy`, `frontend/scripts/tracker.test.ts`.
Production still needs `TRUSTED_PROXY_CIDRS=100.64.0.0/10` (owner action);
old rows keep their stored channel and no country.
