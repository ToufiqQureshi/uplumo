# Backlink research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Serves project-scoped backlink summaries, history, referring-domain and top-page data.

## Decisions to preserve

- Validate target/filter limits and verify plan before any billed provider task. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Preserve null/unsupported metrics by target scope; never present them as zero. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Do not retry a paid task automatically; partial provider limits must be visible. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

DataForSEO paid endpoints and usage metering; project plan/credit checks.

## Show every summary/profile field the report paid for

- **Decision:** Preserve the provider's unclassified breakdown bucket (blank
  key) as `"unknown"`, and render the summary profile fields that were already
  decoded and in the API but hidden: crawled pages, link types and link
  attributes. Correct two misleading labels: the nofollow-domains tooltip now
  says "at least one nofollow link" (the provider's definition, not "only
  nofollow"), the backlinks "Link" column help and export header now say the
  rank is the backlink's own passed rank, not the linking page's authority.
- **Why:** These arrive in the same summary task that is already paid for; hiding
  them or mislabelling them loses customer value with no cost saving. The blank
  key is real paid data (links the provider could not classify), not an error.
- **Preserve:** No new provider call, no change to cache identity, and the
  unknown bucket is never presented as zero. The 6-entry per-list display cap
  and history rank/new-lost-referring-domain chart lines are deliberately left
  for a follow-up; this change only makes already-returned data visible and
  correct.
- **Trace:** `service.go` (`counts`), `frontend/src/client/features/backlinks/
  {BacklinksProfilePanel.tsx,BacklinksTableColumns.tsx,export.ts}`.

## Persist the overview so a cache miss does not re-buy

- **Decision:** Persist complete overview responses in `go_backlinks_overviews`,
  keyed by organization, API target and scope. A Redis miss can reuse a fresh
  snapshot for up to 30 days; the durable lookup runs inside the singleflight
  callback and refills Redis. Subfolder totals are complete and persist too.
- **Why:** The summary and history are separately billed provider tasks. Reusing
  a report avoids repeat spend after the 24-hour Redis TTL.
- **Preserve:** A summary-only response after history failure stays in Redis
  for ten minutes and never replaces a complete snapshot. Writes are best-effort;
  DB read/write failures fall back without failing an already-paid answer.
  The table is server-only, organization-scoped and cascades on org deletion.
- **Trace:** `service.go` (`Overview`, `loadOverviewSnapshot`),
  `00040_backlinks_overviews.sql`, `README.md`, `flow.md`,
  `service_persistence_test.go`.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Preserve bought blocks and coalesce table purchases

- **Decision:** Rows and referring domains attempt a complete small-profile
  fetch through the existing unfiltered first-block cache. `loadPage` shares
  equivalent cold misses through the service's singleflight group with a
  three-minute detached deadline and a second cache check.
- **Why:** When a cached overview undercounted a profile, the speculative
  1,000-row purchase was discarded and bought again immediately and on later
  page-size changes. The regression observed three purchases for two equivalent
  first-block views. Eight simultaneous table requests also bought eight tasks
  instead of one per organization.
- **Preserve:** Incomplete blocks never masquerade as a complete list. Full
  query, organization, kind, sort, mode and block stay in the identity; another
  view may still need its own provider task for a large profile. No automatic
  paid retry, no plan/price/cache-lifetime change, and no credits held for a
  caller cancelled before registration. Shared work can finish for active
  waiters, but separate server processes are not coalesced.
- **Trace:** `service.go`, `whole_list.go`, `table_purchase_test.go`. Tests use a
  provider-shaped local HTTP response, reproduce undercounts/concurrent misses
  before the fix, check organization isolation and cancellation, and run with
  the race detector. Post-deployment raw-response and ledger checks remain
  separate live validation.

## Cancellation before a new paid operation

- **Decision:** `Overview` rechecks cancellation after `readCache` and before `flights.DoChan`.
- **Why:** Preparation could return a cache miss/project result after the caller left, and `context.WithoutCancel` then revived it into paid work.
- **Preserve:** Active callers keep existing cache-outage fallback and tenant/cost checks. Already-registered bounded shared work can finish for other waiters; this does not add a retry, limit or pricing change.
- **Trace:** `cancellation_test.go` reproduced both summary/history tasks after cancellation; it now requires zero. Tests synchronize existing work before counting calls; see the shared [fixture decision](../platform/spend/spendtest/decision.md).

## Preserve a summary when history cannot be bought

- **Decision:** A history credit/billing refusal returns the already-paid summary with absent trends and the existing ten-minute partial cache. Remove the duplicated overview-age declaration that prevented the server from compiling.
- **Why:** Discarding a bought summary left the customer charged without usable output and made a revisit buy it again.
- **Preserve:** A refused summary buys no history; no later paid work or provider retry is introduced. Partial reports never overwrite complete durable snapshots.
- **Trace:** `service.go`, `TestOverviewKeepsTheBilledSummaryWhenCreditsCannotCoverHistory`, existing history-error and persistence regressions. One summary provider call covers the tested repeated visit.

Provider rejection fixtures include explicit zero cost and task path: missing billing metadata represents an uncertain purchase, not a confirmed free rejection. `TestProviderErrorIsNotRetried` still requires one request and the upstream error classification.

## Keep domain normalization together

- **Decision:** Define `NormalizeProjectDomain` in `target.go` with the target normalizer it delegates to; remove the single-function `project_domain.go`.
- **Why:** One validation responsibility no longer requires a separate file.
- **Preserve:** Exact hostname/IDNA/public-suffix validation, callers, errors and all paid/cached request behavior.
- **Trace:** `target.go`, existing target/project and backlink service regressions.
