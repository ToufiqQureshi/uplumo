# Analytics

Uplumo Analytics is a first-party tracker (`/t.js`) and dashboard for
acquisition, sessions, visible page engagement and selected outcome events.

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
   purchase, call it only after the action succeeds, then select that event
   name in Enquiries as the project's goal. A button click is not a confirmed
   outcome.
4. While a page is visible, cumulative engagement time is upserted every 15
   seconds and on page hide/navigation. Duplicate updates keep the maximum
   duration, so retries do not add time twice. Hidden time is excluded; the
   time up to the moment the page is hidden is kept (`visibleSinceTick` in
   `httpapi/tracker.js`), because `visibilitychange` fires after the page is
   already hidden.
5. The summary returns visitor-days, sessions, engaged sessions, bounce and
   goal rates, acquisition channels, tagged campaigns, entry/exit pages,
   average page/session time, referrers, devices, geography and custom events.
   Landing and exit pages, like campaigns and acquisition, come from the
   whole session: a session that crosses the edge of the range keeps its
   real first and last page.

A tab opened by another page starts with a copy of the opener's
`sessionStorage`; the tracker drops that copied session on the tab's first
page only (`history.length === 1`, not a reload), so reloads and later pages
in the same tab stay one session.

An engaged session has at least 10 seconds of visible page time, two
pageviews, or the selected goal event. Time means the page was visible in the
browser; it does not prove the visitor was looking at it.

Acquisition reads `utm_source`, `utm_medium`, `utm_campaign`, `utm_term` and
`utm_content` (each capped at 100 characters) on non-internal pageviews.
`utm_medium` classifies paid and email traffic. Recognized Google, Microsoft,
LinkedIn and Meta click-ID parameter names can supply a platform label when no
UTM source is present; their values are discarded. This identifies a platform
signal, not a specific campaign or proof that a Meta click was paid. Other
query parameters and raw ad click IDs are not stored. Goal rates connect events
only within the same short session. Do not put personal information in UTM values.

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

## Enquiry evidence and change baselines

`enquiries.go` and `enquiry_handler.go` serve project-protected
`/api/v1/projects/{projectId}/analytics/enquiries` routes (contract:
`backend/api/enquiries.yaml`). Choose one exact custom-event name, such as a
successful form submission, and emit it only after success. No paid providers
or third-party event forwarding are used.

- `go_analytics_enquiry_goals` stores one selected event per project.
  `go_analytics_enquiry_changes` stores at most 100 immutable pre-change
  baselines/project; latest 20 are listed, older entries reappear as newer ones
  are deleted. Both cascade on project deletion.
- Report compares the latest two complete UTC weeks, excludes today, and lists
  the top 50 paths by visitor-days. Daily salted hashes rotate: counts are
  visitor-days, never identifiable people. Goal visitors require a matching
  pageview/hash/path/window; repeated events cannot inflate rates above 100%.
- A drop flag requires at least 30 visitor-days in each week, 5 earlier goal
  visitors, traffic within ±20%, and at least a 50% rate decrease. No observed
  selected event in 28 complete days means verify tracking, never lost leads.
  These rules surface a hypothesis; they cannot prove a broken form or revenue
  loss. Flags are calculated on read and displayed in-app, not pushed externally.
- Saving a change requires a path without query/fragment and a note of 1–500
  characters. A project lock serializes the cap and request-key replay. Baseline
  metrics, goal name and seven-day dates are frozen. Comparison uses the first
  seven complete UTC days after the change date, so today never mixes partial
  traffic with complete days. Later goal changes cannot rewrite earlier evidence.
- All reads/writes require existing session/project middleware; baseline ID and
  project are checked together. Deleting a foreign baseline is a harmless no-op.
  The endpoint never requests a customer URL or runs a paid check.
