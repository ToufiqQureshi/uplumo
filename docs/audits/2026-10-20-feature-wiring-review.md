# Feature wiring review

Scope: inspect all 29 client feature folders against Go route registration,
service construction and storage/provider boundaries. This is a source-level
wiring review, not a claim that every line or live customer integration was
exercised. Entry points are `backend/internal/httpapi/httpapi.go`,
`backend/cmd/server/main.go` and each feature's README/flow/decision triplet.

## Client to server and data map

Client folders below are under `frontend/src/client/features`; Go packages are
under `backend/internal`. PG means Postgres. Each row identifies the actual
implementation, including composed screens that need no separate backend.

| Client feature     | Go implementation / API family                       | Persistence and dependencies                                  | Finding                                                                                             |
| ------------------ | ---------------------------------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| ai-mcp             | `mcp`, OAuth, settings API keys                      | PG grants/tokens/API keys; tools delegate to services         | Setup UI, not a duplicate data backend                                                              |
| ai-search          | `aisearch`, keyword provider                         | PG tracked prompts/jobs; Redis lookup/history; paid providers | Wired; keep explicit paid actions                                                                   |
| analytics          | `analytics`                                          | PG sites/pageviews/events/browser registry; Redis salts/rates | Wired including audience/clicks on the parent branch; not deployed by this review                   |
| audit              | `audit`                                              | PG crawls/pages/issues; jobs; Redis progress/quota            | Fixed false JavaScript-rendering capability                                                         |
| auth               | `emailauth`, `auth`, `apikeys`                       | PG sessions/users/orgs; email integration                     | BetterAuth client talks to Go; no TS server to delete                                               |
| backlinks          | `backlinks`, `ahrefs`                                | Redis/provider cache; DataForSEO                              | Wired; provider/config gated                                                                        |
| billing            | `billing`, `plans`, `razorpay`                       | PG subscriptions/credits/ledger/provider spend                | Wired; live payment validation outside this review                                                  |
| branding           | `branding`                                           | PG workspace settings                                         | Wired; plan gated                                                                                   |
| content-editor     | `contenteditor`, `contentauthors`, `linksuggest`     | PG drafts/authors/briefs; optional storage; paid AI/SERP      | Wired; local score does not require paid AI                                                         |
| crawler-access     | `crawleraccess`                                      | PG encrypted crawler credentials                              | Wired; secrets stay server-side                                                                     |
| dashboard          | `dashboardoverview`, `activation`, `ga4`             | Saved PG state; explicit provider refresh                     | Runtime review caught automatic paid refresh and missing plan gate; corrected with regression tests |
| domain             | `domain`                                             | Redis cache; DataForSEO                                       | Wired; explicit paid lookup                                                                         |
| gsc                | `gsc`, `google` and search-performance               | PG Google connections/property selection                      | Composed connection/re-engagement UI                                                                |
| indexnow           | `indexnow`                                           | PG keys/submission history; Redis rate limit                  | Wired; submission is not proof of indexing                                                          |
| integrations       | `google`, `gsc`, `ga4`                               | PG OAuth grants/property mapping                              | Wired; requires authorized external accounts                                                        |
| keywords           | `keywords`                                           | Redis/provider caches; DataForSEO; PG usage                   | Wired; paid research/SERP gates remain                                                              |
| lighthouse         | `audit` Lighthouse endpoints                         | PG saved payloads; PSI/CrUX; Redis quota/cache                | Shares audit backend; no separate paid Lighthouse engine                                            |
| local-seo          | `localseo`, keyword local provider                   | Provider/usage and credit boundaries; bounded grid            | Wired; no new durable report store added by this review                                             |
| onboarding         | `onboarding`                                         | PG onboarding answers/state                                   | Wired; optional connection steps                                                                    |
| projects           | `projects`, `projectcontext`                         | PG projects/context                                           | Wired; scoped to organization                                                                       |
| rank-tracking      | `ranktracking`                                       | PG configs/runs/results/tasks; jobs/scheduler                 | Wired; approved scheduled paid work                                                                 |
| reports            | `reports`                                            | PG templates/reports/shares; Go public rendering              | Wired; public sharing is intentional                                                                |
| sam                | `sam`, registered MCP tools                          | PG sessions/messages; OpenRouter spend guard                  | Wired; per-turn lock currently assumes one server instance                                          |
| saved-keywords     | `savedkeywords`                                      | PG lists/keywords                                             | Wired; distinct from paid keyword research                                                          |
| search-performance | `gsc`, `ga4` growth                                  | PG connections/baselines/report cache; Google APIs            | Wired; not every native Google report is available via API/UI                                       |
| search-tabs        | client navigation to feature queries                 | Uses underlying feature storage                               | Client controller; no independent server expected                                                   |
| settings           | `apikeys`, `branding`, `crawleraccess`, integrations | Underlying PG feature stores                                  | Composed settings screen                                                                            |
| site-publishing    | `sitepublish`                                        | PG WordPress connections/publications                         | Wired; customer credentials/provider needed for live validation                                     |
| team               | `team`, `auth`                                       | PG org/membership/invitations; email                          | Wired; invitation delivery depends on configuration                                                 |

Provider adapters, platform packages, jobs, privacy-erasure CLI and registered
MCP tools need not have one matching sidebar page. Detailed GA4 reports exposed
through MCP/API are not thereby available as ordinary UI dimension/metric
controls. A missing provider key returning a configuration error is different
from a missing backend implementation.

## Corrected behavior

The crawler constructed by `buildAuditService` only fetches HTTP HTML. Neither
`AUDIT_BROWSER_RENDERING` nor `CONTEXT_API_KEY` wires a browser renderer.
Previously either flag could advertise availability and allow a rendering start
to enqueue an ordinary crawl. Capabilities now remain false and such starts
fail before rows/jobs/credit holds. Removed the unsupported client price
estimate and updated issue guidance. PSI's separate Google-run Lighthouse is
unaffected. See both audit decision files.

## Removed code and simplifications

A production `deadcode ./...` scan identified stale functions whose tests were
their only callers. Removed the orphan GSC date-window algorithm (the live
`performance.go` implementation remains), audit signature-expiry/window/text
helpers, a descriptor wrapper and unreachable renderer price calculations.
Tests exclusive to removed unreachable code were removed; active read,
descriptor, window, date and credit-lifecycle behavior retains tests.

MCP scope membership uses standard-library `slices.Contains`. AI search bounds
use built-in `min`/`max` with identical limits. No dependency, paid retry,
provider fan-out, plan price, cache key or retention change was introduced.
Retained test seams/compatibility constructors are not automatically deleted
merely because a production-only scan lists them: e.g. clock injection,
`spendtest` fakes and alternate service constructors.

## Verification and limits

The new service regression was run before the renderer fix and failed because
environment hints advertised the unwired renderer and accepted starts. The
corrected service and mounted HTTP regressions check capabilities and rejection
before persistence and dispatch. Temporarily forcing availability true makes
both regressions fail; the implementation was restored afterward. Mobile (390px) and desktop (1280px) browser regressions use API fixtures to check
the disabled control, unavailable explanation and absence of automatic starts
or paid work. These are UI checks, not authenticated live crawl acceptance.
Full Go race/coverage tests against real local Postgres/Redis, module/format/vet/
static/lint/vulnerability/deadcode checks, 574 frontend tests, frontend checks
and production build passed. Local integration checks use
disposable Postgres/Redis, never production data or paid provider calls.

This review does not certify every paid provider field is displayed, every
cache path is optimal, all features are bug-free, or all merged PRs are live.
Live Google reconciliation needs connected accounts; payment lifecycle needs
provider test-mode validation before live keys; WordPress/email delivery needs
configured destinations. SAM's in-memory turn lock must become distributed
before adding server replicas (documented in its README). The original source-only review did not deploy or change production settings,
secrets or the production database. Release actions are verified separately.

## Runtime review follow-up

A separate local review used the real `cmd/server` binary, its production SPA
build, disposable migrated Postgres/Redis and an authenticated local account.
Account creation/sign-in and the six-step onboarding skip/finish path used real
APIs. Email verification was seeded in the local database; no verification
email delivery or production identity bypass was tested.

The 25-route browser pass opened the main project features, integrations,
Agent setup, Billing, credits, Help and Settings. It found no uncaught page
exception in those observed flows. Search Performance returned the expected
configuration-unavailable response because Google credentials were deliberately
absent. The settings/context path redirects to the canonical context page; it
is not a missing backend. Empty states and Free-plan gates are not acceptance
proof for paid/connected reports. Screenshots and runtime logs were retained
outside the repository, without production customer data.

This runtime pass caught an issue the source matrix did not: `DashboardPage`
used a mount effect to call the otherwise explicit backlink-refresh endpoint.
Its route also omitted the normal paid Backlinks gate, simultaneous stale
reads could buy duplicate summaries, and stale refresh errors could be hidden
as success. The dashboard fix removes the effect, adds a pending-disabled
explicit action/credit guidance, shares the paid route gate, coalesces scoped
refreshes, and retains saved data while reporting failure.

Verification: the browser regression failed before the fix on the unwanted
page-open POST, then passed for visits/reload and one explicit failed action
without retries. Real-Postgres tests cover saved snapshot reuse/persistence,
eight concurrent refreshes, organization isolation and retained stale data.
Isolated mutations proved the tests reject removing the mounted plan gate
(200 instead of 402), removing coalescing (8 purchases instead of 1), and hiding
a stale refusal as success. Existing paid provider guards and the one-replica
constraint remain in force.

Native Analytics was additionally exercised through a controlled publisher
page in Chromium, the real Go tracker/collector, local Postgres, authenticated
report APIs and the actual dashboard with no mocked analytics response. It
recorded two pageviews in one session, one new recognized browser, Facebook
paid-social campaign attribution, over ten seconds of visible engagement, one
internal click with an observed arrival and one external destination. Query
parameters/fragments did not appear in the destination report. Desktop/mobile
UI rendered those real results without document overflow. This used a normal
browser user agent because synthetic headless user agents are intentionally
filtered; no production analytics traffic was fabricated.

Release review additionally caught three dashboard edge cases: older already-open
clients could still submit an empty automatic refresh request; domain changes
could join the previous domain's in-flight purchase; and a prior mutation error
could survive a project switch. Refresh now requires the explicit action marker,
coalesces by organization/project/domain, reads snapshots for the current domain,
and remounts project-local view state on project changes. Tests were observed
failing before each correction. Isolated Go overlay mutations also verify the
legacy marker, domain coalescing and current-domain SQL predicates are necessary.

Additional authenticated local checks opened Personal settings, Team membership
and Report branding against real Go APIs with no page exceptions. The personal
settings API-key flow created a temporary key through the UI, persisted its list
entry across reload, revoked it through the confirmation dialog and confirmed
its absence after reload. The raw key was not logged or committed; no invitation
email or paid branding activation was attempted.

## Editor runtime corrections

Saving and reloading a real draft originally opened an empty editor even though
the article remained in Postgres. The editor now keeps only its opaque draft ID
in the URL and reloads it through the scoped API. Missing/foreign IDs show an
error rather than a writable blank replacement; unsaved navigation is blocked.
The save/open mapping preserves author and review metadata.

A second real-backend pass exposed a contract defect permissive fixtures had
hidden: save requests sent response-only `lastReviewedAt`, whereas Go accepts
`lastReviewedDate`. The client now maps that field explicitly, and fixtures
reject the invalid request. The resulting 400 also revealed an unchanged-draft
autosave loop every three seconds; failed input now waits for a new edit or an
explicit Save draft. Red-before-fix API/browser regressions cover both defects.

The corrected production SPA and Go 1.27.3 binary were exercised with real local
Postgres: save article plus author/review date, read the persisted API result,
reload the selected URL, confirm restored text/metadata and check mobile document
width. One save, no paid calls and no browser exceptions were observed. External
Google/payment/email/WordPress acceptance remains outside this local evidence.

## Remaining production measurement check

A read-only seven-day production aggregate contained 11 pageviews and no stored
country. That small sample does not establish the cause. Railway documents
`X-Real-IP`, while the application accepts forwarded addresses only from explicit
trusted proxy CIDRs; deployed proxy trust needs direct validation before claiming
geography is reconciled. No broad trust ranges, guessed locations, raw visitor IP
logging or fabricated production traffic were introduced to fill the gap.

## Optional rank suggestions cost boundary

The tracker setup step purchased a Labs ranked-keywords task immediately after
configuration creation. It now presents an optional action/credit hint and a
free Skip/manual route; disabled queries never buy work on setup, focus or
reload. The suggestions endpoint requires the explicit-action marker so older
open bundles cannot continue automatic requests.

Eight simultaneous identical suggestions requests also bought eight tasks on
both a cold cache and an unavailable cache. They now share one bounded task
using the existing full tenant/project/target/market cache key. Independent
organizations do not join each other's work, and one abandoned waiter does not
cancel another waiter's active request. Redis retains its existing 12-hour
nonempty-result policy; no migration or paid retry was added. A single replica
remains required. Real provider HTTP fakes and browser setup/Skip/double-click
regressions failed before these corrections. Isolated overlay mutations prove
legacy-action validation, coalescing, tenant isolation and waiter cancellation
are necessary; no live paid calls were made.

Release review also reproduced preparation-cancellation purchases in five paths:
suggestions (1 task), Backlinks (2), Keyword Research (1), Brand Lookup (6) and
dashboard refresh (1). Each now rechecks the request after cache/project lookup
and before detaching paid work. Deterministic tests drain any registered flight
before counting calls; the corrected expectation is zero. Active cache-outage
fallback and already-started shared work remain intact.

Further real Go/Postgres/browser checks saved and reloaded Project Context,
created a saved keyword through the free save API, displayed it after reload,
deleted it through the UI confirmation and confirmed removal after reload.
No paid metric refresh/research was invoked and no browser exception occurred.
