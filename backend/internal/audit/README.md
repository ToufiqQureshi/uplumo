# audit

Site audit: crawl a site, run per-page and cross-page checks, optionally sample
Lighthouse through Google's PageSpeed Insights API, and serve the results to the
React audit pages.
Ported from the TypeScript engine; the results contract keeps its existing
fields and adds explicit coverage information for newer checks.

## Flow

1. `POST /projects/{id}/audit/start` validates the URL (SSRF guard), checks plan
   limits and audit capacity, inserts a `go_audits` row, and enqueues a job on the
   `jobs` queue (`AuditQueueName`).
2. The worker started in `cmd/server/main.go` claims the job and the `Runner`
   runs it: discovery (robots.txt, sitemaps), the crawl, the checks, then the
   Lighthouse phase. Progress goes to Redis (live feed) and `go_audits` (counters).
3. The pages poll `status`, `progress` and `results`. Lighthouse detail pages
   use `lighthouse/issues` and `lighthouse/export`.

Jobs are at-least-once. Row ids are deterministic, so a retried job rewrites the
same rows instead of duplicating them.

## Endpoints

All are `POST /api/v1/projects/{projectId}/audit/<name>`, need a session and
membership of the project's organization, and take a JSON body (unknown fields
are rejected): `start`, `status`, `results`, `history`, `progress`, `delete`,
`capabilities`, `lighthouse/issues`, `lighthouse/export`. Spec:
`backend/api/audit.yaml`.

## Tables

`go_audits`, `go_audit_pages`, `go_audit_issues`, `go_audit_lighthouse_results`
(migration `00007_audit.sql`). Pages, issues and results cascade with the audit.
`go_audit_pages.citation_signals_json` (migration 47) holds each page's AI
citation signals (schema types, question headings, author, dates, lead
paragraph length, lists, tables, quotes, statistics and versioned citation
checks) for Cite Fix (`backend/internal/citefix`). NULL or an older version
means these newer checks were not measured; results ask for a rescan.

The audit history view also compares the two latest completed crawls with the
same start URL. It reports changes in title, meta description, canonical URL,
indexability, HTTP status, page coverage, and issue count. It fetches saved
results only after a user asks to compare; a URL missing from a later crawl is
shown as missing from that crawl, not assumed to have been deleted from the site.

## Orphan-link suggestions

For orphan-page issues, the result view suggests up to three indexable source
pages from the same crawl by matching meaningful title and URL terms. It
excludes other detected orphan pages, makes no provider calls, and offers a
safely escaped link snippet for review and manual placement.

## Env vars

- **JavaScript rendering:** the wired DataForSEO On-Page renderer is available
  only when the provider client is configured. Capabilities then return true,
  with a 20-page limit and a 39-credit maximum. The customer must opt in at
  audit launch; after the ordinary crawl flags app shells, only those pages
  are sent to DataForSEO. Rendered HTML goes through the same `AnalyzeHTML`
  and page check path, refreshing `citation_signals_json`. Rival pages never
  use this paid renderer. `AUDIT_BROWSER_RENDERING` and `CONTEXT_API_KEY` do
  not control the wired provider.
- DataForSEO publishes JavaScript rendering at $0.0015/page. The spend guard
  holds up to 39 credits for 20 shells and settles the actual cost; unusable or
  failed render results charge 0 credits. No paid POST is automatically
  retried. URL claims are unique per audit, so a replay cannot buy the same
  URL again. DataForSEO allows at most 20 tasks and five identical registrable
  domains in each `instant_pages` request. Twenty same-domain pages use four
  paid task POSTs, each followed by one free `raw_html` retrieval POST for that
  batch's successful task IDs (available for seven days). A later batch that
  fails stops new batches but keeps, and charges only, the pages already
  delivered.
- `PAGESPEED_API_KEY` (a Google Cloud API key with the PageSpeed Insights API
  enabled) is needed for Lighthouse. Without it audits still run; Lighthouse is
  skipped.

## Lighthouse through PageSpeed Insights

Google runs Lighthouse on its own machines, so a check costs no money and no
credits. Running Lighthouse ourselves would need Chrome in the image and about
half a gigabyte of memory per check on the shared server. DataForSEO ran the
same Lighthouse for a fee.

- The one cost is quota: 25,000 checks a day for the whole Google Cloud project,
  shared by every workspace. So each workspace may run
  `PageSpeedDailyChecksPerOrganization` (200) checks per UTC day, counted in
  Redis. Over that, the check fails with a message saying when it resets; the
  rest of the audit still runs. If Redis is down the check fails closed.
- A free plan's "auto" becomes "homepage": 2 checks (mobile and desktop). Paid
  plans sample the homepage plus one page per URL template, up to 10 pages, 20
  checks.
- Real-visitor (field) data comes from the Chrome UX Report API when
  `CRUX_API_KEY` is set, otherwise from the copy PageSpeed Insights still embeds
  (Google has said PSI will stop embedding it; the Lighthouse API stays). See
  "Field data" below.
- The key goes in the `X-Goog-Api-Key` header, never the URL: a transport error
  repeats the URL, and that message is stored on the audit for the customer.
- A 429 (per-minute quota, when many audits run at once) or 503 is retried
  once after 5 seconds. Every request, the retry included, takes its own slot
  of the workspace's daily share; a retry without a slot is not sent. A 500
  is not retried.
- Time: the crawl has a 30-minute deadline; finalizing and the Lighthouse
  checks then run on the job's clock, the checks with their own 12-minute
  budget (20 checks take about 10). A check is not started with less than 30
  seconds left; it is recorded as skipped. Finished checks are saved on a
  separate 30-second clock, so a slow provider never loses them or fails the
  audit. Crawl + checks stay inside the job queue's 45-minute timeout.

## Field data

Stored per check as `fieldData` with a `status`:

| Status              | Meaning                                                                                                                              |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `available`         | Metrics returned, with `source` (`crux`/`pagespeed`), `scope` (`page`/`origin`), `coreWebVitals` and, from CrUX, `collectionPeriod`. |
| `insufficient_data` | CrUX answered `404 NOT_FOUND` for the URL and then for the origin. The only status that says the site lacks traffic.                 |
| `unavailable`       | The lookup failed: key rejected, 429, timeout, outage, malformed answer. Says nothing about traffic.                                 |
| `not_configured`    | No `CRUX_API_KEY` and PSI embedded nothing.                                                                                          |

A report with no `fieldData` was saved before it was recorded: "not recorded",
never "Google has no data".

- CrUX lookup: `POST chromeuxreport.googleapis.com/v1/records:queryRecord` with
  `formFactor` `PHONE` (mobile check) or `DESKTOP`, first `url`, then `origin`
  only if the URL got Google's `404 NOT_FOUND`. A 404 without that body (a
  proxy) is a failure. Failures never fall back to the origin.
- Units: CrUX sends CLS p75 as a decimal string ("0.05"); PSI's embedded copy
  sends it times 100. Each source is parsed on its own terms. Categories come
  from Google's published thresholds.
- Core Web Vitals: passed when LCP, CLS and (if measured) INP are good at p75;
  `not_assessed` when LCP or CLS is missing, as PageSpeed Insights does.
- PSI's embedded copy: a page block with `origin_fallback: true` is the whole
  site's data and is stored with `scope: origin`.
- Cache: answers (records and "no record") are cached in Redis for 12 hours
  per form factor and URL or origin. Google recomputes the 28-day window once
  a day, around 04:00 UTC, so this changes at most one day of 28. Failures are
  not cached. Identical concurrent lookups on an instance share one request.
- Quota: CrUX allows 150 queries a minute per Google Cloud project, free and
  not raisable. An audit makes at most 40 lookups (20 checks, URL then
  origin), spread over the minutes its PSI checks take, and the cache absorbs
  repeats. Past the limit a lookup is `unavailable` and the report keeps PSI's
  embedded copy if there is one.
- Setup: enable "Chrome UX Report API" on a Google Cloud API key, restrict the
  key to that API (or to it and PageSpeed Insights if one key serves both) and
  set `CRUX_API_KEY`. The key travels in `X-Goog-Api-Key`, never the URL.
- A page Google could not load (`NO_FCP`, `ERRORED_DOCUMENT_REQUEST`) is the
  customer's site, so it is logged as a warning, not an error.

## Rules that must never break

- Every URL the crawler fetches goes through the `Guard` (private, loopback and
  link-local addresses are blocked).
- Every read and delete is scoped by `project_id`. A result from another
  project is "not found", never "forbidden".
- One workspace's page speed checks never use up another's: the daily cap is
  per organization.
- A free plan crawls at most `FreeMaxAuditPages`; paid at most `PaidMaxAuditPages`.

## Deliberate differences from the TypeScript version

- Lighthouse payloads live in Postgres (`payload_json`), not R2. Results list
  rows carry `hasPayload` instead of `r2Key`, and the list query does not load the
  payload.
- Issue details are returned as a parsed `details` object; the React client
  turns it back into the `detailsJson` string its components read.
- Audits are scheduled through the `jobs` queue, not Cloudflare Workflows.

## Edge cases

- A stored payload that is valid JSON but not a version-2 payload (source
  `pagespeed-insights`, or `dataforseo-lighthouse` from before the switch) is
  treated as "no issue details", not an error. Text that is not JSON is a 500.
- The full Lighthouse export returns the stored text untouched.
- Missing scores sort last (treated as 100) when ranking issues.

## Tests

`go test ./internal/audit/` needs no database for unit and handler tests.
Repository tests need `TEST_DATABASE_URL` and run in CI. To prove a test can
fail, break the code by hand and rerun it (`CLAUDE.md` section 4).

## Technical and external checks

`technicalsignals.go`, `technicalmultipage.go` and `pagereporters.go` inspect
language, viewport, duplicate/relative canonicals and metadata, sitemap
indexability, hreflang, insecure links/forms, nofollow, image dimensions and
HSTS. Canonical and hreflang target checks compare pages actually crawled;
unfetched targets are unknown. Search and training crawler lists live together
in `aiaccess.go`; training-only blocks are informational and do not lower AI
readiness scores.

`externalchecks.go` queues up to 200 external links and 30 images on workers
separate from the audit worker. Requests use the audit SSRF guard, robots.txt,
HEAD for each target, a 5-second request timeout, a shared Redis one-second per-host
interval, 64 KiB robots limit and a 25-second batch budget. Images above
500,000 bytes are flagged when a usable image Content-Length is provided.
Redirects and inaccessible or robots-blocked targets remain unknown. A queued
or incomplete issue shows coverage; a fenced start marker prevents a retry from
repeating HEAD requests. A short workspace claim transaction releases its DB
connection before network waits; unavailable Redis leaves coverage partial
rather than bypassing the host limit. `checksNeedRescan` flags old audit signals.
