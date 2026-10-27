# Analytics

Uplumo Analytics is a first-party tracker (`/t.js`) and dashboard for
acquisition, sessions, visible page engagement and custom events.

## Flow

1. The Analytics page calls `POST /api/v1/projects/{id}/analytics/site`,
   which creates the project's site key once and returns the snippet:
   `<script defer data-site="KEY" src="https://uplumo.com/t.js"></script>`.
2. The tracker posts `{k, u, r, w, s, p}` to `POST /collect` for each
   pageview, including single-page-app navigation. `s` is a random tab
   session ID and `p` deduplicates the pageview and its engagement updates.
3. **Custom events.** Any element with `data-uplumo-event="Name"` is tracked
   when clicked, with no code (`<button data-uplumo-event="Signup">`). Code
   can call `window.uplumo("Name")`. To count a successful form, booking or
  purchase, call it only after the action succeeds. A button click alone does
  not confirm a completed outcome.
   Without any tag the tracker also records `call_click`, `sms_click`,
   `email_click` and `whatsapp_click` (tel:, sms:, mailto: and WhatsApp
   links, never the number or address), `form_submit` for non-search forms
   (never field values; `data-uplumo-ignore` skips a form), `file_download`
   and `outbound_click`. `?uplumo_ignore=1` on any page excludes that browser
   (`=0` undoes it).
4. While a page is visible, cumulative engagement time is upserted every 15
   seconds and on page hide/navigation. Duplicate updates keep the maximum
   duration, so retries do not add time twice. Hidden time is excluded; the
   time up to the moment the page is hidden is kept (`visibleSinceTick` in
   `httpapi/tracker.js`), because `visibilitychange` fires after the page is
   already hidden.
5. The summary returns visitor-days, sessions, engaged sessions, bounce rate,
  acquisition channels, tagged campaigns, entry/exit pages,
   average page/session time, referrers, devices, geography and custom events.
   Landing and exit pages, like campaigns and acquisition, come from the
   whole session: a session that crosses the edge of the range keeps its
   real first and last page.

A tab opened by another page starts with a copy of the opener's
`sessionStorage`; the tracker drops that copied session on the tab's first
page only (`history.length === 1`, not a reload), so reloads and later pages
in the same tab stay one session.

Sources follow GA4's default channel rules (`classify.go`): search only on a
search engine's search hosts, webmail as email, sign-in and payment hosts as
internal, Android app referrers by app. Every session gets exactly one
source (an internal first page is direct). Enquiries (`enquiry.go`) are
WhatsApp, call, SMS, email and form events, counted per distinct visitor-day.
Behind a proxy, set `TRUSTED_PROXY_CIDRS` (Railway: `100.64.0.0/10`) or
visitors, countries and rate limits use the proxy's address; `/collect` logs
a warning once when that happens.

An engaged session has at least 10 seconds of visible page time or two
pageviews. Time means the page was visible in the browser; it does not prove
the visitor was looking at it.

Acquisition reads `utm_source`, `utm_medium`, `utm_campaign`, `utm_term` and
`utm_content` (each capped at 100 characters) on non-internal pageviews.
`utm_medium` classifies paid and email traffic. Recognized Google, Microsoft,
LinkedIn and Meta click-ID parameter names can supply a platform label when no
UTM source is present; their values are discarded. This identifies a platform
signal, not a specific campaign or proof that a Meta click was paid. Other
query parameters and raw ad click IDs are not stored. Do not put personal
information in UTM values.

## Tables

- `go_analytics_sites`: one site key per project.
- `go_analytics_events`: one row per pageview.
- `go_analytics_custom_events`: one row per custom event. Kept apart so
  events never inflate pageviews or visitors.
- Pageview/event IDs are random 128-bit browser values; only site-scoped hashes
  are stored. Session tokens stay in tab-scoped `sessionStorage`, expire after
  30 minutes idle or 24 hours, and are not shared across tabs.
- Daily anonymous visitor counts remain visitor-days. Events from previously
  installed tracker versions keep counting as pageviews but have no session
  duration or session-attribution fields.

## Rules that must never break

- No cookies, raw IPs, raw session tokens, full URLs or raw click IDs are
  stored. Visitor IDs use a daily Redis salt; sessions use site-scoped hashes.
- Store only five bounded UTM fields for campaign reporting.
- Respect Do Not Track and Global Privacy Control before any storage or hit.
- `/collect` is open to every origin and needs no CORS preflight (text/plain
  body). It drops bots, refuses a page outside the project's domain
  (`foreign_host`) and rate-limits per IP and site.
- A custom event goes through the same bot, domain and rate checks as a
  pageview.
- Session/pageview/event IDs are validated; pageviews and events are
  idempotent across browser retries, and duration is capped at 24 hours.

## Production notes

If the Analytics page shows no visits, check the Railway HTTP logs for
`/t.js` and `/collect`: no requests means the snippet is not on the site.

## Retained historical data

The enquiry measurement UI and API have been removed. Migration 00028 is kept
in migration history, so existing goal and baseline rows are not dropped. New
code does not read or write those legacy tables; normal Analytics event data
continues to be collected and reported.
## Daily recognition and click destinations

Optional browser recognition (`data-recognition="true"`) uses a 180-day site-keyed localStorage ID after required consent, with only its site-scoped hash/first-seen timestamp stored server-side. Default snippets retain anonymous visitor-days. Daily new/returning recognized browsers and unrecognized pageview coverage are separate measures; old visits cannot be classified. See backend analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt` for precise counting, consent, retention and verification.

`GET /api/v1/projects/{projectId}/analytics/summary?from&to` adds `audience` rows: `date`, `newBrowsers`, `returningBrowsers`, `unknownPageviews`. `GET /api/v1/projects/{projectId}/analytics/clicks?from&to&page&limit` returns `rows` (`page`, `name`, `host`, `path`, `type`, `clicks`, `visitorDays`, `arrivals`) and `hasMore`. Both require session/project access. Dates are inclusive UTC days, at most 366. Click pages are 1–100 rows, page 1–1000; UI uses 20 rows and project/date-keyed one-minute TanStack caching. No new server cache or paid call.

Collector additions are optional `b` (32 lowercase-hex browser ID), `c` (click boolean), `x` (absolute HTTP(S) destination without credentials). Clicks require event/session/pageview IDs. Configured clicks and automatic outbound links share the same deduplicated event record. Destination query strings/fragments are stripped; JavaScript buttons can specify `data-uplumo-destination`. Internal arrivals are matching later same-tab pageviews within 30 minutes, not proof of causation. External subsequent activity is unavailable.
