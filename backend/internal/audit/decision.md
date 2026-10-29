# Site audit: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Runs an asynchronous bounded crawl, technical checks and optional Lighthouse/CrUX collection.

## Decisions to preserve

- Every initial and redirected URL passes SSRF checks. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- Per-organization crawl and PSI quotas, timeouts and page caps protect shared capacity. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- At-least-once jobs use idempotent row IDs; provider outages remain unavailable/partial, not false zeroes. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

Postgres audit tables; Redis progress/cache/quota; PageSpeed Insights and CrUX; jobs queue.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Opt-in JavaScript rendering for app shells (owner approved 28 Oct 2026)

- **Decision:** Capabilities advertise JavaScript rendering only when the
  DataForSEO On-Page renderer is actually wired. Site Audit offers an opt-in
  checkbox, sends only pages flagged as app shells by the initial crawl, and
  caps a run at 20 pages. Rendered HTML goes through the same analyzer and
  replaces the page's shell signals/issues. Price is $0.0015 per page (max 39
  credits for 20 pages, actual cost charged); failed/unusable rendering is 0
  customer credits. DataForSEO `instant_pages` requests are capped at 20 tasks
  and five per registrable domain; raw HTML is retrieved once for successful
  IDs through the free seven-day `raw_html` endpoint.
- **Why:** React/Next app shells otherwise fail only the `pages-readable` Cite
  Fix check even though the browser can display their content. The owner
  approved the actual published price and a concrete 50-page-site scenario
  before enabling this paid path.
- **Preserve:** Explicit opt-in; render only crawler-detected shells; max 20
  URLs/audit; spend hold before the paid call; actual price ≤ displayed max;
  call with `retry:false`; unique per-audit URL claim so an ambiguous worker
  restart never repeats the purchase. No rival page is rendered. Environment
  flags alone cannot advertise capability. Keep normal HTTP crawl and SSRF
  checks intact.
- **Trace:** `dataforseo_renderer.go`, `rendering.go`, `runner.go`, `service.go`,
  `repository.go`, migration `00049_citefix_evidence.sql`,
  `frontend/tests/audit-capabilities.spec.ts`. The fake provider asserts exact
  counts: 20 same-domain pages make four paid task POSTs, one free HTML POST,
  and one $0.03 hold/actual delivery; failed, 500, timeout and garbage
  responses dispatch once without automatic retry. Mutating the 20-page guard
  to allow 21 or enabling POST retries makes its focused test fail.

## AI access checks: blocked AI crawlers and llms.txt

- **Decision:** Every audit adds two site-wide issues when they apply. `ai-crawlers-blocked` (warning) lists which AI crawlers (OAI-SearchBot, ChatGPT-User, GPTBot, PerplexityBot, Claude-SearchBot, ClaudeBot, Google-Extended) robots.txt keeps off the home page. `missing-llms-txt` (info) appears when `/llms.txt` is absent or is an HTML page. Both are inserted right after discovery, not tied to a page.
- **Why:** An assistant that cannot read the site cannot cite it. This is the first cause to rule out in "why AI does not cite you". Ubersuggest lists both checks but showed no data ([review](../../../docs/audits/2026-10-26-ubersuggest-competitor.md)).
- **Preserve:** No provider call and no credits. robots.txt is the copy discovery already fetched; `llms.txt` is one extra GET to the customer's own origin through the same SSRF-guarded `fetchFollowingRedirects` (10 s, 64 KB). A missing robots.txt blocks nothing. Blocking only training agents is reported but explained as a choice, not an error. Google AI Overviews use Googlebot, already covered by the normal crawl.
- **Trace:** `aiaccess.go`, `aiaccess_test.go` (robots cases, soft-404 HTML, content type, empty, 5xx). Each of 5 mutations fails it: nil robots, HTML prefix, content type, llms flag, a dropped crawler. Also `runner.go`, `issuetypes.go`, `frontend/src/shared/audit-issues.ts`, `IssuesView.tsx` (lists join with commas, redirect hops keep arrows). Not run against a live site yet.

## AI readiness checks: business schema, question answers, phone numbers

- **Decision:** The page analyzer now also collects, for each page:
  - its JSON-LD `@type` values (including `@graph`)
  - how many H2/H3 headings are phrased as questions
  - its `tel:` numbers (last 10 digits)

  `aiSiteSignals` gathers these during the crawl and adds three site-wide issues:
  - `missing-business-schema` (warning): the home page has no Organization or LocalBusiness-type schema. If the crawl never reached the home page, any page is judged instead.
  - `no-question-answers` (warning): no page has FAQPage schema or two or more question headings.
  - `several-phone-numbers` (info): more than one distinct number. The details list up to 5 numbers, each with the first page it appears on.
- **Why:** The owner asked for an audit that says whether a site is AEO/GEO friendly, why not, and what to fix. These are the "does AI know who you are" and "can AI quote you" parts of the approved AI readiness design. They come from HTML the crawl already fetched.
- **Preserve:**
  - No extra request, provider call or credits.
  - Phone numbers stay transient. The other signals are stored per page since Cite Fix (next entry).
  - A crawl with no 2xx HTML page reports nothing.
  - JSON-LD text never enters `BodyText` or the word count, and is capped at 256 KB per page.
  - Invalid JSON-LD counts as no schema.
  - Several phone numbers is info, not an error: real businesses can have departments.
- **Trace:** `aireadiness.go`, `pageanalyzer.go`, `types.go`, `crawler.go`, `runner.go`, `issuetypes.go`, `frontend/src/shared/audit-issues.ts`. Tests are in `aireadiness_test.go`. Each of these mutations fails a test:
  - question mark
  - phone length
  - home-page rule
  - FAQ rule
  - phone count
  - empty crawl
  - status filter
  - `@graph` walk
  - H3 headings

  Not run against a live site yet.

## Cite Fix signals stored with each page

- **Decision:** The analyzer also reads, per page: whether it names an author (meta or itemprop author, `rel=author`, JSON-LD `author`), the first publish and update dates it states (meta `article:*`, `og:updated_time`, itemprop, JSON-LD), the word count of the first paragraph of the main content (outside `nav`, `header`, `footer`, `aside`), lists and tables in the main content, blockquotes, and percent, currency and lakh/crore figures in the body. `AIPageSignals` (minus phones) is stored as `go_audit_pages.citation_signals_json` (migration 47) for every page of a new crawl. `backend/internal/citefix` scores it.
- **Why:** Cite Fix explains why AI assistants do not quote a site. These facts are in HTML the crawl already fetched, so storing them costs one text column, not a second crawl or a provider call.
- **Preserve:** No extra request or credits. Every page of a new crawl gets a non-NULL value (an empty schema list is `[]`), so NULL means "crawled before Cite Fix" and is reported as "scan again", never as zero. Dates are kept as written, at most 40 characters. JSON-LD keys are walked in sorted order, so "the first date" is the same on every crawl. A paragraph closed by the next `<p>` or a heading still counts. `noscript` content never opens or closes navigation.
- **Trace:** `aireadiness.go` (`AIPageSignals`, `jsonLDFacts`, `countStatistics`), `pageanalyzer.go`, `repository.go` (`pageArgs`, column 36), `types.go`, migration `00047_audit_citation_signals.sql`. Tests: `aireadiness_test.go` (`TestAnalyzeHTMLCollectsCitationSignals`, `TestAnalyzeHTMLCitationSignalsAbsent`, `TestAnalyzeHTMLAuthorSources`, `TestJSONLDFactsKeepsTheSameFirstDate`), `repository_helpers_test.go` (`TestPageArgsFlattensLinks`), and `citefix/store_test.go` for the round trip through Postgres. Not run against a live site yet.

## Keep paid JavaScript renders when a later batch fails

- **Decision:** `DataForSEORenderer.Render` reads each batch's raw HTML right
  after that batch (`renderBatch`). A failed later batch stops new batches but
  keeps the pages already rendered and settles only their cost.
- **Why:** review of PR #244: raw HTML was read once after all batches, so a
  failure in batch 2 of 4 returned nothing and threw away renders DataForSEO had
  already billed ("show what we paid for").
- **Preserve:** at most 20 pages, `retry:false`, one hold per audit, customer
  charge = delivered pages only; an uncertain (5xx/timeout) outcome stays
  unsettled for reconciliation but the delivered pages are still saved.
- **Trace:** `dataforseo_renderer.go`,
  `TestDataForSEORendererKeepsPaidPagesWhenALaterBatchFails` (fails on the
  previous code); `raw_html` calls are now one per batch, all free.

## Additional technical findings and isolated external checks (PR 3)

The page analyzer derives technical checks from the response it already has;
canonical/hreflang targets are checked only when actually crawled, so an
unvisited page is not labeled healthy or broken. Same-origin canonical and
hreflang targets join the existing bounded crawl queue, subject to robots rules.
Sitemap pages disallowed by
robots.txt are reported without fetching them. Search and training bot rules
are kept in one list: blocked training bots produce an informational issue,
while only search bot blocks reduce readiness scores.

External links and image sizes use a separate queue so a slow third-party host
does not hold a site-audit worker. Each audit collects at most 200 distinct
external links and 30 image targets, HEAD only for each target, with a 5-second
request limit, robots respected, a shared Redis limit of one second between
same-host requests, 64 KiB
robots cap and a 25-second batch budget. An unsuccessful or unsupported check
is partial coverage, never an invented error; Redis failure does not bypass
the host limit. A short database transaction
fences each workspace's in-flight job before any network request, with no
connection held during network waits; an ambiguous restart does not retry the
HEADs. URLs are removed from completed job payloads. Tests cover SSRF, robots,
limits, partial results, retry, rate limiting and workspace claims.
