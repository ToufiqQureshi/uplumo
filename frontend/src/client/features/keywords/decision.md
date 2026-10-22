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

## Tell customers what Google shows and which place ideas cover

- **Decision:** Add a "Google shows" column built from `serpFeatures`, with
  plain-language badges and a what-to-do tooltip each, include it in CSV/Sheets
  exports, and show an info notice when `placeFilter` is set.
- **Why:** The table listed numbers without telling a small business what to do.
  Whether Google shows a map/hotel pack, an AI Overview or questions decides
  whether the answer is a Google Business Profile, a direct answer or an FAQ.
  The backend now receives this data free with the research task; dropping it
  would waste paid data.
- **Preserve:** No new request, retry or refetch for these fields; results
  cached before the change (no `serpFeatures`/`placeFilter`) still parse and
  render "-" and no notice. Badge order: Maps, AI, Snippet, Q&A, Video, at most
  three.
- **Trace:** `components/SerpFeatureBadges.tsx` and its test,
  `page/KeywordResearchTable.tsx`, `page/KeywordResearchResults.tsx`,
  `state/keywordControllerActions.ts`, `state/keyword-delivery.test.ts`. Backend
  decision: `backend/internal/keywords/decision.md`.

## Paid-result delivery and repeat requests

- **Decision:** Keep successful SERPs fresh and retained for 12 hours, show the
  missing-exact-match notice in Auto too, and normalize whitespace consistently
  when selecting overview metrics. Export bid bounds, referring domains,
  quarterly change and monthly history through the existing CSV/Sheets flow.
  SERP rows and exports retain the descriptions already decoded from live organic
  results, without another paid lookup. Live organic output does not document
  the traffic/backlink enrichment fields used by Labs, so no extra paid
  enrichment or synthetic metrics are added.
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
  `components/SerpAnalysisCard.test.ts` checks description display and export,
  using the description already present in the API response.


## Deliver paid research fields consistently

- **Decision:** Omit absent secondary intents from JSON and replace both local competition value and label with the Google Ads city-level values.
- **Why:** Ads rows have no secondary-intent list, and local competition must not be paired with the national LOW/MEDIUM/HIGH label.
- **Preserve:** Missing data stays unavailable; the schema accepts older cached rows with no secondary intents.
- **Trace:** Keyword research response schema, local result mapping, and table/export delivery.
