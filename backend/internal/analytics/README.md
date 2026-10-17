# Analytics

Uplumo Analytics: a tiny cookie-free tracker (`/t.js`, source in
`internal/httpapi/tracker.js`) that counts pageviews and custom events for a
project's site, and the summary the Analytics page shows.

## Flow

1. The Analytics page calls `POST /api/v1/projects/{id}/analytics/site`,
   which creates the project's site key once and returns the snippet:
   `<script defer data-site="KEY" src="https://uplumo.com/t.js"></script>`.
2. The tracker posts `{k, u, r, w}` to `POST /collect` for each pageview,
   including single-page-app navigation.
3. **Custom events.** Any element with `data-uplumo-event="Name"` is tracked
   when clicked, with no code (`<button data-uplumo-event="Signup">`). Code
   can call `window.uplumo("Name")`. The tracker posts the same body plus
   `n` (the event name, trimmed, 1 to 64 characters).
4. `GET .../analytics/summary` returns visitors, pageviews, pages, channels,
   AI sources, referrers, devices and `events` (custom events by count and
   visitors, top 50).

## Tables

- `go_analytics_sites`: one site key per project.
- `go_analytics_events`: one row per pageview.
- `go_analytics_custom_events`: one row per custom event. Kept apart so
  events never inflate pageviews or visitors.

## Rules that must never break

- No cookies, no stored IP. The visitor id is SHA-256 of a daily salt that
  lives only in Redis, the site key, the IP and the user agent; it cannot be
  followed across days.
- `/collect` is open to every origin and needs no CORS preflight (text/plain
  body). It drops bots, refuses a page outside the project's domain
  (`foreign_host`) and rate-limits per IP and site.
- A custom event goes through the same bot, domain and rate checks as a
  pageview.

## Production notes

If the Analytics page shows no visits, check the Railway HTTP logs for
`/t.js` and `/collect`: no requests means the snippet is not on the site.

## Enquiry evidence and change baselines

`enquiries.go` and `enquiry_handler.go` serve project-protected
`/api/v1/projects/{projectId}/analytics/enquiries` routes (contract:
`backend/api/enquiries.yaml`). Choose one exact custom-event name, such as a
WhatsApp click or successful form submission. Existing ingestion is unchanged;
a click is not a confirmed enquiry. No paid providers, cookies or new visitor
identifiers are used.

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
