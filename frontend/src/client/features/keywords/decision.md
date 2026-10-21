# Keyword research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Researches seed terms and SERP results for a project market, with location selection and keyword metrics.

## Decisions to preserve

- Research and SERP calls can be billed by the provider; reject invalid requests before spend and do not add blind POST retries. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Cache keys must include every option that changes results, project and organization scope. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.
- Show absent/null provider metrics as unavailable, not as zero. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- Location results must use canonical provider locations; do not silently fall back to a different geography. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.

## Verification pointers

- `hooks/useKeywordResearchData.test.ts`
- `groupSharedVolumeRows.test.ts`

## Paid-result delivery and repeat requests

- **Decision:** Keep successful SERPs fresh and retained for 12 hours, show the
  missing-exact-match notice in Auto too, and normalize whitespace consistently
  when selecting overview metrics. Export bid bounds, referring domains,
  quarterly change and monthly history through the existing CSV/Sheets flow.
- **Why:** Revisiting a keyword caused another request immediately; cached
  shallow snapshots also disappeared after the default five-minute retention.
  Missing seed data silently selected a related row, and exports dropped useful
  data that was already present in the response.
- **Preserve:** No paid retries on failure, focus or reconnect; project/market/
  depth query identity; selected/filtered export scope; zero versus unavailable;
  and explicit user selection of a different overview keyword. No price changes.
- **Trace:** `hooks/useKeywordSerpAnalysis.ts`, `state/useKeywordOverviewState.ts`,
  `state/keywordControllerActions.ts`, `state/keyword-delivery.test.ts` and
  `hooks/keyword-serp-cache.test.ts`. Backend seed preservation and same-depth
  coalescing are recorded in the backend keyword decision document.
