# GA4 reports

This package serves read-only Google Analytics 4 reports for a project. The
handler authorizes the session and project first, the service applies report
rules, and the repository reads the selected property and OAuth account from
the existing `ga4_connections` row. Google tokens are always requested for
the connecting user, the `google-analytics` grant, and that grant's account.

The Go report endpoint is `POST /api/v1/projects/{projectId}/ga4/reports/run`;
the organic dashboard endpoint is
`POST /api/v1/projects/{projectId}/ga4/overview/organic`. The report route
returns the existing report JSON shape, including source and date metadata,
rows, pagination, quota, warnings, optional previous-period comparison, and
report-specific diagnostics/activity. Report filters and dimensions stay
explicit because GA4 rejects incompatible combinations and may bill failed
requests.

## Rules and edge cases

- The connection table `ga4_connections` is part of migration version 1. The
  project-access middleware is the authorization boundary for report reads.
- A report defaults to organic search and the last 28 complete days in the
  property's IANA timezone. Future end dates clamp to yesterday in that zone;
  partial, malformed, or reversed date ranges fail before contacting Google.
- Page size is 1–1,000, offset is non-negative, and provider responses are
  checked against the exact requested dimension and metric headers. Invalid
  rows and non-finite values fail closed. Restricted metrics become `null` and
  mark the report as limited.
- Comparison is available only for event key events, channel-group acquisition,
  device audience, and new-versus-returning audience. Complete report reads are
  capped at the same 1,000 rows used by the legacy service; a requested page
  beyond that buffer is fetched separately.
- GA4 API, OAuth grant, quota, and malformed-report failures use stable
  `ga4_*` error codes. The shared Google API client only retains Google's
  allowlisted `SERVICE_DISABLED` reason and numeric `Retry-After` value; it
  never includes provider error text in logs or responses.
- Property listing, connection setup/removal, MCP reports/overview/measurement
  health/Search Opportunities, and their backing services are available in Go.
  The setup UI and the dashboard card call these routes; the report,
  measurement-health and Search Opportunity routes are used by the MCP tools.

## Property setup

`POST /api/v1/projects/{projectId}/ga4/connection/status` reports the selected
property, current user's Google grant, integration-management permission, and
OAuth configuration. `properties/list` lists properties from each of the
user's grants with isolated reconnect/unavailable states. `connection/set`
requires an owner/admin, confirms the grant belongs to the user, verifies the
property from Google's account summaries, then reads canonical property
metadata before saving. `connection/disconnect` has the same role gate.
User-info email lookup is optional. Property discovery is capped at 100 pages of 200 account
summaries.

Deliberate difference: the Go routes do not emit the legacy PostHog
`ga4:property_select` and `ga4:disconnect` events. The setup UI calls these Go
routes now; those two analytics events are not emitted.

## Organic overview

The overview runs a bounded aggregate report for the current range, an equal-
length previous range, and a daily or weekly trend. All three use the same
organic-search filter and property timezone as the report endpoint. The daily
trend is capped at 1,000 rows; a truncated trend carries the legacy
`trend_truncated` warning. Key-event decline diagnostics are suppressed when
any report is limited, or the previous period contains fewer than five events.

## Measurement health

`POST /api/v1/projects/{projectId}/ga4/measurement-health` reads up to 200
data streams, each web stream's enhanced-measurement settings, key events, and
custom definitions. Admin API calls use the connecting user's existing
`google-analytics` grant through the shared bounded Google client. Provider
stream names are checked against the selected property before follow-up calls.

## Search opportunities

`POST /api/v1/projects/{projectId}/ga4/search-opportunities` joins up to 1,000
final GSC page rows with up to 1,000 organic GA4 landing-page rows. It keeps
GSC positions 4 through 20, normalizes host/path keys, scores joined rows by
demand, business value, and reachability, and returns coverage and truncation
metadata. The default range ends three property-local days before today to
match Search Console's final-data lag. Results expose the difference between
the GSC Pacific time zone and the Analytics property time zone as a warning.

## Environment

Uses the existing Postgres connection and Google OAuth credentials. No new
environment variable or database migration is needed for report reads.

## Search growth analyst

`growth.go` joins current/previous final GSC pages with **Google** organic GA4
landing pages, with bounded query evidence. `POST .../ga4/growth/report` is an
explicit analysis action: three GSC reads and two GA4 reads on a standard cache miss,
1,000 rows each (goal/expanded variants below), no paid provider. Identical requests coalesce per process;
Postgres keeps one report per project for ten minutes. Cache identity includes
organization, project, dates, brand terms and both selected connections. Report
ranges are at most 90 days and both comparison windows must fit GSC history.

Findings are review candidates: traffic loss, near-top positions, CTR decline,
performing pages needing reach, key-event-rate gaps and recorded purchase
revenue declines (at least five previous transactions). Business findings need
at least 100 sessions; winner findings also need five key events. Restricted or
limited Analytics suppresses business recommendations. Missing previous rows
or ambiguous Analytics URL variants remain unknown. Query branding and competing
page evidence do not attribute GA4 outcomes to queries. GA4 key events are never
represented as verified leads or sales. Rates compare observed pages, not a
universal industry benchmark or a controlled experiment.

`growth_repository.go` stores server-derived page snapshots and change notes in
`go_search_growth_actions`. All operations include organization and project.
Saving a page is idempotent; a transaction advisory lock serializes the 100-page
project cap. Marking applied preserves the first applied timestamp. Comparisons
wait for 14/28 complete days after the applied day in both source timezones,
then make one exact-page GSC read and one host/path-filtered GA4 read. Baseline
and post-change ranges are explicit; clicks/day normalizes different durations.
This is observational evidence, not proof the edit caused the result. Changed
properties/timezones/currencies reject the comparison; missing rows are unknown.

`growth/devices` is a separate explicit one-read drill-down, filtered to Google
organic. Measurement setup reuses `measurement-health`; saved AI citations reuse
the existing AI Tracking GET endpoints, with no new paid check or schedule.
The SAM/MCP `get_search_growth_report` tool shares the report cache. Search
analyst chat mode executes only allowlisted free data/site-read tools; AI
completions retain SAM's existing credit holds, settlement and turn ceiling.

## Business goals, diagnosis and weekly follow-up

`growth_goal.go` discovers registered event-specific session rates from property
metadata. `goalEvent` uses `sessionKeyEventRate:<event>` without event-filtering
away the session denominator. Raw `keyEvents` stays the all-event count;
`goalRate` is separate. Minimum-success checks use sessions × goal rate as an
estimate, not a measured event count. Unsupported goals fail before report
reads. Cache identity includes goal, comparison, coverage and a schema version;
saved actions preserve their original goal even after settings change.

`comparison: year` shifts both endpoints 364 days to match weekdays. It is not
calendar-year/holiday matching. `expanded: true` explicitly pages up to 5,000
rows/source; standard remains 1,000. A cache miss makes at most 5 standard or
25 expanded report reads, plus 1 metadata read for a selected goal. The 90-second
analysis deadline and source quota/error handling still apply. Reaching a row
bound is visible, and GSC can withhold rows even below it. BigQuery is an optional
external setup link, not an enabled/billed integration.

`growth_clicks.go` investigates a server-saved report page with 2 exact-page
query reads (1,000 rows/period, 60-second overall timeout). It separates low
visibility, lower position and CTR changes; missing query rows remain null.
The UI's separate single-URL inspection reuses GSC's stored-index endpoint.
Neither search averages nor a crawl result establishes why a searcher did not
click. Sitemap reads are implemented by `gsc/sitemaps.go`, not by the GA4 API.

`growth_funnel.go` makes 1 explicit Data API **v1alpha** funnel read, 2–5 distinct
configured events, a closed ordered user funnel, optional device breakdown,
Google organic filtering and a 100-row cap. It preserves sampling/restrictions
and rejects invalid rates. This is user-scoped sequence evidence across the date
range, not landing-page session conversion. Example steps must be verified against
actual site instrumentation. `growth_tracking.go` makes 1 bounded 100-row all-channel
host/source/medium read; self-referral candidates require 20 observed sessions.
Its checklist does not pretend to verify browser tags, cross-domain settings or
consent. Device drill-down uses the selected goal (1 report + optional metadata).

Optional unchanged comparison pages are selected from the server report and
frozen in the saved action's `source.control`. Comparing one adds 1 exact-page
GSC read. The relative click-change ratio is descriptive, not causal: it needs
at least 30 baseline clicks on both pages and 30 after clicks on the comparison
page. Missing rows or inadequate samples yield null. The user must verify that
the comparison page has similar intent/trends and stayed unchanged.

`growth_monitor.go` and migration 32 add opt-in weekly **in-app** checks. Read/save
configuration endpoints are `growth/monitor` and `growth/monitor-save`; enabling
requires a registered goal and stores both property identities. The server checks
one due project/minute, claims atomically across replicas, and uses a revision
fence so disabled/changed configuration cannot receive an old digest. It checks
the current GoogleData entitlement before provider reads. Success waits 7 days;
failure waits 1 day. Changed property identity requires review/re-enable. Reports
use complete 28-day default windows and standard coverage, never expanded reads,
paid AI, SEO providers or email. The last digest holds at most 3 action pages and
5 alerts. An alert requires clicks down >=20% from >=30, goal rate down >30%,
>=100 sessions in each period and >=5 estimated baseline goal-reaching sessions;
limited reports suppress alerts. These thresholds select review candidates, not
statistical significance. Configuration/digest storage is scoped by organization
and project, cascades on deletion, and has RLS with no public-client policy.

Expanded reports retain at most 10 queries/page and 5 example competing URLs/query
with a separate full returned-page count, avoiding quadratic response growth.
The UI browses 10 opportunity cards at a time after the initial top three and
bounds comparison-page choices to the first 100 other report pages.
