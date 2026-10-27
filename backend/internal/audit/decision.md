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

## Honest rendering capability and reachable code

- **Decision:** Keep rendering unavailable until an actual renderer is wired into the crawler. Environment flags alone cannot advertise it. Reject requests before rows, jobs or credit holds. Remove unreachable renderer pricing and orphan crawl helpers, retaining the active credit lifecycle tests.
- **Why:** The HTTP crawler never executed JavaScript; accepting rendering requests misrepresented the work performed. Test-only port helpers duplicated inactive behavior.
- **Preserve:** HTTP crawl, SSRF checks, plan limits, job idempotency and credit cleanup remain intact. Existing rendering interfaces are retained for the runner lifecycle; they do not imply a usable renderer.
- **Trace:** `rendering.go`, `service_test.go` (`TestEnvironmentFlagsCannotAdvertiseAnUnwiredRenderer`, reproduced failing before the fix); `crawleraccess.go`, `crawlwindow.go`, `htmlresponse.go`, `issues_test.go`. `handler_test.go` checks the mounted capabilities/start responses and no rows/jobs; a forced-true mutation fails both service and HTTP regressions. No browser renderer/provider was added.

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
